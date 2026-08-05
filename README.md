# Nuage

Self-hosted cloud file storage for the Facile Suite. Go API, SvelteKit client, PostgreSQL
for metadata and MinIO for bytes.

Files live in spaces, are versioned on every reupload, survive deletion in a trash, and are
reachable over WebDAV as well as the browser. Large uploads go through a resumable chunked
session rather than a single request.

Live at [nuage.facile.studio](https://nuage.facile.studio).

## What it does

- Uploads, downloads, renames, and organizes files in nested folders
- Chunked resumable uploads for large files, with per-session status and abort
- Keeps a version history per file and restores any earlier version
- Soft-deletes to a trash with restore, permanent delete, and empty
- Shares files or folders through public tokenized links with an optional expiry
- Signs time-limited presigned download URLs for unauthenticated clients
- Mounts the whole tree over WebDAV using an API token as the Basic auth password
- Enforces per-user storage quotas, with an admin view and recalculation
- Exposes an incremental sync feed with tombstones so offline clients converge
- Records an activity log, emits events to Nook, and tees its logs to Journal
- Email and password accounts plus optional OIDC SSO with an `SSO_ONLY` mode

## Stack

| Layer | Tech |
|---|---|
| API | Go 1.24, Chi v5, GORM, PostgreSQL 16, MinIO, `go-oidc/v3`, Journal SDK |
| Client | SvelteKit 2, Svelte 5 (runes), Tailwind CSS 4, `adapter-node`, Bun |
| Storage | MinIO for file bytes, a local volume for avatars |
| Deploy | Docker Compose, four services behind Traefik |

## Quick start

```sh
cp .env.example .env
# set POSTGRES_PASSWORD, MINIO_SECRET_KEY, PRESIGN_SECRET (openssl rand -base64 36)
docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build
```

Then open <http://localhost:3000>. Compose refuses to start when a required secret is
missing — there are no default credentials. Plain `docker compose up` is the production
shape and publishes no host ports.

The first account created is promoted to administrator on the next migration run.

### Local development

Start the backing services, then each half in its own terminal:

```sh
docker compose -f docker-compose.yml -f docker-compose.dev.yml up nuage-db nuage-minio -d
```

```sh
cd apps/api && cp .env.example .env && go run .
```

```sh
cd apps/client && bun install && bun run dev
```

The client serves on `5173` and proxies to the API on `4000`.

## Configuration

| Variable | What it does |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `PRESIGN_SECRET` | Signing key for presigned download links; the API refuses to boot without it |
| `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET` | Object storage connection |
| `STORAGE_DIR` | Local directory for avatars, mounted as a volume |
| `ALLOWED_ORIGINS` | Comma-separated CORS origins |
| `OIDC_ISSUER` | Enables OIDC; four companion variables become required with it |
| `ORIGIN` | Public URL of the SvelteKit client, needed for its CSRF check |

Full reference: [docs/configuration.md](docs/configuration.md).

## Structure

```
apps/
  api/       Go backend — modules/ (auth, files, sharing, sync, webdav, ...),
             schemas/ (GORM models and migrations), internal/, tests/ (integration)
  client/    SvelteKit frontend, also a reverse proxy for /api/*
docs/        Architecture, configuration, development, deployment, API
```

## Documentation

| Doc | What's in it |
|---|---|
| [Architecture](docs/architecture.md) | Request flow, data model, how the pieces fit |
| [Configuration](docs/configuration.md) | Every environment variable and default |
| [Development](docs/development.md) | Local setup, tests, the quality gate |
| [Deployment](docs/deployment.md) | Docker Compose, Dokploy, Traefik routing |
| [API](docs/api.md) | HTTP endpoints and payloads |

---

Part of the [Facile Suite](https://facile.studio) — self-hosted tools for creative studios
and freelancers. One login, zero cloud dependency.
