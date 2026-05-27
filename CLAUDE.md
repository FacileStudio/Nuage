# Nuage

Cloud file storage for the Facile Suite. Self-hosted, Docker-deployed, with a Go API backend and SvelteKit frontend.

## Tech Stack

| Layer | Tech |
|-------|------|
| API | Go 1.24, Chi router, GORM, PostgreSQL 16, MinIO (S3-compatible) |
| Client | SvelteKit 5 (Svelte 5 runes), Tailwind CSS 4, Bun, adapter-node |
| Auth | Session cookies, OIDC/SSO (optional), API tokens |
| Infra | Docker Compose, Traefik (production), Dokploy |
| Tests | Go `testing` + `testify` (integration tests against real DB + MinIO) |

## Key Commands

### Docker (full stack)

```sh
cp .env.example .env
docker compose up --build          # all services on localhost:3000
```

### Local Development

```sh
# 1. Start backing services
docker compose up db minio -d

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

Tests require a running PostgreSQL and MinIO (the `docker compose up db minio -d` from above).

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
  docker-compose.override.yml      # exposes ports for local dev
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
Internet --> SvelteKit (:3000) --> Go API (:4000) --> PostgreSQL / MinIO
```

The SvelteKit client is the only public endpoint. It reverse-proxies `/api/*` requests to the Go API internally. PostgreSQL and MinIO are internal Docker services with hardcoded credentials.

## Conventions

- API modules follow a consistent pattern: each module directory contains a handler, service, and route registration function (`RegisterRoutes`).
- GORM models live in `apps/api/schemas/` with auto-migration in `schemas/migrate.go`.
- The client uses Svelte 5 runes (`$state`, `$props`, `$derived`, `$effect`) with TypeScript enabled.
- All API calls from the client go through `src/lib/backend.ts`.
- Environment variables: internal services (Postgres, MinIO) use hardcoded defaults inside Docker. Only external-facing vars (ORIGIN, OIDC, LOG_LEVEL) need configuration.

## Gotchas

- The API Dockerfile context is the repo root (not `apps/api/`) because it copies from `apps/api/`. The client Dockerfile context is `apps/client/`.
- Tests are integration tests that need real Postgres and MinIO running. There is no mock layer.
- The client has `BODY_SIZE_LIMIT=Infinity` to support large file uploads via chunked transfer.
- Rate limiting is set at 100 requests/minute per IP on the API.
- Production routing uses Traefik labels in docker-compose.yml (Dokploy deployment on `nuage.facile.studio`). The `/api` prefix is stripped by Traefik before hitting the Go API.
