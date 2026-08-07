# Nuage — Configuration

Every environment variable the code actually reads, grouped by the process that reads it.

Nuage does not use `tronc` — `apps/api/internal/env` loads its own configuration. Anything
missing falls back to a hardcoded default rather than to a shared core.

## API

### Core

| Variable | Required | Default | What it does |
|---|---|---|---|
| `DATABASE_URL` | no | `postgres://postgres:postgres@localhost:5432/nuage?sslmode=disable` | Postgres connection string. The default is a local-dev convenience, not something to rely on in production |
| `PORT` | no | `4000` | HTTP listen port. Must parse to 1–65535 or the process exits 1 |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error`. Anything else exits 1 |
| `STORAGE_DIR` | no | `./data` | Local directory for avatars. `<STORAGE_DIR>/avatars` is created at boot |
| `ALLOWED_ORIGINS` | no | — | Comma-separated CORS origins, trimmed. Unset means an empty list |
| `PRESIGN_SECRET` | **yes** | — | HMAC key for presigned download links. Empty logs `PRESIGN_SECRET is required` and exits 1 |

`PRESIGN_SECRET` is the one hard requirement. Presigned links are unauthenticated by
design, so a guessable key lets anyone forge a download URL for any file. Generate one with
`openssl rand -base64 36`.

### Object storage

| Variable | Required | Default | What it does |
|---|---|---|---|
| `MINIO_ENDPOINT` | no | `localhost:9000` | MinIO host and port |
| `MINIO_ACCESS_KEY` | no | `minioadmin` | Access key |
| `MINIO_SECRET_KEY` | no | `minioadmin` | Secret key |
| `MINIO_BUCKET` | no | `nuage` | Bucket for file bytes; created at boot if absent |
| `MINIO_USE_SSL` | no | `false` | TLS to MinIO. Only the exact string `true`, case-insensitive, enables it |

The defaults are MinIO's own throwaway credentials. Compose overrides all of them and
refuses to start when `MINIO_SECRET_KEY` is unset, but a bare `go run .` will happily come
up on `minioadmin` — that is local-dev behavior, not a supported deployment.

### OIDC

| Variable | Required | Default | What it does |
|---|---|---|---|
| `OIDC_ISSUER` | no | — | Issuer URL. Setting it turns the other four into hard requirements |
| `OIDC_CLIENT_ID` | with issuer | — | Client identifier |
| `OIDC_CLIENT_SECRET` | with issuer | — | Client secret |
| `OIDC_REDIRECT_URL` | with issuer | — | Callback URL, e.g. `https://nuage.facile.studio/api/auth/oidc/callback` |
| `OIDC_SUCCESS_URL` | with issuer | — | Where to send the browser after a successful login |
| `SSO_ONLY` | no | `false` | Only the exact string `true` enables it. Removes `/auth/register` and `/auth/login` from the router |

Setting `OIDC_ISSUER` while leaving any of the four companions empty exits 1 with an
explicit message. The suite's provider is Authentik at `porte.facile.studio`, with a
per-app slug: `https://porte.facile.studio/application/o/nuage/`.

### Log shipping

| Variable | Required | Default | What it does |
|---|---|---|---|
| `JOURNAL_URL` | no | — | Journal API base URL |
| `JOURNAL_TOKEN` | no | — | Per-app Journal ingest key |

Shipping only turns on when **both** are non-empty; otherwise the default `slog` handler is
left alone. The SDK posts to `<JOURNAL_URL>/ingest` and Journal serves ingest at
`/api/ingest`, so the value must end in `/api` — `http://journal-api:4010/api` on the
compose network, `https://journal.facile.studio/api` publicly. Journal's SPA catch-all
answers any unmatched path with `200`, so a base URL missing the suffix looks successful
while every line is discarded.

## Client

The client builds with `adapter-static` and is served by the API binary, so at runtime it
reads nothing at all — there is no SvelteKit server, no `API_URL`, no `ORIGIN` and no
`BODY_SIZE_LIMIT`. The browser always talks to same-origin `/api/*`.

`CLIENT_DIR` tells the API where the built bundle lives; the image pins it, and tronc's
`spa` package refuses to mount a directory that is not there rather than serving 404s.

## Compose substitutions

These are interpolated by `docker-compose.yml` and never reach a Go process directly.

| Variable | Default | What it does |
|---|---|---|
| `POSTGRES_USER` | `nuage` | Internal Postgres user, also spliced into `DATABASE_URL` |
| `POSTGRES_PASSWORD` | none — `:?` required | Internal Postgres password |
| `POSTGRES_DB` | `nuage` | Internal database name |
| `MINIO_ACCESS_KEY` | `nuage-minio` | MinIO root user |
| `MINIO_SECRET_KEY` | none — `:?` required | MinIO root password |
| `PRESIGN_SECRET` | none — `:?` required | Passed through to the API |

`POSTGRES_PASSWORD`, `MINIO_SECRET_KEY`, and `PRESIGN_SECRET` use Compose's `:?` syntax, so
`docker compose up` aborts with the message in the file rather than silently starting on a
default credential. Inside the compose file the API's `STORAGE_DIR` is fixed to `/app/data`,
`MINIO_ENDPOINT` to `nuage-minio:9000`, `MINIO_BUCKET` to `nuage`, and `PORT` to `4000`.

## Tests

The integration suite reads its own set, so it can never point at a live database by
accident.

| Variable | What it does |
|---|---|
| `TEST_DATABASE_URL` | Postgres for the test run |
| `TEST_MINIO_ENDPOINT` | MinIO host and port for the test run |
| `TEST_MINIO_ACCESS_KEY` | MinIO access key for the test run |
| `TEST_MINIO_SECRET_KEY` | MinIO secret key for the test run |
| `CI` | Set by the workflow; the suite refuses to skip when it is set |

## Traps

- **No `PRESIGN_SECRET`, no boot.** It is the only variable whose absence stops the process.
- **A bad `PORT` or `LOG_LEVEL` also exits 1**, before the listener binds, so a restart loop
  is the first symptom.
- **`OIDC_ISSUER` is all-or-nothing.** Setting it alone is a boot failure, not a degraded
  mode.
- **`avatar_url` rows already contain `/api/`.** The public API prefix is baked into stored
  data, so changing where the API is mounted needs either the same public path or a column
  migration. See [architecture.md](architecture.md).
- **Neither `.env.example` is the source of truth.** The root file is a compose template and
  the `apps/api` one is a local-dev template; both drift. This page is generated from the
  code that reads the variables.
