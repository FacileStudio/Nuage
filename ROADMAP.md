# Nuage Roadmap

Nuage is a minimalist, developer-friendly cloud storage for the Facile Suite. It's not trying to be Nextcloud — it's trying to be the storage layer you actually enjoy using: fast, clean, extensible, self-hosted.

This roadmap is organized in phases. Each phase builds on the previous one.

---

## Current State

**What exists today:**
- File upload/download with drag-and-drop, folders, breadcrumb navigation
- Grid and list views with image/PDF/video/audio preview
- Sharing (public links + user-to-user) with optional expiration
- Trash with restore and permanent delete
- OIDC/SSO + email/password auth
- API tokens for programmatic access
- Nook webhooks (file/folder/share events, HMAC signing)
- Sync endpoints (full state + incremental changes)
- Duplicate filename auto-suffixing
- PostgreSQL + MinIO (S3) backend, Docker Compose deployment
- Chunked uploads with resume support (multi-GB files)
- File versioning (configurable max versions, restore any version)
- Per-user storage quota management
- Activity log with pagination and per-file history
- Share permission enforcement (view/edit, server-side checks)
- API integration tests (auth, files, folders, shares, trash, sync, versioning, quotas, activity)

**What's missing (and why this roadmap exists):**
- No CLI tool
- No desktop/mobile sync client
- No WebDAV
- No E2E encryption
- No 2FA
- No server-side thumbnails
- Share permissions stored but not enforced

---

## Phase 1 — Solid Foundation

Make what exists reliable before adding surface area.

### Testing
- [x] API integration tests (auth, files, folders, shares, trash, sync)
- [x] Upload/download round-trip tests with hash verification
- [x] Share expiration and permission enforcement tests

### Chunked Uploads
- [x] Multipart upload endpoint: init → upload parts → complete
- [x] Resume interrupted uploads
- [x] Remove the 64MB ceiling — support multi-GB files
- [ ] Progress tracking per-chunk on the frontend

### File Versioning
- [x] Keep N previous versions of a file (configurable, default 5)
- [x] Version list endpoint: `GET /files/{id}/versions`
- [x] Restore specific version
- [x] Auto-cleanup of oldest versions beyond limit
- [x] Version diff metadata (size change, date)

### Quota Management
- [x] Per-user storage quota (configurable by admin)
- [x] `GET /quota/me` — current usage vs limit
- [x] Reject uploads when quota exceeded
- [ ] Usage bar in the web UI
- [x] Admin: view all users' usage

### Activity Log
- [x] Log all mutations: uploads, deletes, renames, shares, restores
- [x] `GET /activity` endpoint with pagination and filters
- [ ] Activity feed in web UI (sidebar or dedicated page)
- [x] Per-file activity: `GET /activity/files/{id}`

### Enforce Share Permissions
- [x] `view` = read-only (download, preview)
- [x] `edit` = read + rename + move + upload into shared folder
- [x] Check permissions server-side on every mutation
- [x] Shared folder: respect permission for all contents

---

## Phase 2 — nuage-cli & Sync

The CLI is the power-user interface. WebDAV is the universal one.

### nuage-cli

A standalone binary (Go) that talks to the Nuage API.

```
nuage login https://nuage.example.com
nuage upload ./report.pdf /documents/
nuage download /documents/report.pdf .
nuage ls /documents/
nuage mkdir /documents/2026/
nuage mv /documents/report.pdf /documents/2026/
nuage rm /documents/old.txt
nuage share /documents/report.pdf --expires 7d
nuage sync ./local-folder /remote-folder
nuage search "quarterly report"
nuage whoami
nuage token create "ci-pipeline"
```

- [ ] Auth: `nuage login` stores token in `~/.config/nuage/config.json`
- [ ] Core commands: `ls`, `upload`, `download`, `mkdir`, `mv`, `rm`, `cat`
- [ ] Sharing: `share`, `unshare`, `shares`
- [ ] Sync: bidirectional folder sync using `/sync/changes` endpoint
- [ ] Search: `search` with query
- [ ] Token management: `token create`, `token list`, `token revoke`
- [ ] Output formats: human-readable (default), `--json` for scripting
- [ ] Progress bars for uploads/downloads
- [ ] Glob patterns: `nuage upload *.pdf /documents/`
- [ ] Pipe support: `cat file | nuage upload - /documents/stdin.txt`

### WebDAV Support
- [ ] Mount Nuage as a network drive on any OS
- [ ] Read/write with standard WebDAV clients (Finder, Windows Explorer, Cyberduck)
- [ ] Map Nuage folders to WebDAV collections
- [ ] Auth via session token or API token
- [ ] Enables integration with any tool that speaks WebDAV

### Desktop Sync Agent
- [ ] Lightweight tray app (Go + systray or Tauri)
- [ ] Watch local folder → push changes to Nuage
- [ ] Pull remote changes → write to local folder
- [ ] Conflict resolution: last-write-wins with conflict copies
- [ ] Selective sync: choose which remote folders to sync
- [ ] Uses `/sync/changes` for efficient delta sync

---

## Phase 3 — Developer Experience

Make Nuage a great building block for other tools.

### OpenAPI Specification
- [ ] Machine-generated OpenAPI 3.1 spec from Go handlers
- [ ] Serve at `/docs` with Scalar or Swagger UI
- [ ] Auto-generate TypeScript and Go client SDKs
- [ ] Versioned API (`/v1/`) for stability guarantees

### Webhook Improvements (Nook v2)
- [ ] Retry with exponential backoff (3 attempts, 10s/60s/300s)
- [ ] Delivery log: `GET /settings/nook/deliveries`
- [ ] New events: `file.versioned`, `user.created`, `quota.exceeded`
- [ ] Webhook filters: subscribe to specific event types
- [ ] Batch delivery option (aggregate events over 5s window)

### S3-Compatible API (Subset)
- [ ] `GET /`, `PUT /{bucket}/{key}`, `GET /{bucket}/{key}`, `DELETE /{bucket}/{key}`
- [ ] `ListObjectsV2`, `HeadObject`, `CopyObject`
- [ ] AWS Signature V4 auth using API tokens
- [ ] Enables use of existing S3 tools (rclone, mc, boto3) against Nuage
- [ ] Not full S3 — just the subset that covers file CRUD

### Presigned URLs
- [ ] `POST /files/{id}/presign` — time-limited download URL (no auth header needed)
- [ ] Configurable expiry (default 1h, max 7d)
- [ ] Useful for embedding in emails, external tools, CDN-style access

---

## Phase 4 — Privacy & Security

Self-hosted means self-sovereign. Make it real.

### Two-Factor Authentication
- [ ] TOTP setup (QR code + recovery codes)
- [ ] Enforce 2FA on login when enabled
- [ ] Admin option: require 2FA for all users

### Client-Side Encryption (E2E)
- [ ] Optional per-folder encryption with user-held keys
- [ ] Encrypt before upload, decrypt after download
- [ ] Server never sees plaintext — zero-knowledge for encrypted folders
- [ ] Key management: passphrase-derived (Argon2) or hardware key
- [ ] nuage-cli and web UI both support encryption
- [ ] Shared encrypted folders: key exchange via recipient's public key

### Audit Logging
- [ ] Immutable audit log (append-only table)
- [ ] Track: who, what, when, from where (IP + user agent)
- [ ] Admin-only access: `GET /admin/audit`
- [ ] Export as CSV/JSON for compliance
- [ ] Retention policy (configurable, default 90 days)

### Security Hardening
- [ ] Rate limiting on auth endpoints
- [ ] CSP, HSTS, X-Frame-Options headers
- [ ] Session management: list active sessions, revoke individually
- [ ] API token scoping (read-only, write, admin)
- [ ] Password requirements policy (configurable)

---

## Phase 5 — Scale & Polish

Make it feel complete.

### Server-Side Thumbnails
- [ ] Generate thumbnails on upload (images: resize, PDF: first page render)
- [ ] Store in MinIO alongside originals
- [ ] `GET /files/{id}/thumbnail` endpoint
- [ ] Lazy generation for existing files on first request
- [ ] Dramatically faster grid view (no full-size image fetches)

### Full-Text Search
- [ ] Index file names, metadata, and optionally file content
- [ ] PostgreSQL `tsvector` for simple deployments
- [ ] Optional Meilisearch/Typesense integration for scale
- [ ] Search suggestions and fuzzy matching
- [ ] Filter by type, date range, size, folder

### Multi-Storage Backends
- [ ] Support local filesystem alongside MinIO/S3
- [ ] Configurable per-folder storage backend
- [ ] Migration tool: move files between backends
- [ ] Enables: "hot" local storage + "cold" S3 archival

### Admin Dashboard
- [ ] User management: create, suspend, delete users
- [ ] Storage usage overview (per-user, total)
- [ ] System health: DB size, MinIO status, active sessions
- [ ] Webhook delivery stats
- [ ] Invite system: generate signup links

### Mobile
- [ ] PWA with offline support and installability
- [ ] Camera upload (auto-upload photos from phone)
- [ ] Share-to-Nuage from any app (Web Share Target API)
- [ ] Push notifications for shared files

### Office Document Preview
- [ ] LibreOffice-based server-side rendering (or Collabora CODE)
- [ ] Preview .docx, .xlsx, .pptx without downloading
- [ ] Optional: collaborative editing via Collabora integration

---

## Non-Goals

Things Nuage will **not** become:

- **A collaboration suite** — no calendar, contacts, email, video calls. That's not the mission.
- **A Nextcloud clone** — Nuage is storage-first, not everything-first.
- **An app platform** — no plugin/app ecosystem. Webhooks + API for integrations.
- **A social network** — no comments, reactions, or activity feeds beyond audit.

Nuage is the file layer of the Facile Suite. It stores, serves, syncs, and shares files. Other tools in the suite handle everything else.

---

## Priority Signal

If you're contributing or planning work, here's how to think about priority:

| Signal | Meaning |
|--------|---------|
| Breaks existing workflows | Fix immediately |
| Blocks nuage-cli | Phase 2 priority |
| Security gap | Phase 4 |
| Developer asks for it | Phase 3 |
| Nice to have | Phase 5 |

---

*Last updated: 2026-05-25*
