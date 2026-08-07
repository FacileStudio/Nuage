# Nuage — Development

Local setup, the integration test suite and what it needs, and the checks CI runs.

## Prerequisites

| Tool | Version | Why |
|---|---|---|
| Go | 1.24 | `go 1.24.0` in `apps/api/go.mod`; the image builds on `golang:1.24.9-alpine` |
| Bun | 1.3 | Client install, dev server, type-check, build, and the client tests |
| Docker | any recent | Postgres and MinIO, and the full-stack compose run |

There is no `mise.toml` and no `scripts/check.sh` in this repo — unlike most of the Go
family, Nuage's gate is the GitHub Actions workflow, and locally you run the commands
directly.

## Setup

```sh
cp .env.example .env
```

Fill in `POSTGRES_PASSWORD`, `MINIO_SECRET_KEY`, and `PRESIGN_SECRET`. Generate each with
`openssl rand -base64 36`. Compose aborts rather than starting on a default credential.

## Running

Start the backing services:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up nuage-db nuage-minio -d
```

The dev overlay is what publishes ports, all bound to `127.0.0.1`: `5432` for Postgres,
`9000` and `9001` for MinIO, `4000` for the API. There is no client port — the API serves
the built client itself.

Then the API:

```sh
cd apps/api
cp .env.example .env
go run .
```

`apps/api/.env.example` points at `localhost` for both Postgres and MinIO, so make its
credentials match the ones you put in the root `.env`.

Then the client:

```sh
cd apps/client
bun install
bun run dev
```

Vite serves on `5173` and proxies `/api` and `/webdav` to the API on `4000`. The API's
`ALLOWED_ORIGINS` example already lists `5173` on `localhost` and `127.0.0.1`.

To run everything in containers instead:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

## Tests

`apps/api/tests/` is an integration suite against a real Postgres and a real MinIO. There is
no mock layer, which is deliberate: the sync cursor, quota accounting, chunked uploads, and
WebDAV semantics are exactly the things a mock would get wrong.

```sh
cd apps/api
TEST_DATABASE_URL=postgres://nuage:<password>@localhost:5432/nuage_test?sslmode=disable \
TEST_MINIO_ENDPOINT=localhost:9000 \
TEST_MINIO_ACCESS_KEY=nuage-minio \
TEST_MINIO_SECRET_KEY=<secret> \
go test ./tests/ -count=1
```

When a `TEST_*` variable is missing the suite skips — unless `CI` is set, in which case it
calls `t.Fatalf` instead. A green CI run can therefore never mean "the infrastructure was
absent so nothing ran".

The suite covers auth, authorization, files, folders, chunked uploads, presigned links,
quota, search, shares, sync, trash, versioning, activity, and WebDAV.

Client checks:

```sh
cd apps/client
bun run check     # svelte-kit sync + svelte-check against tsconfig.json
bun test src      # the client unit tests
bun run build     # production build
```

## Continuous integration

`.github/workflows/ci.yml` runs on pushes to `main` and on every pull request. It brings up
a `postgres:16.11-alpine` service container, then starts MinIO with `docker run` rather than
as a service container — GitHub service containers cannot pass a `command`, and MinIO needs
`server /data`. It waits on `/minio/health/live`, then runs `go build ./...`,
`go vet ./...`, and `go test ./tests/ -count=1` with `CI=true`.

## Conventions

Each `apps/api/modules/<name>/` exposes a `RegisterRoutes` function and splits into
`handler.go`, `service.go`, and `types.go`. Shared plumbing lives in `apps/api/internal/`.
GORM models live in `apps/api/schemas/`, with `schemas/migrate.go` owning both the
`AutoMigrate` call and the hand-written DDL around it.

The client is Svelte 5 runes only — `$state`, `$props`, `$derived`, `$effect` — with
TypeScript, and every API call goes through `src/lib/backend.ts`.

## Things that will bite you

- **Two Docker build contexts.** `apps/api/Dockerfile` builds from the repo root but only
  copies `apps/api`; `apps/client/Dockerfile` builds from `apps/client`. Each directory has
  its own `.dockerignore`.
- **Rate limits during manual testing.** 100 requests per minute per IP overall, and 10 per
  minute on `/auth/login` and `/auth/register`. Upload and WebDAV paths are exempt.
- **The sync cursor redelivers.** `server_time` is dated slightly in the past on purpose, so
  a client must apply changes idempotently by `id`. A test that asserts an exact change
  count over consecutive polls will flake.
- **`ensureAdmin` runs on every migration.** On a fresh database the earliest account is
  promoted to administrator, and once an administrator exists nothing happens.
