# Nuage

Cloud file storage for the Facile Suite. Self-hosted, Docker-deployed, with a Go API backend and SvelteKit frontend.

## Tech Stack

| Layer | Tech |
|-------|------|
| API | Go 1.25, Chi router, GORM, PostgreSQL 16, MinIO (S3-compatible) |
| Client | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, Bun, adapter-static |
| Auth | Session cookies, OIDC/SSO (optional), API tokens |
| Infra | Docker Compose, Traefik (production), Dokploy |
| Tests | Go `testing` + `testify` (integration tests against real DB + MinIO) |

## Key Commands

### Docker (full stack)

```sh
cp .env.example .env               # then set PRESIGN_SECRET
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build   # one container on localhost:4000
```

### Local Development

```sh
# 1. Start backing services (ports published on 127.0.0.1 via docker-compose.dev.yml)
docker compose -f docker-compose.yml -f docker-compose.dev.yml up nuage-db nuage-minio -d

# 2. API (port 4000)
cd apps/api
cp .env.example .env
go run .

# 3. Client (port 5173)
cd apps/client
bun install
bun run dev
```

### API Tests

Tests require a running PostgreSQL and MinIO (the `docker compose -f docker-compose.yml -f docker-compose.dev.yml up nuage-db nuage-minio -d` from above).

```sh
cd apps/api
go test ./tests/ -v
```

### Client

```sh
cd apps/client
bun run build                      # production build
bun run check                      # svelte-check + TypeScript
```

## Project Structure

```
Nuage/
  docker-compose.yml               # full stack: db, minio, api, client
  docker-compose.dev.yml           # publishes ports on 127.0.0.1 for local dev (opt-in via -f)
  .env.example                     # production env template
  apps/
    api/
      main.go                      # entrypoint, router setup, service wiring
      internal/                    # shared packages
        activity/                  # activity logging
        authcontext/               # request-scoped auth context
        authcrypto/                # legacy argon2 helpers (no password sign-in any more)
        database/                  # GORM DB connection
        env/                       # config loading from env vars
        errors/                    # error types
        httpjson/                  # JSON response helpers
        logger/                    # structured logging (slog)
        middleware/                # CORS, security headers, request logging, auth
        nook/                      # webhook (Nook) notifier
        storage/                   # MinIO S3 client wrapper
      modules/                     # feature modules (handler + service + routes)
        auth/                      # porte mount (sessions, OIDC), the UserStore that links Registre sign-ins, token scopes
        files/                     # upload, download, versions, chunked uploads
        trash/                     # soft delete, restore, permanent delete
        sharing/                   # public share links, permissions
        sync/                      # full state + incremental change endpoints
        quota/                     # per-user storage quota
        search/                    # file/folder search
        activity/                  # activity log endpoints
        settings/                  # app settings, Nook webhook config
        users/                     # user management, avatars
        webdav/                    # WebDAV server (Basic Auth with API tokens)
        docs/                      # API documentation endpoint
      schemas/                     # GORM models + auto-migration
      tests/                       # integration tests
    client/
      src/
        lib/backend.ts             # typed API client (fetch wrapper)
        lib/icons.ts               # muse icons + Nuage's own glyphs, fileIcon()
        lib/format.ts              # formatSize / formatDate / relativeTime
        lib/files/                 # file-browser logic: selection, uploads, dnd, tree
        lib/components/files/      # file-browser views: grid, table, preview, share
        routes/
          +page.svelte             # landing (muse LandingHeader and LandingFooter)
          login/                   # login page
          (app)/                   # authenticated layout group
            files/                 # main file browser (grid/list, upload, preview)
            trash/                 # trash view
            shared/                # shared files view
            activity/              # activity feed
            spaces/                # team spaces, members, space settings
            settings/[[section]]/  # settings; the section is a real route
          s/[token]/               # public share viewer
          docs/                    # API docs page
      static/                      # favicon, logo, fonts, PDF worker
```

## Architecture

```
Internet --> Go API (:4000, serves the SPA) --> PostgreSQL / MinIO
```

One container. The Go binary registers the application modules under `/api`, WebDAV at
`/webdav` (external clients such as Finder depend on that URL, so it must never move under
`/api`), `/health` at the root, and mounts the built SvelteKit client last as the catch-all
via tronc's `spa` package. PostgreSQL and MinIO are internal Docker services with hardcoded
credentials.

## Conventions

- API modules follow a consistent pattern: each module directory contains a handler, service, and route registration function (`RegisterRoutes`).
- GORM models live in `apps/api/schemas/` with auto-migration in `schemas/migrate.go`.
- The client uses Svelte 5 runes (`$state`, `$props`, `$derived`, `$effect`) with TypeScript enabled.
- All API calls from the client go through `src/lib/backend.ts`.
- **The UI is built from `@facile/muse`, and its `CHARTE.md` is the contract.** Reach for a muse
  component before writing markup: `Button`, `Card`, `Table`, `Modal`, `ConfirmModal`, `Drawer`,
  `EmptyState`, `SettingsSection`/`SettingsRow`, `SecretField`, `Toaster`. Style with `fc-*`
  token utilities only — no raw hex, and no stock Tailwind palette classes (`text-amber-600`,
  `bg-emerald-100`); colour is reserved for the status tones and the role pills.
- Feedback goes through muse's `toast`, never an inline status string. A destructive action
  routes through `ConfirmModal` whose description says what actually breaks.
- Icons come from `src/lib/icons.ts` — `icons` re-exported from muse plus Nuage's own `nuage`
  map. Do not inline a `solar:*` string at a call site.
- Configuration is read through `tronc/env`: `Config` embeds `troncenv.Core`, so `DATABASE_URL`, `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY` and `MINIO_SECRET_KEY` are required and the API exits 1 without them. `CORS_ALLOWED_ORIGINS` is the canonical origin list, with `ALLOWED_ORIGINS` and the other drifted names still read as fallbacks. `PRESIGN_SECRET` is required — the API exits 1 without it.

## Gotchas

- The single Dockerfile lives at the repo root and its context is the repo root: it builds the client from `apps/client/` and the API from `apps/api/` into one image.
- Tests are integration tests that need real Postgres and MinIO running. There is no mock layer.
- The client builds with `adapter-static` and is served by the Go binary, so there is no SvelteKit server and no `BODY_SIZE_LIMIT` to raise: upload size is bounded by the API and Traefik alone.
- Rate limiting is 600 requests/minute per IP on `/api/*` only, skipped for transfer paths (`middleware.IsTransferPath`: chunk uploads, any `/download`, presigned links), SPA assets and WebDAV are never counted: a grid of thumbnails or a PDF read as range requests used to exhaust the old 100/minute budget in one page load. The client IP is taken from the *rightmost* `X-Forwarded-For` entry, since the leftmost is caller-controlled.
- Moving an item to the root sends `folder_id: 0` / `parent_id: 0`; a missing or null value means "unchanged". Use `backend.moveItem`. A folder cannot move into its own subtree (REST and WebDAV), and the target must sit in the item's space.
- `PRESIGN_SECRET` is required: the API refuses to boot without it, because presigned download links are unauthenticated and forgeable if the signing key is guessable.
- Sync clients feed the previous response's `server_time` back as `since`. That cursor is intentionally dated slightly in the past, so a small window of changes is redelivered — clients must apply changes idempotently by `id`.
- Permanent deletions are recorded in a `tombstones` table (90-day retention) so clients whose cursor predates the purge still learn the file is gone instead of re-uploading it.
- **The trash expires.** `startMaintenance` runs hourly and calls `trash.Service.PurgeExpired`,
  which deletes for good every trashed file and folder whose `deleted_at` is older than the
  `trash_retention_days` setting (default 30, `0` = never, admin-only via `PUT /api/settings`,
  max 3650). It reuses `purgeFilesTx`/`purgeFoldersTx`, so tombstones are recorded and stored
  objects removed. Quota is refunded per *uploader* (`files.uploaded_by`), not to whoever trashed
  the file, and versions are refunded to that same uploader. Batches of 500 run under a Postgres
  advisory lock with `FOR UPDATE`, so two instances cannot double-refund. A retention change
  applies to items already in the trash: lowering it purges them on the next hourly run.
  `GET /api/trash` returns `retention_days` and a per-item `expires_at`.
- **Sign-in is Registre only.** `/login` is muse `AuthScreen` and every "Sign in with Facile" button is `<SignInWithFacile />`. There are no `/auth/register` or `/auth/login` routes, no password UI and no `SSO_ONLY`: with an issuer configured porte mounts no password route. People keep their API tokens (CLI, daemon, WebDAV). `PATCH /users/me` edits name and colour only; the address is Registre's and refreshes at each sign-in. `auth.UserStore.UpsertFromOIDC` applies porte's linking rules (verified email only, refuse when the account already holds another subject of the issuer).
- **GDPR erasure.** `internal/erasure` wires `directory.Deletions`: it polls Registre and takes the signed push on `POST /api/registre/deletions` (root router, outside session and CSRF). `ErasePerson` purges personal files and folders (`trash.Service.ErasePersonal`), app folders (`appfolders.Service.DeleteAll`), then in one transaction reassigns space authorship (files, versions, folders, tombstones, upload sessions, activity) to the `deleted-user@nuage.invalid` placeholder, drops shares, quota, charges and the account. The placeholder is hidden from `GET /users`. State lives in `directory_deletion_state`. Idempotent: an unknown subject is a no-op.
- **i18n.** The UI is in fr, en, es and de through typed rows (`src/lib/i18n/messages/*.ts`, `[en, fr, es, de]`), `t()` from `$lib/i18n/index.svelte` and `provideLocale` in the root layout. The locale comes from the Registre profile (`users.locale`, filled from the `locale` claim), then the `fc_locale` cookie, then the browser. `bun run test` also runs `scripts/check-messages.ts`. API error messages stay English. Dates, sizes and relative times use Intl with the session locale.
- **No borders.** The owner rule: no `border-*`, `divide-*` or `ring-*` on controls, cards or dividers. Use `bg-fc-field`, `bg-fc-surface` and spacing. Muse components are never restyled locally.
- **One bucket per space.** A space's files, versions and uploads live in `nuage-space-<facile_id>`; personal files stay in the shared bucket. The `bucket` column on `files` and `file_versions` records where an object lives, and an empty value means the shared bucket. Every write goes through `spacebucket.Open`, and `MigrateBuckets` moves legacy space files at boot. Deleting a space is refused (409) while any file row, trashed ones included, still points at it.
- **App folders replace app-provisioned spaces.** `modules/appfolders`: tables `registered_apps` (seeded from `NUAGE_APPS` by `appregistry.Sync` at boot, never from a token label), `app_folders` (unique `(user_id, app_id)`) and `app_objects` (unique `(folder_id, key)`, bytes in the shared bucket under `apps/<user>/<app>/<fac_id>`). The only way to act as an app is a delegated Registre JWT (porte SessionID 0) whose `Actor` (act.sub) is an active registered client id: `middleware.RequireAuth` resolves it, puts it in `appregistry.With`, and then allows that JWT on `/app/*` only, and nothing else on `/app/*`. App bytes join personal usage in `quota.usageOf` (`appUsageSQL`), the legacy counter, `RecalculateUsage` and the boot over-plan report. Put lock order: folder `KEY SHARE`, per-key advisory lock, plan lock last; deleting an app's data takes the folder `FOR UPDATE`. `DELETE /app-data/{app}` is consent, refused to delegated JWTs in `managesCredentials` (an unrestricted personal API token still counts as the person). Actors are deny-by-default: a JWT whose actor is neither an active registered app nor in `NUAGE_FULL_ACCESS_ACTORS` (compose default `bulle`) gets 403 on every route (`middleware.actingApp`). WebDAV is the one route outside `RequireAuth`; `auth.AuthenticateRequest` refuses every JWT (SessionID 0) there. The migration skips files whose `origin_app` differs from the space's app and only reads `registered_apps`. The rate limiter still counts `/api/app/` per IP, so a busy app server shares one 600/min budget.
- **App storage cutover order.** 1. deploy app folders with `NUAGE_APPS` set (and Registre `token_exchange` entries giving Plume and Ardoise audience Nuage with `nuage.read`/`nuage.write`); 2. switch the Plume and Ardoise clients to `/api/app/object` with exchanged tokens; 3. run `nuage-api migrate-app-folders` (dry run), read the report, then `--apply`, rerun until only skips remain; 4. retire `PUT /spaces/by-facile-id` and provisioning for restricted tokens, then remove the emptied spaces. Never run step 3 before step 2: an old client would recreate files in the space.
- **Cross-app contract.** Other Facile apps (Ardoise, Opus attachments) store files through the REST API with a restricted personal API token, never a forwarded session. `POST /api/users/me/api-token` takes `app` plus `space_facile_ids`, `provision: true`, or both (`app` is required for any restriction); the scope lives in `api_token_scopes`, keyed on the porte session. A restricted token is allowlisted to `/files` (upload, list, get, download, presign, delete, chunked upload) and `PUT /spaces/by-facile-id/{facile_id}`, cannot touch personal files, cannot replace or version an existing file, is refused by WebDAV, and stamps `origin_app` with its `app` whatever the request says. The scope is enforced in `tokenscope.Permits` (routes), `spaceaccess.Require`/`MemberIDs` (spaces) and the files service (personal scope, id lookups).
- **Provision scope.** A token with `provision: true` may create a space for any well-formed `fac_` id through the find-or-create and then reaches every space whose `spaces.provisioned_by` equals its `app`, plus its listed ids. `provisioned_by` is set to the token's app when a restricted token creates the space and stays empty for user-created spaces, so a provision token never reaches spaces made by users or other apps, even knowing their id. The flag is `api_token_scopes.provision`; `ScopeForSession` resolves the space ids on every request.
- **Idempotent publish.** `POST /api/files` with a restricted token and a `linked_to` form field (at most 200 printable characters) returns the existing live file of the same space, `origin_app` and `linked_to` with 200 instead of storing a copy (201 for a fresh upload). The check and insert run under a per-key advisory lock, and the loser deletes its just-uploaded object. There is no unique index on purpose: `linked_to` legitimately repeats for unrestricted callers (many attachments per task). The chunked path honours the same key: `POST /api/files/upload/init` accepts `linked_to` (restricted tokens only, ignored otherwise) and stores it on `upload_sessions.linked_to`. If the key already exists, init answers 200 with `existing_file` and no session, so a retry never re-uploads. At completion the same rule runs under the same advisory lock (`savePublished`): the loser deletes its assembled object, finishes its session and gets the existing file with 200, with no quota change.
- **Lookup by key.** `GET /api/files?space_id=N&linked_to=KEY` works for a restricted token inside its reachable spaces and is refused (403) outside them. It does not filter on `origin_app`, so one app can find a file another app published in a space both may reach.
- `PUT /api/spaces/by-facile-id/{facile_id}` is the idempotent find-or-create: `fac_` plus 24 hex, `INSERT ... ON CONFLICT DO NOTHING` on the unique `facile_id`, the caller becomes owner only when the call created the row, an existing space the caller may not use (non-member, or outside the token's scope) answers the same 404 `space not found` as a missing one, so the error does not confirm an id is taken. The bucket name always derives from the id, never from the request.
- A file upload whose exact name and content hash already exist live in the same folder or space returns that file (200) and stores nothing, instead of a `(1)` copy. `saveNewFile` decides it under the folder's name lock, so racing sync uploads of the same bytes collapse into one file. Same name with different content still gets a `(1)` name.
- Find-or-create trusts the caller with an unguessable `facile_id` (96 random bits), so anyone who learns a real suite space id can claim it first, and a caller who is allowed to create can still tell a free id (201) from a taken one (404). Closing that needs suite-level proof of who may provision an id, which does not exist yet.
- **Registre owns shared spaces.** With OIDC configured, `spaces.Registre.Run` imports every
  unlinked user-made space (`provisioned_by = ''`) into Registre once, retrying with backoff
  up to 5 minutes, and only then starts `directory.Syncer.Run` (boot, every 5 minutes, and
  2 s after a nudge on `POST /api/registre/sync`, mounted on the root router outside every
  auth middleware). `Apply` runs in one transaction: link by `registre_space_id`, else the
  oldest unlinked user-made space with the same `lower(name)`, else create; members resolve
  by porte identity, then email, then a pre-provisioned user plus identity row. Removals only
  touch members holding an identity under the issuer, so accounts that never signed in through Registre stay.
  Provisioned spaces (Plume, Ardoise publishing targets) are never imported or linked. A linked
  space answers 409 to `POST /spaces`, rename, delete, member changes and leave; its
  description, folders and files stay Nuage's.
- Every space-scoped endpoint calls `spaceaccess.Require` before using a caller-supplied `space_id` as a query or write predicate.
- Production routing uses Traefik labels in docker-compose.yml (Dokploy deployment on `nuage.facile.studio`). There is **no** strip-prefix middleware: `/api` is owned by the Go router, so a local `curl` against port 4000 must include it.
