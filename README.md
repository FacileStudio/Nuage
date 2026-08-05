# Nuage

Cloud file storage for the Facile Suite.

## Architecture

Single container, single public endpoint: the Go binary serves `/api/*`, `/webdav/*` and the
static SvelteKit build. Postgres and MinIO are internal Docker services with hardcoded
credentials — no configuration needed.

```
Internet → Go API (:4000, serves the SPA) → Postgres / MinIO
```

## Stack

- `apps/api`: Go, Chi, GORM, PostgreSQL, MinIO
- `apps/client`: SvelteKit 5, Tailwind CSS 4, Bun, adapter-static
- `Dockerfile`: builds the client, then the API, into one image
- `docker-compose.yml`: PostgreSQL, MinIO and the API service

## Quick start

### Docker

```sh
cp .env.example .env
docker compose up --build
```

Open `http://localhost:4000`.

Postgres and MinIO are internal services with fixed credentials — there is nothing to configure for them.

### Local development

1. Start PostgreSQL and MinIO:

```sh
docker compose up db minio -d
```

2. Start the API:

```sh
cd apps/api
cp .env.example .env
go run .
```

3. Start the client in another terminal:

```sh
cd apps/client
bun install
bun run dev
```

The client serves `http://localhost:5173` and the Vite dev server proxies `/api` and
`/webdav` to the API on `http://localhost:4000`.

## Configuration

Only external-facing variables need configuration. Internal services (Postgres, MinIO) use hardcoded defaults inside Docker.

| Variable | Description | Default |
|---|---|---|
| `ALLOWED_ORIGINS` | Allowed frontend origins for CORS | — |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | `info` |
| `OIDC_ISSUER` | OIDC provider issuer URL | — |
| `OIDC_CLIENT_ID` | OIDC client ID | — |
| `OIDC_CLIENT_SECRET` | OIDC client secret | — |
| `OIDC_REDIRECT_URL` | OIDC callback URL (e.g. `https://nuage.example.com/api/auth/oidc/callback`) | — |
| `OIDC_SUCCESS_URL` | Post-login redirect | — |
| `SSO_ONLY` | Hide password login | `false` |

See [`.env.example`](.env.example) for a production-ready template.
