# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Releases are tagged `v*` on `main`. The API deploys from `main` on push, so a
tag records what shipped rather than triggering the deploy.

## [Unreleased]

## [0.1.0] - 2026-09-30

First tagged release. The whole history before this commit is folded into it.

### Added

- Files and folders in nested trees, with upload, download, rename and move.
- Chunked resumable uploads — init, parts, complete, status and abort — for
  anything past the single-request limit.
- A version history per file, and restore of any earlier version. A reupload
  keeps the file's id, name, folder, share links and history.
- A chunked upload may name an existing file (`file_id` on complete) so a client
  replaces its content as the next version instead of creating a second object.
- Soft-delete to a trash, with restore, permanent delete and empty.
- Public share links for files and folders, with optional expiry and view/edit
  permissions enforced server-side.
- Time-limited presigned download URLs for unauthenticated clients.
- Spaces with membership roles, gating every space-scoped endpoint, and a
  per-space WebDAV mount.
- An incremental sync feed with tombstones, and a full-state endpoint, so
  offline clients converge.
- WebDAV over the whole tree, using an API token as the Basic auth password.
- Per-user storage quotas, with an admin view and recalculation.
- Search across files and folders by name, with type and folder filters.
- An activity log, Nook webhooks, and log shipping to Journal.
- Email and password accounts, plus optional OIDC SSO with an `SSO_ONLY` mode.

### Known limitations

- A create that collides with a name already in the folder is renamed
  `name (1).ext`. The name check runs before the insert and no constraint backs
  it, so two creates of the same name racing in one folder can both be stored.
  Serializing the deduplication and the insert is the fix; it is not applied yet.
