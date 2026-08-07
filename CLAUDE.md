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
        authcrypto/                # password hashing
        database/                  # GORM DB connection
        env/                       # config loading from env vars
        errors/                    # error types
        httpjson/                  # JSON response helpers
        logger/                    # structured logging (slog)
        middleware/                # CORS, security headers, request logging, auth
        nook/                      # webhook (Nook) notifier
        storage/                   # MinIO S3 client wrapper
      modules/                     # feature modules (handler + service + routes)
        auth/                      # login, register, sessions, OIDC, API tokens
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
        routes/
          +page.svelte             # landing / register page
          login/                   # login page
          (app)/                   # authenticated layout group
            files/                 # main file browser (grid/list, upload, preview)
            trash/                 # trash view
            shared/                # shared files view
            activity/              # activity feed
            settings/              # settings, tokens, Nook webhooks
          api/[...path]/           # reverse proxy to Go API
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
- Configuration is read through `tronc/env`: `Config` embeds `troncenv.Core`, so `DATABASE_URL`, `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY` and `MINIO_SECRET_KEY` are required and the API exits 1 without them. `CORS_ALLOWED_ORIGINS` is the canonical origin list, with `ALLOWED_ORIGINS` and the other drifted names still read as fallbacks. `PRESIGN_SECRET` is required — the API exits 1 without it.

## Gotchas

- The single Dockerfile lives at the repo root and its context is the repo root: it builds the client from `apps/client/` and the API from `apps/api/` into one image.
- Tests are integration tests that need real Postgres and MinIO running. There is no mock layer.
- The client builds with `adapter-static` and is served by the Go binary, so there is no SvelteKit server and no `BODY_SIZE_LIMIT` to raise: upload size is bounded by the API and Traefik alone.
- Rate limiting is 100 requests/minute per IP, skipped for `/api/files/upload/*` and `/webdav/*` (a single large upload issues hundreds of sequential chunk requests), plus a stricter 10/minute on `/api/auth/login` and `/api/auth/register`. The client IP is taken from the *rightmost* `X-Forwarded-For` entry, since the leftmost is caller-controlled.
- `PRESIGN_SECRET` is required: the API refuses to boot without it, because presigned download links are unauthenticated and forgeable if the signing key is guessable.
- Sync clients feed the previous response's `server_time` back as `since`. That cursor is intentionally dated slightly in the past, so a small window of changes is redelivered — clients must apply changes idempotently by `id`.
- Permanent deletions are recorded in a `tombstones` table (90-day retention) so clients whose cursor predates the purge still learn the file is gone instead of re-uploading it.
- Every space-scoped endpoint calls `spaceaccess.Require` before using a caller-supplied `space_id` as a query or write predicate.
- Production routing uses Traefik labels in docker-compose.yml (Dokploy deployment on `nuage.facile.studio`). There is **no** strip-prefix middleware: `/api` is owned by the Go router, so a local `curl` against port 4000 must include it.
