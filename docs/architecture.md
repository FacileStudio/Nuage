# Nuage — Architecture

How a request reaches the Go API, where file bytes and metadata live, and how sync,
sharing, and WebDAV are built on top.

## Runtime topology

```
                     ┌──▶ /api/*    ──┐
Internet ──▶ Traefik ┤   /webdav/*  ──┼──▶ Go API (:4000) ──┬──▶ Postgres 16
                     └──▶ everything ─┘   serves the SPA    ├──▶ MinIO (bucket)
                                                            └──▶ /app/data/avatars
```

One container, one hostname. Traefik routes `Host(nuage.facile.studio)` to the API and
nothing else; the Go binary owns `/api` inside its own router, serves `/webdav` at the root
because external clients such as Finder depend on that URL, and mounts the built SvelteKit
bundle last as the catch-all through tronc's `spa` package. Postgres and MinIO are internal —
they only `expose` their ports on the compose network.

Nuage used to be the visible exception to the suite's one-container rule, running a separate
`adapter-node` SvelteKit server and a strip-prefix middleware. That is gone: the client builds
with `adapter-static`, there is no SvelteKit server, no `/api/[...path]` proxy and no
`hooks.server.ts`. There is exactly one route from the browser to the API.

## Components

| Component | Path | Role |
|---|---|---|
| API | `apps/api` | Chi router, feature modules, migrations, WebDAV, Nook notifier |
| Client | `apps/client` | SvelteKit UI, built static and served by the API binary |
| Postgres | compose service | All metadata |
| MinIO | compose service | File bytes and chunk parts |

The API is a single Go module, `github.com/FacileStudio/Nuage/apps/api`. Feature modules
under `modules/` each expose `RegisterRoutes` and split into handler, service, and types;
shared plumbing lives in `internal/`.

## Request lifecycle

The middleware stack, in the order `main.go` installs it:

1. `chimiddleware.RequestID`
2. `middleware.RealIP` — reads the **rightmost** `X-Forwarded-For` entry, since the leftmost
   is caller-controlled and would let anyone forge an IP
3. `middleware.CORS(AllowedOrigins)`
4. `middleware.SecurityHeaders`
5. `middleware.RequestLogger`
6. `chimiddleware.Recoverer`
7. `middleware.RateLimitExcept(100, time.Minute, "/files/upload", "/webdav")`
8. `middleware.RateLimitPaths(10, time.Minute, "/auth/login", "/auth/register")`

Upload and WebDAV paths are exempt from the general limit because one large transfer issues
hundreds of sequential chunk requests and would otherwise rate-limit itself.

`/health` and `/ready` are registered before everything else. `/ready` pings Postgres and
calls `EnsureBucket` on MinIO with a 2-second budget, returning `503` with a `reason` of
`database` or `storage`.

The HTTP server deliberately sets no whole-request read or write deadline. A multi-gigabyte
transfer legitimately outlives any fixed budget, and exceeding one truncates the response
after `Content-Length` has already promised more. Slow-header and idle attacks are bounded
by `ReadHeaderTimeout` of 5s, `IdleTimeout` of 120s, and a 1 MiB header cap instead.

## Authentication

`middleware.RequireAuth` accepts an `Authorization` header, and falls back to a `token`
query parameter when the header is absent — which is what makes a plain `<img>` or download
link work. `middleware.RequireAdmin` sits on top for admin-only routes.

Three credentials exist:

- **Session tokens** from `POST /auth/login` or `/auth/register`, stored in `sessions`.
- **API tokens** from `POST /users/me/api-token`, stored in `api_tokens`. These are also the
  WebDAV password: `requireBasicAuth` ignores the Basic username entirely and authenticates
  on the password alone.
- **OIDC**, enabled when `OIDC_ISSUER` is set. `GET /auth/oidc` starts the flow,
  `/auth/oidc/callback` completes it and upserts the user, and `POST /auth/sync-profile`
  refreshes the profile. `SSO_ONLY=true` removes `/auth/register` and `/auth/login` from the
  router entirely rather than merely hiding them in the UI.

Nuage is one of the six Go backends federating to Authentik at `porte.facile.studio` through
`go-oidc/v3`. OIDC remains additive: the local user table stays authoritative, and the
callback upserts into it.

## Storage

File bytes live in MinIO under a bucket key; metadata lives in `files`. Avatars are the
exception — they are written to `STORAGE_DIR/avatars` on a local volume and served by a
`http.FileServer` mounted at `/avatars/*` with a one-day immutable cache header.

**Avatar URLs are persisted with the public prefix baked in.** Both the upload path and the
OIDC avatar importer store `avatar_url` as `/api/avatars/<filename>`, so every existing row
assumes the API is publicly mounted under `/api`. Changing that mount — for instance moving
Nuage to the suite's one-container shape where the Go router owns `/api` itself — has to
either preserve the same public path or migrate the column. Deletion already compensates by
trimming a leading `/api/` or `/files/` before resolving the file on disk, and refuses to
unlink anything outside `STORAGE_DIR/avatars`.

## Data model

| Table | What it holds |
|---|---|
| `users` | Email, name, `password_hash`, `is_admin`, color, avatar fields, OIDC tokens and `profile_synced_at` |
| `sessions` | Session token as primary key, `user_id`, `expires_at` |
| `api_tokens` | Named long-lived tokens, unique `token`, `user_id` |
| `spaces`, `space_members` | Shared containers and their membership roles, unique on `(space_id, user_id)` |
| `files` | `facile_id`, name, mime, size, hash, `bucket_key`, `folder_id`, `space_id`, `uploaded_by`, `deleted_at`, `origin_app`, `linked_to` |
| `folders` | `facile_id`, name, `parent_id`, `owner_id`, `space_id`, `deleted_at` |
| `file_versions` | One row per superseded upload: `file_id`, `version`, `bucket_key`, hash, size |
| `upload_sessions`, `upload_chunks` | Resumable uploads, chunks unique on `(session_id, part_number)` |
| `shares` | Public token, target file or folder, `permission`, optional `expires_at` |
| `user_quotas` | `storage_used` and `storage_limit` per user |
| `activity_logs` | Event type, resource type and id, JSON metadata |
| `nook_deliveries` | Outbound event queue with status, attempts, `next_retry_at`, response code |
| `tombstones` | Permanent deletions kept for `TombstoneRetention`, 90 days |
| `settings` | Key-value application settings |

`files` and `folders` carry a `facile_id`, the suite-wide external identifier, alongside the
numeric primary key.

`schemas.Migrate` wraps everything in one transaction holding
`pg_advisory_xact_lock(4919)`, so concurrently starting instances serialize instead of
racing. Inside it, `preMigrate` runs hand-written DDL that GORM cannot express — repairing
the `api_tokens` primary key when it is still the token column, and dropping the legacy
`shares.shared_with` column — then `AutoMigrate` over the sixteen models, then
`ensureAdmin`, which promotes the earliest account when no administrator exists and does
nothing once one does. Afterwards `usercolor.BackfillMissing` fills in absent user colors.

## Sync

`GET /sync/state` returns the full picture; `GET /sync/changes` returns everything since a
cursor. Clients feed the previous response's `server_time` back as `since`. That cursor is
deliberately dated slightly in the past, so a small window of changes is redelivered and
clients must apply changes idempotently by `id`.

Permanent deletions would otherwise be invisible to an offline client, which would then
re-upload the file it thinks it is missing. `tombstones` records them for 90 days; a client
whose cursor is older than that must resynchronize from `/sync/state`. A pruner goroutine
drops expired markers every 12 hours.

## Cross-app integration

`internal/nook` queues outbound events in `nook_delivery` and retries them in the
background, which is what `POST /settings/test-nook` exercises and
`GET /settings/nook/deliveries` inspects. Logging goes to Journal: when both `JOURNAL_URL`
and `JOURNAL_TOKEN` are set, `main.go` wraps the default `slog` handler with
`journal.NewHandler`, so everything the app already logs is teed to the central instance.
`JOURNAL_URL` must include the `/api` suffix or every line is silently dropped.
