# Nuage

Cloud file storage for the Facile Suite.

## Stack

- `apps/api`: Go, Chi, GORM, PostgreSQL, MinIO
- `apps/client`: SvelteKit 5, Tailwind CSS 4, Bun
- `docker-compose.yml`: PostgreSQL, MinIO, API, and client services

## Quick start

### Docker

1. Copy the root env file and adjust values if needed:

```sh
cp .env.example .env
```

2. Start the full stack:

```sh
docker compose up --build
```

3. Open the app:

- Client: `http://localhost:3000`
- API: `http://localhost:4000`
- MinIO Console: `http://localhost:9001`

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

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@db:5432/nuage?sslmode=disable` |
| `DOMAINS` | Allowed frontend origins for CORS | `http://localhost:3000` |
| `PORT` | API port | `4000` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, or `error` | `info` |
| `MINIO_ENDPOINT` | MinIO server address | `minio:9000` |
| `MINIO_ACCESS_KEY` | MinIO access key | `minioadmin` |
| `MINIO_SECRET_KEY` | MinIO secret key | `minioadmin` |
| `MINIO_BUCKET` | S3 bucket name | `nuage` |
| `MINIO_USE_SSL` | Use HTTPS for MinIO | `false` |
| `OIDC_ISSUER` | OIDC provider issuer URL | — |
| `OIDC_CLIENT_ID` | OIDC client ID | — |
| `OIDC_CLIENT_SECRET` | OIDC client secret | — |
| `OIDC_REDIRECT_URL` | OIDC callback URL | — |
| `OIDC_SUCCESS_URL` | Post-login redirect | First `DOMAINS` entry |
| `SSO_ONLY` | Hide password login | `false` |
| `VITE_API_BASE_URL` | Client-side API base URL (build-time) | `http://localhost:4000` |

See [`.env.example`](.env.example) and [`apps/api/.env.example`](apps/api/.env.example) for examples.
