# Nuage — Deployment

How the image is built, what Compose starts, and how Traefik maps one hostname to it.

## Image

Nuage ships one image, built by the root `Dockerfile` with the repo root as context:

1. `oven/bun:1` installs with `--frozen-lockfile` and builds the client with
   `adapter-static`.
2. `golang:1.25-alpine` downloads the module's dependencies, copies `apps/api`, and builds a
   static binary with `CGO_ENABLED=0`.
3. The runtime stage copies the binary and the built client, and runs on port `4000`.

There is no client image and no `BODY_SIZE_LIMIT`: with no SvelteKit server in the path,
upload size is bounded by the API and Traefik alone. The healthcheck is the binary's own
`healthcheck` subcommand, so the runtime needs no shell and no `wget`.

## Compose topology

```
dokploy-network ──▶ nuage-api (:4000, serves the SPA)
                                            │
default network ──▶ nuage-db    (postgres:16-alpine, expose only)
                └──▶ nuage-minio (expose only)
```

| Service | Notes |
|---|---|
| `nuage-db` | Named volume `nuage_db_data`, `pg_isready` healthcheck |
| `nuage-minio` | `server /data --console-address :9001`, volume `nuage_storage_data`, `/minio/health/live` healthcheck |
| `nuage-api` | Volume `nuage_api_data` at `/app/data` for avatars, `/api healthcheck` healthcheck, waits on both backing services being healthy |

The API sets `stop_grace_period: 60s`, which matters because its own
shutdown budget is 45 seconds — a transfer in flight gets a chance to finish instead of
being killed mid-stream.

Postgres and MinIO publish nothing. Their images are pinned to exact tags rather than
floating ones. `docker-compose.dev.yml` is the only file that binds host ports, all on
`127.0.0.1`.

## Traefik

One hostname, one service. `Host(`nuage.facile.studio`)` goes to `nuage-svc`, with a `web`
router redirecting to HTTPS through `redirect-to-https@file` and a `websecure` router with
`tls.certresolver: letsencrypt`. A second pair of routers exists for
``PathPrefix(`/webdav`)``, pointing at the same service — kept explicit because that prefix
must never move under `/api`, and a named router makes that visible.

**No strip-prefix middleware.** `/api` is owned by the Go router, not by Traefik, so the
API's routes are declared *with* the prefix and a local `curl` against port `4000` must
include it. That also keeps the persisted `avatar_url` values — which carry `/api/` in the
stored string — resolving unchanged across the move to one container.

Because nothing is stripped and nothing is proxied, a bare `docker compose up` works with no
edge proxy in front.

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
