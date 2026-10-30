# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Releases are tagged `v*` on `main`. The API deploys from `main` on push, so a
tag records what shipped rather than triggering the deploy.

## [Unreleased]

## [0.3.0] - 2026-10-30

### Added

- **Every space has its own MinIO bucket, `nuage-space-<facile_id>`.** Files, versions and
  chunked uploads in a space are written there; personal files stay in the shared bucket. The
  new `bucket` column on `files` and `file_versions` records where an object lives, and an
  empty value means the shared bucket. At boot `MigrateBuckets` copies each existing space's
  objects into its bucket, repoints the rows, then deletes the originals. It is safe to
  re-run, and if it fails the API still starts and serves the files from the shared bucket.
- **API tokens can be restricted to spaces.** `POST /api/users/me/api-token` accepts `app`
  and `space_facile_ids` together. A restricted token may only use the file endpoints and the
  find-or-create below, for those spaces. It cannot reach personal files, WebDAV, sharing,
  trash or token management, cannot replace an existing file, and stamps `origin_app` with its
  `app`. This is how Ardoise and other apps store files without a forwarded session.
- **`PUT /api/spaces/by-facile-id/{facile_id}` finds or creates a space.** It answers 201 when
  it creates the space and makes the caller owner, 200 for a member of an existing one, and 403
  for anyone else. Concurrent calls produce one space.

### Changed

- **Deleting a space is refused with 409 while any file row, trashed ones included, still
  points at it.** The foreign keys already blocked it with a generic 500.

### Fixed

- **Switching space while inside a subfolder returns to the root of the new space** instead of
  opening the old folder id in the wrong space.

## [0.2.0] - 2026-10-01

### Added

- **`GET /api/version` names the build the container is serving.** The image build reads
  the commit out of the repository it was built from and links it in, so a deploy can be
  asked which revision it is rather than assumed to be the one that was pushed. It answers
  `{"version":"<commit>"}`, or `dev` for a binary with no stamp — a local `go run`, or a
  build context that carried no `.git`.

### Changed

- **The container runs as `nonroot`, and every base image is pinned by tag**
  (`oven/bun:1.3.13`, `golang:1.26.8-alpine`, `gcr.io/distroless/static-debian12:nonroot`),
  so a rebuild no longer moves on its own to whatever the floating tag points at. Nothing
  in the image needed root: file bytes go to MinIO, WebDAV stages uploads in the base's
  `1777` `/tmp`, and the only directory the API writes is `STORAGE_DIR`, which the image now
  hands over already owned by 65532. **An existing `api_data` volume was written by the
  root container and must be chowned once** — `docker run --rm -v <project>_api_data:/data
  alpine chown -R 65532:65532 /data` — or avatar uploads fail. Starting is not what breaks:
  the startup `MkdirAll` finds `avatars` already there and the failure waits for the first
  avatar change.

## [0.1.2] - 2026-10-01

### Fixed

- **A rename or a move that landed on a name the destination folder already
  held stored a second file under that name.** The create path deduplicated, but
  `PUT /files/{id}` wrote the name it was handed, so a client move could leave
  two files holding one name in a folder — and the sync clients, which match by
  name, would then oscillate between them. A rename or move now takes the
  destination folder's name lock and deduplicates against it, excluding the file
  being moved so it can still keep its own name.

## [0.1.1] - 2026-09-30

### Fixed

- **Two creates of the same name in one folder could both be stored.** The name
  check ran before the insert with nothing serialising them, so a concurrent
  pair — the sync client, WebDAV, the web UI — could both see the name free and
  both take it. The check and the insert now run in one transaction under a
  per-folder advisory lock, so a create sees the row a rival committed before it
  decides on a name. The upload stays outside the lock, so it is the insert that
  is serialised, not the transfer; the object key no longer carries the name as
  a result.

### Known limitations

- WebDAV's own create path still stores the name it was handed without checking
  for a collision, so a PUT racing a create of the same name can still double it.
  The API paths — single-request upload, chunked complete, and folder create —
  are serialised.

## [0.1.0] - 2026-09-30

First tagged release. Everything before it is folded in here, grouped by the
wave each part landed in; the dates come from the commits, and the tags start
now rather than at the first commit.

### 2026-05-24 — the platform

- Files and folders in nested trees, with upload, download, rename and move.
- Chunked resumable uploads — init, parts, complete, status and abort — for
  anything past the single-request limit.
- A version history per file, and restore of any earlier version.
- Soft-delete to a trash, with restore, permanent delete and empty.
- Public share links for files and folders, with optional expiry and view/edit
  permissions enforced server-side.
- Time-limited presigned download URLs for unauthenticated clients.
- WebDAV over the whole tree, using an API token as the Basic auth password.
- An incremental sync feed with tombstones, and a full-state endpoint, so
  offline clients converge.
- An activity log, Nook webhooks, and per-user storage quotas.
- Search across files and folders by name, with type and folder filters.
- Email and password accounts, plus optional OIDC SSO.

### 2026-06 — spaces

- Spaces with membership roles, gating every space-scoped endpoint; a
  caller-supplied `space_id` cannot reach a space the caller is not in.
- Space filtering carried through files, folders, activity, trash and shares.

### 2026-07 — log shipping

- Log shipping to Journal through a slog tee, with a per-app key.

### 2026-08 — the suite chassis

- Accounts, sessions and space authorization moved onto porte; the HTTP layer
  onto tronc.
- Sync tombstones, per-real-IP rate limiting, and a device-grant token the
  suite CLI signs in with.
- The client rebuilt on muse and served from the API binary, in one container.
- Proxy trust made explicit, so rate limits key on the real visitor.

### 2026-09 — per-space WebDAV and versioned completes

- Each space mounted at `/webdav/spaces/{id}`, with a copyable link per space.
- A chunked upload may name an existing file (`file_id` on complete) so a client
  replaces its content as the next version instead of creating a second object,
  keeping the file's id, name, folder, share links and history.
