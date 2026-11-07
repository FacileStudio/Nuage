# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Releases are tagged `v*` on `main`. The API deploys from `main` on push, so a
tag records what shipped rather than triggering the deploy.

## [Unreleased]

### Changed

- **One user menu and one settings shell** (`@facile/muse` v0.13.0 to v0.14.1). The sidebar user
  card and the mobile avatar open the muse `UserMenu`: account, settings, subscription (when a
  plan sets the limit), language, theme, help and sign out. The cog link is gone. Settings use
  `SettingsLayout` with the canonical order: Profile, Preferences, App data, Nook, API tokens,
  Subscription, Advanced. Profile is read-only and links to Facile; the name form and the
  in-settings sign-out are removed.
- The API tokens section is muse `TokensSection`; the API calls are unchanged (a token is still
  created with `{name}` and never expires, so the expiry choice in the drawer is not sent). The
  WebDAV URLs stay in that section. The storage limit and plan source moved into the new
  Subscription section.
- `appearance` is now `preferences` and `api` is now `tokens`; the old addresses redirect.
- A language picked in the menu is stored in the `fc_locale` cookie and wins over the Registre
  profile locale on the next visit.

### Fixed

- The storage meter and plan data never loaded: `refreshQuota` returned early because the
  bearer token is empty under cookie sign-in.

## [0.13.0] - 2026-11-07

### Changed

- **Sign-in goes through Registre only.** The local register and login routes, the password
  and email-change paths and the `SSO_ONLY`/`AllowRegistration` wiring are gone. `/login` is
  muse `AuthScreen`, and every sign-in button, on the landing page, header and login screen, is
  `SignInWithFacile`. The landing page uses `LandingHeader` and `LandingFooter`. `/auth/register`
  and `/auth/login` answer 404. API tokens (CLI, daemon, WebDAV) and delegated JWTs are
  untouched, and the share links stay public.
- **The email address is Registre's.** `PATCH /users/me` takes `name` and `color` only and
  answers 400 to any other field. porte v0.10.0 made `Kit.Verify` refuse under OIDC, which ended
  the old "confirm with the current password" email change; the address now refreshes from the
  `email` claim at each sign-in.
- **Account linking follows porte v0.10.0.** A sign-in links to an existing account only by a
  verified email, and is refused when the account already holds another subject of the same
  issuer. `users.locale` stores the `locale` claim.
- Libraries: porte v0.7.0 to v0.12.1, `porte/directory` v0.2.1 to v0.3.0, tronc v0.16.0 to
  v0.17.1, `@facile/muse` v0.11.0 to v0.13.0 (borderless, custom `Select`). The local security
  headers are replaced by tronc's: HSTS, an enforced anti-framing policy and a report-only CSP
  carrying the client's inline script hashes and the plan catalogue origin. The CSP stays
  report-only. The client `/docs` route (Scalar from jsDelivr, a stale copy of the server's
  `/docs`) is removed; `/docs` is served by tronc's `apiref` with its own CSP.
- `APP_VERSION` is a build argument (compose passes it through) and becomes the version served
  at `/version` and `/api/version`; without it the commit stamp is used.
- No borders: the file browser, breadcrumbs, selection toggles, viewers and drop overlay use
  fills instead.

### Added

- **The UI in French, English, Spanish and German**, from the Registre profile locale, the
  `fc_locale` cookie or the browser. Dates, sizes and relative times follow the locale. The
  Spanish and German texts need native review. API error messages stay English.
- **GDPR deletion.** Nuage polls Registre's deletion list and accepts its signed push on
  `POST /api/registre/deletions`. A deleted person loses their personal files, folders, app
  folders, tokens, quota and account. Space files and folders they authored stay, attributed to a
  hidden "Deleted user" placeholder, and their public share links are revoked. Replays are
  no-ops. Polls and pushes hold a Postgres advisory lock, so the old and the new process of a
  start-first deploy never handle deletions at the same time.
- `ci.yml`: gofmt, vet, build, tests against Postgres and MinIO (three shards), client check, tests and build,
  and filet.

### Migrations

All run at boot and are idempotent: `users.locale` (AutoMigrate), `porte_login_codes.provider`
and `.subject`, and the `directory_deletion_state` table.

### Removed

- `POST /api/auth/register`, `POST /api/auth/login`, the per-IP limit on them, the password
  fields of `PATCH /users/me` and `SSO_ONLY`. A client that signed in with the account password
  must use an API token.

## [0.12.1] - 2026-11-06

### Added

- **Pricing on the landing page.** A "Pricing" section shows the live plans from Ardoise's public catalog through muse's `PricingPlans`, with the storage allowance first. It stays hidden while the catalog is empty or unreachable.

### Changed

- `@facile/muse` v0.8.1 to v0.11.0 (icon set regenerated). Nuage has no locale, so muse's English defaults apply.

## [0.12.0] - 2026-11-06

### Added

- **App folders.** Each app registered in the new `NUAGE_APPS` setting
  (`key:registre_client_id[:Name]`, `;`-separated, needs `OIDC_MACHINE_AUDIENCE`) gets a
  private folder per user, created on first use. It is never a space, never shows in the file
  tree, and its bytes count toward the user's personal plan under the same
  `nuage:plan:user:<id>` lock as personal files, trash, versions, charged spaces and open uploads.
  Apps are rows in `registered_apps`, seeded from the setting at every boot; an app removed from
  it stops acting but its users keep their data.
- **App API (`/api/app`).** `GET /app/objects`, `GET|PUT|DELETE /app/object?key=`. The caller is
  a registered app acting for one user: a delegated Registre JWT (token exchange, `sub` = the
  user, `aud` = Nuage, `act.sub` = the app's client id). It reaches only that user's folder for
  its own app. Objects are at most 100 MiB, replacing a key charges only the growth, and a full
  plan answers 402. Cookie sessions, API tokens (including restricted tokens labelled with an
  app name) and JWTs of unregistered actors get 403 there, and a registered app's JWT gets 403
  everywhere else.
- **Delegated actors are denied by default.** A delegated JWT whose actor (`act.sub`) is
  neither an active registered app nor listed in the new `NUAGE_FULL_ACCESS_ACTORS` is refused
  on every route, so an app removed from `NUAGE_APPS` loses access instead of gaining the whole
  drive. docker-compose defaults the list to `bulle`, which keeps Bulle working.
- **App data for users.** `GET /api/app-data` lists per-app usage, and `DELETE
  /api/app-data/{app}` deletes everything that app stores for the caller (own session only).
  Settings has a new App data section with both. An unrestricted personal API token counts as
  the person here, since porte does not tell it apart from a sign-in session.
- **`migrate-app-folders` command.** Moves the files of app-provisioned spaces into the charged
  user's folder for the registered app whose key equals the space's `provisioned_by` label.
  Dry run by default (`--apply` to move), JSON report on stdout, idempotent. Only live files
  without versions or share links, published by the space's own app (`origin_app`), move; the
  rest stay and are reported. It reads the registry the server synced and never writes the
  schema or the registry. Spaces are not deleted.

### Security

- WebDAV refuses every delegated JWT in the Basic password. WebDAV never runs `RequireAuth`, so
  any JWT used to reach the whole drive through it, whatever its actor.

## [0.11.1] - 2026-11-06

### Fixed

- **A person's plan is read under the subject Registre knows.** A user holding several
  identities under the issuer (such as one left from the Authentik era) had their plan read
  under whichever row Postgres returned first. Nuage now prefers the subject named in the last
  directory snapshot (one the plan store accepted, never one it ignored as older), else the
  most recently synced identity. Boot logs
  `forfait: user holds several identities under the issuer` for each such user; no row is
  deleted.
- **Go-live reports.** Boot also logs everyone over the free plan with all their subjects, and
  the snapshot that first makes plans apply logs every person and space over their resolved
  ceiling. `docs/deployment.md` lists the steps to create Ardoise overrides before go-live.
- **A session can no longer find-or-create a space once Registre owns spaces.**
  `PUT /api/spaces/by-facile-id/{id}` answers 409 for a session caller, as `POST /api/spaces`
  does; app tokens still provision.
- **The quota meter under an admin limit** shows the usage plans count (app-provisioned spaces
  charged to the person included) against that limit.
- **Parallel chunks cannot stage past `total_size`, and a late chunk cannot land during
  assembly.** Each chunk is stored under its own object key, then counted and recorded under
  its session's row lock after re-checking the session is still pending; a replaced part's
  object is deleted only once the new row is committed.
- porte/directory v0.2.1 and porte/forfait v0.1.1 (restore-safe ordering, serialised sync).
  An empty saved `entitlement_state` row still makes Nuage enforce the free-plan defaults, not
  fail open. `docs/deployment.md` documents the recovery after a Registre restore: delete the
  row and restart.
- The people over the free plan were counted once per identity, so a user with two subjects
  showed double usage in the boot report.

## [0.11.0] - 2026-11-04

### Added

- **Storage follows the Registre plan (`nuage.storage_gb`, free plan 5 GB).** Every directory
  sync now also feeds the plan entitlements to porte/forfait, and the state is saved in the new
  `entitlement_state` row and restored at boot, so a restart keeps the last known plans. A saved
  row that cannot be imported enforces the free-plan defaults until Registre answers, and a
  database error while reading it is retried at boot instead of failing open. Files
  in a space count against that space's Registre plan; personal files count against the owner's
  person plan. Usage is every stored byte (files, trashed ones included until they are purged,
  plus their versions, plus the declared size of open chunked uploads) against
  `storage_gb * 1024^3`, and `-1` is unlimited.
- **Every path that adds bytes is checked**: single and chunked uploads (at session start with
  the declared size, atomically with the session insert, and again at completion), WebDAV PUT
  and COPY, version uploads and restores. The check runs again inside the transaction that
  records the bytes, under a per-scope advisory lock, so two uploads racing at the ceiling
  cannot both land. Trash restore adds nothing, since trashed bytes still count.
- **At most 10 chunked uploads open per user.** The eleventh `POST /files/upload/init` answers
  429 until one completes, is aborted or expires.
- **Over the plan answers HTTP 402** in the usual error envelope with `code: "payment_required"`,
  `key`, `limit` (GB) and `upgrade_url` (the Registre space page, or `/spaces` for personal
  files). WebDAV answers 507 Insufficient Storage. The client shows a banner with an upgrade
  link (Registre's exact origin or a same-origin path only), and the usage meter shows the plan ceiling.
- **Boot report of usage above the free plan.** At every boot the API logs a
  `forfait: ... storage above the free plan` warning per person and space storing more than
  5 GB, so overrides can be added in Ardoise before its first entitlements push.

### Changed

- Until Registre serves a snapshot with a non-zero `entitlements_revision`, nothing changes:
  the per-user quota and its instance default apply exactly as before. Afterwards an explicit
  per-user limit set by an admin replaces the person's plan ceiling (`-1` unlimited), measured
  against the same usage the plan counts, app-provisioned spaces charged to them included. It
  no longer applies to uploads into Registre-linked spaces, which follow the space's plan. A space another app provisioned (Plume, Ardoise) is charged to the
  personal plan of the user who created it: `spaces.charged_to` is set at creation and
  backfilled at boot from the earliest owner of existing provisioned spaces (each one logged).
  Those spaces and the person's own files share one ceiling under one lock. Any other space
  not linked to Registre refuses new files with 412 once plans apply.
- **The payer of an app-provisioned space cannot leave or be removed while others remain** (409).
  Billing moves in two steps: the payer offers it to another owner with
  `POST /api/spaces/{id}/billing {"user_id"}` (open 7 days, withdrawn with `DELETE`), and only
  that owner's own `POST /api/spaces/{id}/billing/accept` moves it. App tokens and delegated
  suite JWTs cannot offer, withdraw or accept: consent comes only from the person's session. A space whose payer's account was deleted refuses new files (412) until the next boot
  reassigns it to its earliest remaining owner.
- **Idle chunked uploads stop reserving.** A pending session with no chunk for an hour no longer
  counts toward the plan; its next chunk must fit the plan again (402 otherwise). It still
  counts toward the 10-upload cap until it expires. The web client aborts a session whose
  upload fails.
- Downloads, listings, deletes and emptying the trash are never gated, whatever the usage.

## [0.10.3] - 2026-11-03

### Fixed

- **The desktop sidebar drops the Spaces entry.** The space switcher at the top of the sidebar
  already lists spaces and links to "Manage spaces", so the bottom nav row was a duplicate. The
  mobile bar keeps it, since it has no switcher.

## [0.10.2] - 2026-11-03

### Fixed

- **"Back to spaces" uses the back chevron.** The button on the "Space not found" state of the
  space, members and settings pages shows `icons.chevronLeft` again, like every other
  back-navigation button in the suite. Spaces keep `icons.space` everywhere else.

## [0.10.1] - 2026-11-03

### Changed

- **Spaces use the share-circle icon.** The web app moves to muse v0.8.1 and shows
  `icons.space` on the Spaces nav item (sidebar and mobile bar), the space cards, the "No spaces
  yet" and "Space not found" states, "Back to spaces" and "Manage in Registre". Member lists keep
  the people icons.

## [0.10.0] - 2026-11-03

### Changed

- **Registre owns shared spaces.** With OIDC configured, Nuage hands its user-made spaces to
  Registre once at boot, retrying until Registre answers, then mirrors Registre's spaces and
  members at boot, every 5 minutes and within seconds of a nudge on `POST /api/registre/sync`.
  A space links by Registre id or, the first time, by name. Members Registre adds are
  provisioned before their first sign-in; members it removes lose access; local password
  accounts are never removed. Folders, files and the description stay Nuage's. Spaces that
  Plume or Ardoise provisioned are left out and work as before.
- **Space management moves to Registre.** The spaces pages link to
  https://sso.facile.studio/spaces instead of offering create, rename, delete, invite, role
  change, remove and leave. For a linked space those endpoints answer 409 with the Registre
  page, and `POST /spaces` answers 409 whenever OIDC is configured. The member lists stay, with
  pictures.

## [0.9.0] - 2026-11-03

### Changed

- **Borderless cards, panels and floating bars.** The web app moves to muse v0.8.0 and drops the
  decorative borders on the landing cards, the login error and tabs, list containers, the context
  menu, the viewers' toolbars and the member rows. Tone and spacing separate them now. Inputs,
  focus rings, the drop zone and the selection toggle keep theirs.

## [0.8.0] - 2026-11-03

### Added

- **Delegated access for suite services.** With `OIDC_MACHINE_AUDIENCE` set to Nuage's client
  id and `OIDC_SCOPE_PREFIX=nuage`, a short-lived JWT that Registre mints for another suite
  service (Bulle) opens the API for the signed-in user it names. `nuage.read` allows GET, HEAD
  and OPTIONS; anything that changes data needs `nuage.write`. A delegated token never holds the
  admin role. Both variables are empty by default, which leaves behaviour unchanged.
- **Registre profile pictures for everyone, without signing in again.** At boot, every SSO
  account is pointed at `<OIDC_ISSUER>/avatars/<subject>`, the stable URL Registre serves the
  current picture from. A picture changed in Registre shows on the next page load, and a person
  without one sees initials.
- **The activity feed names who did what.** Each event carries an `actor` with `name`, `email`
  and `avatar_url`, and the feed shows that person's picture or initials.

### Changed

- porte v0.5.2 to v0.7.0.
- @facile/muse v0.5.0 to v0.6.5.
- **Pictures are set in Registre.** Settings links to `https://sso.facile.studio/account`
  instead of offering an upload. Uploaded pictures stay stored and still show for an account
  with no Registre picture URL. `POST` and `DELETE /api/users/me/avatar` are unchanged.

## [0.7.0] - 2026-11-01

### Added

- **Server-side thumbnails.** The grid loads 320 or 640 px JPEGs instead of the originals:
  51 phone photos of 3.8 MB come to 588 KiB instead of about 188 MB. JPEG, PNG, GIF and WebP
  are supported, EXIF rotation is applied, and renders are cached in the file's own bucket and
  deleted with the file. Public shares get the same through `GET /api/shared/{token}/thumbnail/{id}`.
- **The trash empties itself.** Items deleted more than `trash_retention_days` ago (default 30,
  0 turns it off, admins set it under Settings > Advanced) are purged hourly, and each
  uploader's quota is refunded. The trash page shows when each item goes. **The first run after
  this release permanently deletes everything already trashed for more than 30 days.**
- **The public share page is rebuilt** on the file viewers: a shared file opens inline, a shared
  folder is a card grid with thumbnails, a full-screen viewer and next and previous.
- **Phone gestures:** swipe between files in the viewer, pinch to zoom images and PDFs, and a
  bottom previous/next bar. Touch targets are 44 px and the viewers respect safe areas.

### Changed

- Files uploaded before v0.6.0 as `application/octet-stream` get their type from the extension
  at startup (audio, video, image and PDF extensions only; specific types are never changed and
  `updated_at` is left alone), so they preview.
- The pdf.js worker is bundled with a hashed, immutable URL, and icons register before first
  paint. Grid tiles render lazily in large folders.
- On phones the grid actions button sits over the thumbnail, folders stack in one column, and
  the trash row actions are icons.
- WebDAV uploads use the same media-type table as the REST API.

### Fixed

- **Image zoom did nothing.** Every zoom reset to 100% at once; wheel, buttons, double-click and
  pinch now work.
- **A trashed file stayed downloadable through its share link.** A share of a trashed file or
  folder now answers 404.
- **Saving Advanced or Nook settings blanked them.** The settings page read the API response
  wrongly and wrote empty values back, including the instance name and Nook configuration.
- Video controls no longer wrap at 360 px.

## [0.6.0] - 2026-11-01

### Added

- **Full-screen file viewer** with next and previous across the folder (arrow keys too),
  file details, Share and Download in the top bar.
- Images zoom with the wheel, pinch or double-click, pan by dragging, and rotate.
- **PDFs scroll continuously** through pdf.js's own viewer, with selectable text, find in
  document, a page jump field, zoom and fit to width.
- Audio and video get custom controls: seek with buffered progress, skip 10 seconds, speed,
  volume, full screen, and keyboard shortcuts (space or k, j, l, m, f).
- **Move to a folder** from the context menu or the selection toolbar, through a folder
  picker. "Move to <parent>" moves items up one level. The up button beside the breadcrumbs
  navigates up and accepts drops. Dragging a selected item moves the whole selection.

### Changed

- The grid view uses muse cards, groups folders above files, and keeps an actions button
  on every tile, visible without hover on touch screens.
- **Rate limiting covers `/api/*` only**, at 600 requests a minute per IP, and no longer
  counts uploads, downloads or presigned links. Opening a folder of photos or a long PDF
  used to answer 429.
- Thumbnails load from a versioned URL the browser caches for a day.
- An upload sent as `application/octet-stream` takes its type from the extension for common
  image, audio, video and PDF formats, so it can be previewed.
- Space members can rename and move a teammate's file, as they could already delete it.

### Fixed

- Moving an item to the root (drop on "Files", undo of a move out of the root) did nothing.
- A folder could be moved into its own subfolder over the API or WebDAV, which made the whole
  branch vanish.
- A file could be moved into a folder of another space, or into a trashed folder.
- The app no longer calls the removed `/api/auth/sync-profile` endpoint on every load.

## [0.5.0] - 2026-10-30

### Added

- **Chunked uploads publish idempotently.** `POST /api/files/upload/init` takes `linked_to` for
  a restricted token, and completion applies the same rule as a plain upload: when the app
  already holds a live file under that key in the space, that file is returned (200) and
  nothing is stored. If the key already exists at init, the answer is 200 with `existing_file`
  and no session is created, so a large retry never re-sends the bytes.

### Changed

- **An existing space the caller cannot use answers 404** on `PUT /api/spaces/by-facile-id/{id}`
  instead of 403, so the error no longer says that the id exists. A caller allowed to create
  can still tell a free id (201) from a taken one (404).
- A chunked completion that finds an identical file in the folder returns it with 200, like a
  plain upload.

### Fixed

- **A new user's first concurrent requests no longer fail with 500.** The quota row was found
  and then created, so two requests racing for a user's first quota both inserted and one hit
  the unique constraint ("failed to load quota"). This showed up as sync clients losing the
  first uploads of a fresh account.
- Concurrent find-or-create of the same space no longer returns 500 when two calls both try to
  create its bucket.

## [0.4.0] - 2026-10-30

### Added

- **Tokens can be minted with `provision`.** `POST /api/users/me/api-token` accepts
  `provision: true` with `app`. Such a token may create spaces through
  `PUT /api/spaces/by-facile-id/{facile_id}` and afterwards reaches every space that its app
  created, plus any it lists. Spaces made by users or by another app stay out of reach even if
  the token knows their id, and the acting user must still be a member. A new
  `spaces.provisioned_by` column records which app created a space.
- **Publishing is idempotent.** `POST /api/files` from a restricted token with a `linked_to`
  key returns the existing file (200) when that app already holds a live file under that key in
  the space, and stores nothing. Concurrent publishes of one key produce one file. Chunked
  uploads are not idempotent yet.

### Changed

- **Uploading the same bytes under the same name twice gives one file.** When a folder or space
  already holds a live file with that exact name and content hash, the upload returns it (200)
  instead of creating `name (1)`. The check runs under the folder's name lock, so sync clients
  that upload the same file at once no longer leave a copy behind. Same name with different
  content still gets a `(1)` name.

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
