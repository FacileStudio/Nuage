# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Releases are tagged `v*` on `main`. The API deploys from `main` on push, so a
tag records what shipped rather than triggering the deploy.

## [Unreleased]

### Added

- **`GET /api/version` names the build the container is serving.** The image build reads
  the commit out of the repository it was built from and links it in, so a deploy can be
  asked which revision it is rather than assumed to be the one that was pushed. It answers
  `{"version":"<commit>"}`, or `dev` for a binary with no stamp — a local `go run`, or a
  build context that carried no `.git`.

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
