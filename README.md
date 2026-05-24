# Nuage

Cloud file storage for the Facile Suite.

## Architecture

Single public endpoint: the SvelteKit client handles all user traffic and proxies `/api/*` requests to the Go API internally. Postgres and MinIO are internal Docker services with hardcoded credentials — no configuration needed.

```
Internet → SvelteKit (:3000) → Go API (:4000) → Postgres / MinIO
```

## Stack

- `apps/api`: Go, Chi, GORM, PostgreSQL, MinIO
- `apps/client`: SvelteKit 5, Tailwind CSS 4, Bun
- `docker-compose.yml`: PostgreSQL, MinIO, API, and client services

## Quick start

### Docker

```sh
cp .env.example .env
docker compose up --build
```

Open `http://localhost:3000`.

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

The client defaults to `http://localhost:5173` and talks to `http://localhost:4000`.

## Configuration

Only external-facing variables need configuration. Internal services (Postgres, MinIO) use hardcoded defaults inside Docker.

| Variable | Description | Default |
|---|---|---|
| `ORIGIN` | Public URL of the SvelteKit app (needed for CSRF) | `http://localhost:3000` |
| `DOMAINS` | Allowed frontend origins for CORS | `http://localhost:3000` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | `info` |
| `OIDC_ISSUER` | OIDC provider issuer URL | — |
| `OIDC_CLIENT_ID` | OIDC client ID | — |
| `OIDC_CLIENT_SECRET` | OIDC client secret | — |
| `OIDC_REDIRECT_URL` | OIDC callback URL (e.g. `https://nuage.example.com/api/auth/oidc/callback`) | — |
| `OIDC_SUCCESS_URL` | Post-login redirect | — |
| `SSO_ONLY` | Hide password login | `false` |

See [`.env.example`](.env.example) for a production-ready template.
