# Nuage — API

Every route the Go API registers, grouped by module, as declared in `apps/api/main.go` and
each module's `RegisterRoutes`.

**Paths below are the API's own.** Traefik strips a `/api` prefix before forwarding, so the
public URL of `/files` is `https://nuage.facile.studio/api/files` while a local `curl`
against port `4000` uses `/files`. `/webdav` is the exception: it is forwarded without a
strip, so the prefix is the same on both sides.

Auth is `Authorization: Bearer <token>`, either a session token or an API token.
`RequireAuth` also accepts a `?token=` query parameter when the header is absent, which is
what lets a plain `<img src>` or a browser download work. WebDAV uses HTTP Basic instead,
with an API token as the password.

An interactive reference generated from `apps/api/modules/docs/openapi.yaml` is served at
`/docs`, with the raw spec at `/docs/openapi.yaml`.

## Health and docs

| Method | Path | Auth |
|---|---|---|
| GET | `/health` | public |
| GET | `/ready` | public |
| GET | `/docs` | public |
| GET | `/docs/openapi.yaml` | public |
| GET | `/avatars/*` | public |

`/ready` pings Postgres and MinIO with a 2-second budget and answers `503` with
`{"status":"not_ready","reason":"database"}` or `"storage"`. `/avatars/*` is a file server
over `STORAGE_DIR/avatars` with `Cache-Control: public, max-age=86400, immutable`.

## Auth

| Method | Path | Auth |
|---|---|---|
| GET | `/auth/config` | public |
| POST | `/auth/register` | public |
| POST | `/auth/login` | public |
| POST | `/auth/logout` | session |
| GET | `/auth/oidc` | public |
| GET | `/auth/oidc/callback` | public |
| POST | `/auth/sync-profile` | session |

`GET /auth/config` returns `{"sso_only":bool,"oidc_enabled":bool}` and drives which options
the login screen offers.

`/auth/register` and `/auth/login` are **not registered at all** when `SSO_ONLY=true`, so
they answer `404` rather than a policy error. The three OIDC routes are registered only when
`OIDC_ISSUER` is set and the provider was reached successfully at startup; a discovery
failure is logged and leaves them unregistered.

Both credential endpoints are rate limited to 10 requests per minute per IP.

## Users

| Method | Path | Auth |
|---|---|---|
| GET | `/users` | session |
| GET | `/users/me` | session |
| PATCH | `/users/me` | session |
| POST | `/users/me/avatar` | session |
| DELETE | `/users/me/avatar` | session |
| GET | `/users/me/api-token` | session |
| POST | `/users/me/api-token` | session |
| DELETE | `/users/me/api-token/{id}` | session |
| GET | `/users/{id}` | session |

A user is `{"id","email","name","avatar_url","avatar_source","color","created_at"}`.
`avatar_source` distinguishes an uploaded avatar from one imported from OIDC, which is what
stops a profile sync from overwriting a deliberate upload.

`POST /users/me/avatar` is multipart and capped at 6 MiB by a `MaxBytesReader`. Accepted
types are PNG, JPEG, GIF, and WebP; anything else is rejected. The stored `avatar_url` is
`/api/avatars/<filename>` — the public prefix is part of the persisted value.

API tokens are the WebDAV password. `POST /users/me/api-token` takes a name and returns the
token.

## Files

| Method | Path | Auth |
|---|---|---|
| POST | `/files` | session |
| GET | `/files` | session |
| GET | `/files/{id}` | session |
| GET | `/files/{id}/download` | session |
| PUT | `/files/{id}` | session |
| DELETE | `/files/{id}` | session |
| POST | `/files/{id}/link` | session |
| POST | `/files/{id}/presign` | session |
| POST | `/files/{id}/reupload` | session |
| GET | `/files/{id}/versions` | session |
| POST | `/files/{id}/versions/{versionId}/restore` | session |
| GET | `/presigned/{token}` | public |

`POST /files` is `multipart/form-data`: the file itself under `file`, plus optional
`folder_id`, `space_id`, and `origin_app` form values. The body is capped at 100 MiB by a
`MaxBytesReader` and the multipart parser buffers 64 MiB in memory. Anything larger belongs
in a chunked session.

`GET /files` filters on `folder_id`, `space_id`, `search`, `linked_to`, and `origin_app`,
and returns `{"files":[…]}`. A file is
`{"id","facile_id","name","mime_type","size","hash","folder_id","origin_app","linked_to","uploaded_by","created_at","updated_at"}`.

`PUT /files/{id}` takes `{"name","folder_id"}` with both optional, so it covers rename and
move. `POST /files/{id}/link` takes `{"linked_to"}`, the cross-app reference other suite
apps use to attach a file to one of their own records.

`POST /files/{id}/reupload` replaces the content and pushes the previous bytes into
`file_versions`. `GET /files/{id}/versions` lists them and
`POST /files/{id}/versions/{versionId}/restore` promotes one back.

`POST /files/{id}/presign` takes `{"expires_in"}` in seconds — default 3600, minimum 60,
maximum 604800 (seven days) — and returns `{"url","expires_at"}`. The URL resolves through
`GET /presigned/{token}`, which takes no credential. The token is
`base64url(payload) + "." + base64url(HMAC-SHA256(payload))` where the payload is just the
file id and the expiry, keyed by `PRESIGN_SECRET` — which is exactly why the API refuses to
boot without one, and why rotating it invalidates every outstanding link.

### Chunked uploads

| Method | Path | Auth |
|---|---|---|
| POST | `/files/upload/init` | session |
| PUT | `/files/upload/{sessionId}/part/{partNumber}` | session |
| POST | `/files/upload/{sessionId}/complete` | session |
| GET | `/files/upload/{sessionId}/status` | session |
| DELETE | `/files/upload/{sessionId}` | session |

`init` takes `{"file_name","mime_type","total_size","folder_id","origin_app","space_id"}`
and returns `{"session_id","expires_at"}`. Each part is a raw `PUT` body capped at 100 MiB.
`status` returns the session plus `uploaded_chunks`, each
`{"part_number","size","hash"}` — which is what makes a resume possible after a network
drop. `complete` assembles the parts and returns the finished file; `DELETE` aborts and
discards them.

All `/files/upload` paths are exempt from the general 100/minute rate limit, since one large
transfer issues hundreds of sequential requests. A background sweeper expires abandoned
sessions.

## Folders

| Method | Path | Auth |
|---|---|---|
| POST | `/folders` | session |
| GET | `/folders` | session |
| GET | `/folders/{id}` | session |
| PUT | `/folders/{id}` | session |
| DELETE | `/folders/{id}` | session |

`POST` takes `{"name","parent_id","space_id"}`; `PUT` takes `{"name","parent_id"}`, covering
both rename and move. `GET /folders` filters on `parent_id` and `space_id`. `GET
/folders/{id}` returns the folder with its immediate `files` and `folders`. A folder's
`size` is computed at response time rather than stored on the row.

## Trash

| Method | Path | Auth |
|---|---|---|
| GET | `/trash` | session |
| DELETE | `/trash` | session |
| POST | `/trash/{type}/{id}/restore` | session |
| DELETE | `/trash/{type}/{id}` | session |

`{type}` is `file` or `folder`. `GET /trash` accepts `space_id`. Deletion is soft — it sets
`deleted_at` — until a permanent delete, which removes the bytes from MinIO and writes a
`tombstones` row so sync clients learn about it.

## Sharing

| Method | Path | Auth |
|---|---|---|
| POST | `/shares` | session |
| GET | `/shares/by-me` | session |
| DELETE | `/shares/{id}` | session |
| GET | `/shared/{token}` | public |
| GET | `/shared/{token}/files` | public |
| GET | `/shared/{token}/download/{fileId}` | public |

`POST /shares` takes `{"file_id","folder_id","permission","expires_at","space_id"}` — one of
`file_id` or `folder_id` — and returns the share including its `token`. The three
`/shared/{token}` routes are unauthenticated by design and back the client's public viewer
at `/s/{token}`; they expose only `{"id","facile_id","name","mime_type","size"}` per item,
never the owner's other files. An expired share stops resolving.

## Spaces

| Method | Path | Auth |
|---|---|---|
| POST | `/spaces` | session |
| GET | `/spaces` | session |
| GET | `/spaces/{id}` | session |
| PUT | `/spaces/{id}` | session |
| DELETE | `/spaces/{id}` | session |
| GET | `/spaces/{id}/members` | session |
| POST | `/spaces/{id}/members` | session |
| PUT | `/spaces/{id}/members/{memberId}` | session |
| DELETE | `/spaces/{id}/members/{memberId}` | session |

A space is a shared container: files and folders carry an optional `space_id`, and
membership carries a role. Membership is unique on `(space_id, user_id)`, and both foreign
keys cascade on delete.

## Sync

| Method | Path | Auth |
|---|---|---|
| GET | `/sync/changes` | session |
| GET | `/sync/state` | session |

`GET /sync/changes?since=<RFC3339>` requires `since` — omitting it or sending a
non-RFC3339 value is `400`. It returns
`{"files":{"changed":[…],"deleted":[…]},"folders":{…},"server_time":"…"}`. A deleted item is
`{"id","facile_id","name","space_id","deleted_at","permanent"}`, where `permanent` separates
a trashed item from a purged one reconstructed from `tombstones`.

Feed the previous `server_time` back as the next `since`. It is deliberately dated slightly
in the past, so a small window of changes is redelivered and clients must apply changes
idempotently by `id`. A client offline longer than the 90-day tombstone retention must
resynchronize from `/sync/state`, which returns the full listing plus a `server_time`.

## Quota

| Method | Path | Auth |
|---|---|---|
| GET | `/quota/me` | session |
| POST | `/quota/me/recalculate` | session |
| GET | `/quota/users` | admin |
| PUT | `/quota/users/{userId}` | admin |

Usage is `{"user_id","storage_used","storage_limit","percentage"}`. `PUT` takes
`{"storage_limit"}` in bytes, where `0` means the instance default and `-1` means unlimited
— the only negative value accepted. `recalculate` re-derives `storage_used` from the files
that actually exist, which is the repair path when accounting drifts.

## Search, settings, activity

| Method | Path | Auth |
|---|---|---|
| GET | `/search` | session |
| GET | `/settings` | session |
| PUT | `/settings` | session |
| POST | `/settings/test-antenne` | session |
| GET | `/settings/antenne/deliveries` | session |
| GET | `/activity/me` | session |
| GET | `/activity/files/{id}` | session |
| GET | `/activity/` | admin |

`GET /search` requires `q` and accepts a `type` filter; an empty `q` is `400`.
`POST /settings/test-antenne` sends a probe event through the Antenne notifier, and
`GET /settings/antenne/deliveries` shows the queue with per-attempt status and response codes.

## WebDAV

| Method | Path | Auth |
|---|---|---|
| OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, MOVE, COPY, LOCK, UNLOCK | `/webdav/*` | Basic |

Backed by `golang.org/x/net/webdav` over a custom filesystem that maps the user's files and
folders onto MinIO. Authentication is HTTP Basic with `WWW-Authenticate: Basic realm="Nuage
WebDAV"`; the **username is ignored** and the password must be an API token. `PUT` bodies
are capped at 2 GiB, locking is in-memory, and the whole prefix is exempt from the general
rate limit.

## Errors and rate limits

Errors come from `apps/api/internal/errors` and are written as JSON by `internal/httpjson`,
with the status the error type maps to: `400` invalid, `401` unauthorized, `403` forbidden,
`404` not found, `409` conflict, `500` internal.

| Scope | Limit |
|---|---|
| Everything except `/files/upload` and `/webdav` | 100/min per IP |
| `/auth/login`, `/auth/register` | 10/min per IP |

The client IP comes from the **rightmost** `X-Forwarded-For` entry, because the leftmost is
caller-controlled and would otherwise let anyone spoof their way past a limit.
