# Nuage — Deployment

How the two images are built, what Compose starts, and how Traefik splits one hostname
across three routes.

## Images

Nuage ships two images, not one.

**API** — `apps/api/Dockerfile`, context the repo root:

1. `golang:1.24.9-alpine` downloads the module's dependencies, copies `apps/api`, and builds
   a static binary with `CGO_ENABLED=0`.
2. `alpine:3.20` installs `wget` for the healthcheck, creates a `nuage` system user, copies
   the binary to `/app/api`, creates `/app/data/avatars`, and runs as `nuage` on port `4000`.

The runtime is Alpine rather than distroless because the container writes to `/app/data` and
the compose healthcheck shells out to `wget`.

**Client** — `apps/client/Dockerfile`, context `apps/client`:

1. `oven/bun:1.3` installs with `--frozen-lockfile` and builds with `adapter-node`.
2. `oven/bun:1.3-slim` copies `build/` and `package.json`, creates a `nuage` user, and runs
   `bun ./build/index.js` with `PORT=3000`, `ORIGIN`, and `BODY_SIZE_LIMIT=2000000000`
   baked in as defaults.

The 2 GB body limit is what lets a large upload survive the SvelteKit proxy hop.

## Compose topology

```
dokploy-network ──▶ nuage-client (:3000)  ──┐
                └──▶ nuage-api    (:4000)  ─┤
                                            ▼
default network ──▶ nuage-db    (postgres:16.11-alpine, expose only)
                └──▶ nuage-minio (RELEASE.2025-04-22T22-12-26Z, expose only)
```

| Service | Notes |
|---|---|
| `nuage-db` | Named volume `nuage_db_data`, `pg_isready` healthcheck |
| `nuage-minio` | `server /data --console-address :9001`, volume `nuage_storage_data`, `/minio/health/live` healthcheck |
| `nuage-api` | Volume `nuage_api_data` at `/app/data` for avatars, `wget` healthcheck on `/health`, waits on both backing services being healthy |
| `nuage-client` | Healthcheck fetches `/` with `bun -e`, waits on the API being healthy |

Both application services set `stop_grace_period: 60s`, which matters because the API's own
shutdown budget is 45 seconds — a transfer in flight gets a chance to finish instead of
being killed mid-stream.

Postgres and MinIO publish nothing. Their images are pinned to exact tags rather than
floating ones. `docker-compose.dev.yml` is the only file that binds host ports, all on
`127.0.0.1`.

## Traefik

One hostname, two services, three route groups:

| Rule | Target | Middleware |
|---|---|---|
| ``Host(`nuage.facile.studio`) && PathPrefix(`/api`)`` | `nuage-api-svc` | `nuage-strip-api` strips `/api` |
| ``Host(`nuage.facile.studio`) && PathPrefix(`/webdav`)`` | `nuage-api-svc` | none — the Go handler owns the `/webdav` prefix |
| ``Host(`nuage.facile.studio`)`` at `priority: 1` | `nuage-client-svc` | none |

The catch-all sits at priority 1 so the two prefixed rules always win. Each group has a
`web` router redirecting to HTTPS through `redirect-to-https@file` and a `websecure` router
with `tls.certresolver: letsencrypt`.

This is where Nuage diverges from the suite's one-container, one-router, one-hostname rule:
two containers answer one hostname, and the `/api` prefix is a Traefik concern rather than
part of the Go router. Two consequences follow. The API's own routes are declared without
the prefix — `/auth/login`, not `/api/auth/login` — so any local `curl` against port `4000`
must drop it. And `avatar_url` values are persisted with `/api/` already in them, so the
public mount point is effectively part of the data. Consolidating to one container means
either preserving the same public paths or migrating that column.

Note that Traefik is not the only path to the API: the SvelteKit server proxies `/api/*` and
`/webdav*` to `API_URL` itself, which is what makes a bare `docker compose up` work without
an edge proxy in front.

## Deploying on la ruche

Deployments are managed through Dokploy at `gare.facile.studio`, which owns the environment
file and triggers the Compose build. Prefer the `dokploy` CLI over SSH plus `docker`.

Set `POSTGRES_PASSWORD`, `MINIO_SECRET_KEY`, and `PRESIGN_SECRET` — Compose refuses to start
without all three. Set `ORIGIN` to the public URL, or the SvelteKit CSRF check rejects every
form post. Set `ALLOWED_ORIGINS` if anything cross-origin needs to reach the API. For SSO,
set the five `OIDC_*` variables together and point `OIDC_REDIRECT_URL` at
`https://nuage.facile.studio/api/auth/oidc/callback`. For log shipping, set `JOURNAL_URL`
ending in `/api` and a `JOURNAL_TOKEN` minted on Journal's Keys page.

**Rotating `PRESIGN_SECRET` invalidates every outstanding presigned link.** That is the
point, but it is not a free operation.

## Migrations

There is no migration tool and no separate step. `schemas.Migrate` runs at boot inside one
transaction holding `pg_advisory_xact_lock(4919)`, so several instances starting at once
serialize instead of racing. It runs the hand-written `preMigrate` DDL, then `AutoMigrate`
over sixteen models, then `ensureAdmin`, then backfills user colors. A failure returns
before the listener binds.

## Health and readiness

`/health` returns `{"status":"ok"}` unconditionally. `/ready` pings Postgres and calls
`EnsureBucket` on MinIO with a 2-second budget, returning `503` with
`{"status":"not_ready","reason":"database"}` or `"storage"`. Use `/health` for restarts and
`/ready` for traffic.

Both are declared on the API router without a prefix, so publicly they are `/api/health` and
`/api/ready` after Traefik strips the prefix — and the compose healthcheck reaches
`http://localhost:4000/health` directly inside the container.

A green health check proves only that the Go process is up. It says nothing about the
SvelteKit container, which has its own healthcheck fetching `/`. Verify a deploy by loading
the site root, not by curling the API.
