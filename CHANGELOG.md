# Changelog

All notable changes to this project are documented here. The format is
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While on
`0.x`, a breaking change bumps the minor.

Releases are tagged `v*` on `main`. The API deploys from `main` on push, so a
tag records what shipped rather than triggering the deploy.

## [Unreleased]

## [0.20.2] - 2026-11-14

### Security

- API: porte v0.20.1 and porte/mcpauth v0.3.1. porte v0.20.0 marked a browser session token with `b-`, which about 1 in 4096 random CLI or API tokens also start with, so such a session could confirm a pending MCP action on `/pending`. The mark is now `b~`, which a random token never contains.

### Upgrade notes

- Browser cookies issued by v0.20.0 or v0.20.1 stop counting as a browser session, so each person signs in once more before they can confirm an action.

## [0.20.1] - 2026-11-14

### Fixed

- The landing's download section no longer scrolls the page sideways on narrow desktop windows: the CLI card now fits the column and long commands scroll inside their own box.

## [0.20.0] - 2026-11-14

### Changed

- API: tronc v0.23.1, porte v0.20.0, porte/mcpauth v0.3.0 and porte/forfait v0.1.3. The MCP server uses the tronc kit for its state key (byte-identical), list pagers, waiting texts, plan errors and the hourly prune of finished actions.
- API: `nuage_save_file` on a full storage plan says nothing was saved, or that the file was not replaced and keeps its content, and offers freeing space or upgrading. The upgrade link is shown only when it is a safe URL.

### Fixed

- API: only the browser session cookie can list, confirm or dismiss a pending MCP action on `/pending`. A bearer token, a CLI, device or mobile session, or an API key now gets 403. With tronc v0.23.1, a confirmation asked by one OAuth client is never reused by another.
- Client: when the confirm page is refused, it asks the person to sign in again in this browser instead of showing the raw error.

### Upgrade notes

- Each person signs in once in the browser before they can confirm an action.
- A `nuage_list_folder` or `nuage_list_shares` cursor issued before the deploy is refused once; the model starts the listing again.

## [0.19.0] - 2026-11-14

### Added

- Client: the landing page shows a mise-en-scene demo: the `nuage` daemon uploads a file, which then appears in Files and is shared with a link.
- Client: a public `/downloads` page lists every way to install the `nuage` CLI (install script, Homebrew, `facile install nuage`), with links to the releases and checksums. The landing's download section shows only the visitor's device: the CLI commands on a desktop, an "All downloads" link on a phone.

### Changed

- Client: muse v0.23.0 (was v0.21.0), so the app inherits Magnetik for titles and body and the landing footer is left-aligned on phones. Headings use `font-bold`. The landing copy is shorter, with four feature blocks.

### Removed

- Client: the unused `@fontsource-variable/inter` dependency.

## [0.18.5] - 2026-11-14

### Fixed

- Containers now honour `restart: unless-stopped` instead of the swarm-only `restart_policy: any`.

## [0.18.4] - 2026-11-14

### Added

- Client: `app.html` loads Vision's tracker (`s.js` with `data-mask`) on `nuage.facile.studio` only, and never on a `/s/` share link, whose path carries the share token. The security config adds `https://vision.facile.studio` to `script-src`.

## [0.18.3] - 2026-11-13

### Changed

- Client: muse v0.21.0 (was v0.18.0). Icon-only controls in muse components now show a muse tooltip.

### Fixed

- Client: every icon-only control shows a muse tooltip: the file, image, PDF and media viewer toolbars, the row actions, the list/grid switch, the share page download links. The breadcrumb "up" button shows its drop hint in a muse tooltip instead of the native `title`.

## [0.18.1] - 2026-11-13

### Changed

- Deploy: healthchecks run every 30 s with a 1 s start interval.

## [0.18.0] - 2026-11-13

### Changed

- Database: the schema is ordered goose SQL under `apps/api/migrations/`, applied at boot by `tronc/migrate` v0.1.1 behind a Postgres advisory lock. GORM `AutoMigrate` and the hand-written `CREATE TABLE IF NOT EXISTS` for the porte tables, `api_token_scopes` and `mcp_actions` are gone from the boot path; the GORM structs stay as the query layer. `scripts/check.sh` fails on an `.AutoMigrate(` call outside a test file.
- Database: `00001_baseline.sql` is production's schema as `pg_dump --schema-only` printed it on 2026-11-13, the unused `antenne_deliveries` table included. A database built before this release (Nuage's tables, no applied migration) is stamped at version 1 on its first boot with tronc/migrate's `baseline` command, so the baseline is recorded without running. A database built from the migrations dumps identical to production. `apps/api/schema.sql` is the normalized dump of the whole set.
- Boot: the data steps that ran at every boot still run at every boot, after the migrations and in the same order (`schemas.Reconcile`, one transaction under advisory lock 4919): the avatar, media type and plan payer backfills, the first administrator, the local identity re-key, the password and OIDC subject adoption and the Registre picture backfill, then the colour backfill. A failed migration exits 1 before the listener binds.
- Rollback: going back to v0.17.x drops the `00002` defaults again, because AutoMigrate removes a default its struct tags do not declare, while the ledger still records version 2. Before deploying this release again after such a rollback, run `DELETE FROM goose_db_version WHERE version_id = 2`.
- build: `github.com/FacileStudio/tronc/migrate` v0.1.1 with `pressly/goose/v3` v3.27.3; goose raises `segmentio/asm` from v1.1.3 to v1.2.1.

### Added

- `docker run <image> migrate status` (also `up`, `version`, `baseline <n>`) answers without a shell.
- Database: `00002_column_defaults.sql` gives `created_at`, `updated_at` and `joined_at` a `now()` default on every table that has them, and the users columns porte/kit does not write the values `OnCreate` writes (`''` and `0001-01-01 00:00:00+00`), so a row inserted outside GORM no longer gets NULL there. It has a Down. The kit's `Unwritten()` is now `color` and `oidc_subject`.
- Tests: `TestGORMModelsMatchTheMigratedSchema` fails when a struct names a column the migrations do not create or a type its field cannot hold; `TestPorteTablesMatchPorteSchema` and `TestMCPPendingSchemaIsInTheMigrations` fail when a porte or tronc bump changes a table the baseline holds; the stamping tests cover a database built by AutoMigrate, with and without the ledger `migrate status` creates.

### Removed

- Boot: the pre-porte conversions no longer run: the `api_tokens` primary key repair, the `shares.shared_with` drop, and the copy of the `sessions` and `api_tokens` tables into `porte_sessions` before dropping them. Production has none of those tables or columns (checked 2026-11-13), and a database built from the migrations never has them.

## [0.17.5] - 2026-11-12

### Security

- fix(auth): an account re-created for a person Registre erased is erased again on the next deletion push or poll, without a restart (porte v0.19.0, `directory.Deletions.Holds`). Before, a bearer token minted before the erasure, or an MCP first call, could bring the account back and it stayed until the next restart.
- fix(auth): sign-in, bearer and MCP provisioning, and profile sync refuse an identity Nuage already erased (`oidc.Deps.Erased`, read from the deletion receiver's saved state), and the Registre space sync no longer provisions an erased member from a stale snapshot.
- build: porte v0.19.0 and porte/directory v0.5.0, which also raise go-chi/chi to v5.5.0.

## [0.17.4] - 2026-11-11

### Fixed

- fix(brand): favicons, touch and manifest icons, the link preview image and `site.webmanifest` come from muse v0.19.2's brand generator, and `app.html` carries its head block: the catalog title "Nuage · File storage for the studio", its description, theme colours and Open Graph and Twitter tags. The landing page no longer overrides the title and description, so every locale shows the catalog's.

## [0.17.3] - 2026-11-11

### Security

- Go modules past their advisories: `github.com/go-jose/go-jose/v4` 4.1.9 and `github.com/klauspost/compress` 1.20.4.
- Client dependencies past their high advisories: `pdfjs-dist` 6.5.438, `@sveltejs/kit` 2.70.4 and `vite` 7.3.8, with overrides pinning `devalue` 5.9.4, `nanoid` 3.3.20, `postcss` 8.5.32 and `source-map-js` 1.2.2. The PDF viewer renders, selects text and finds matches on pdf.js 6 unchanged.

### Changed

- `scripts/check.sh` runs the Go tests the way CI does: one package at a time (`-p 1`, `-count=1`), with a 40-minute timeout for `./tests`. Before, `schemas` and `tests` ran in parallel against the one test database and failed locally, and `./tests` (about 12 minutes) hit the 10-minute default timeout.

## [0.17.2] - 2026-11-11

### Security

- `GET /api/users` and `GET /api/users/{id}` show only the caller and the people it shares a space with. Any other id answers 404. Before, every signed-in account read every account's name and email. An administrator's own session (`is_admin`, no token scope, not delegated) still sees every account.
- `POST /api/spaces/{id}/members` takes a `user_id` only for someone the caller already shares a space with, and now also takes an exact `email` for anyone else. Before, an owner could add guessed ids to its own space and read their addresses in the member list.
- The member picker lists the people you already share a space with and adds anyone else by their full email address.

## [0.17.1] - 2026-11-11

### Changed

- The MCP server and its confirm page run on tronc v0.22.0's shared app kit (`mcp.NewApp`, `mcp.PendingPage`, `mcp.PendingRoutes`, `mcp.Digest`): the local list, confirm and dismiss handlers and their types are gone. Tool names, tool schemas and the `/api/mcp/pending` bodies are unchanged, and actions opened before the upgrade still confirm. The OpenAPI list schema is now named `PendingList_PendingItem`, and `PendingResult` documents its optional `url`.
- Account erasure deletes pending MCP actions through `mcp.ErasePending`, in the same transaction.
- tronc v0.22.0. Go 1.26.11 with the matching builder image, which clears the Go advisories GO-2026-6848 to GO-2026-6859 that douane could not score.

## [0.17.0] - 2026-11-10

### Added

- The user menu shows the person's pending Registre space invitations, with a count and a link to accept them (`GET /api/me/invitations`, porte/kit, muse v0.18.0). It is mounted with `OIDC_ISSUER` and never fails a page: an unreachable Registre answers `{"count":0}`.

### Changed

- Sign-in goes through porte/kit's user store. The stored address is now rewritten at a sign-in only when Registre marks it verified, the rule every suite app follows. New accounts are filled exactly as before: colour, empty profile fields, `created_at`.
- GDPR erasure runs in one transaction through porte/kit: personal files and folders, app folders, memberships, shares, quota, billing offers, Nook deliveries naming the address, pending MCP actions, then the account. A failure rolls everything back, and stored bytes are removed only after the commit. A Registre notice matches the identity with or without the issuer's trailing slash. A space whose only owner is erased gets its longest-standing member as owner (spaces Registre does not manage); a space the person was alone in is kept with its files.
- The `/api` rate limit is tronc's token bucket: a burst of 600, refilled at 10 per second, keyed by the visitor's address as the trusted proxy reported it (IPv6 by /64) and bounded at 50,000 visitors. A refusal carries `Retry-After` in seconds; the `X-RateLimit-*` headers, which no client read, are gone.
- Client: the fetch kernel and the theme store come from `@facile/client` v0.1.0. The CSRF header is sent on mutations only, and a 401 outside the session probe returns to the sign-in page. The stored theme choice is kept.
- tronc v0.21.0, porte v0.18.2 (`directory` v0.4.0), muse v0.18.0. Go 1.26.10 with the matching builder image, `golang.org/x/net` v0.62.0 and `golang.org/x/image` v0.45.0, which clear the critical advisories douane reported.
- A test pins the upload, download, share, trash and quota answers byte for byte (`tests/contract_test.go`), so the web client, nuage-cli and the suite apps keep the same contract.
- Quality gate: filet `preset: facile` at v0.19.1 in CI and as the lefthook pre-push step (`filet gate .`), douane 0.7.1 in CI with `-fail critical -scope prod`. `.githooks/` is replaced by `lefthook.yml`.
- Licensed under the GNU Affero General Public License v3.0 only (`AGPL-3.0-only`). See `LICENSE`.

## [0.16.0] - 2026-11-10

### Added

- MCP server at `POST /mcp` (MCP 2026-07-28, stateless) on tronc/mcp v0.19.0, for any MCP client holding a Registre token bound to `https://nuage.facile.studio/mcp`. Ten tools: `nuage_list_folder`, `nuage_search`, `nuage_read_file` (UTF-8 text, 60,000 bytes per call with `next_offset`), `nuage_save_file` (text or base64, 5 MB, new file or new version), `nuage_create_folder`, `nuage_move`, `nuage_delete`, `nuage_create_share`, `nuage_revoke_share` and `nuage_list_shares`. They call the REST routes' own service methods, so spaces and ownership hold exactly as in the API, and delegated tokens follow the same actor rule.
- Moving to the trash and creating a public link are confirmed by the person: through the client (MRTR elicitation) when it can ask, otherwise on the new `/pending` page, which `GET /api/mcp/pending`, `POST /api/mcp/pending/{id}/confirm` and `POST /api/mcp/pending/{id}/dismiss` serve. A `/pending` link survives sign-in.
- `MCP_RESOURCE` (compose default `https://nuage.facile.studio/mcp`) turns the server on; the confirmation key derives from `PRESIGN_SECRET`.

### Changed

- tronc v0.19.0, porte v0.16.0 and porte/mcpauth v0.2.0. A Registre user's first MCP call creates their Nuage account through the OIDC kit, as a first web login would, so nobody has to open Nuage on the web first. Boot creates the `mcp_actions` table (additive).

## [0.15.0] - 2026-11-10

### Added

- Ideas: a lightbulb entry at the top of the sidebar and phone bar, with a badge for new ideas, and an `/ideas` page where Bulle suggests what to tidy, find or share in the active space. An idea runs in place: Nuage polls the run, shows an approval card for each held call, then the result with its links and a link to continue in Bulle. Ideas are paid (`ai.access`): a free plan sees the upsell and Registre's plans link.
- API: `GET /api/ideas`, `POST /api/ideas/generate`, `POST /api/ideas/{id}/run`, `POST /api/ideas/{id}/dismiss`, `GET /api/ideas/runs/{run}` and `POST /api/ideas/runs/{run}/approvals/{call}`, mounted when `OIDC_ISSUER` and the new `BULLE_URL` are set. They call Bulle as the signed-in user through a Registre token exchange (audience `bulle`, scopes `bulle.read` and `bulle.write`). `space_id` must be a space the caller belongs to and that Registre knows. Nuage declares `ai.access` as a plan feature and answers 402 with the upgrade link, or `ai_access: false` on the list, without calling Bulle when the plan lacks it.

### Changed

- porte to v0.15.1 (`porte/exchange`) and tronc to v0.18.0 (`tronc/bulle`). Client: `@facile/muse` v0.16.1 to v0.17.1.

## [0.14.3] - 2026-11-07

### Changed

- Client: `@facile/muse` v0.14.4 to v0.16.1. Secondary buttons use the `secondary` variant in place of the deprecated `outline`. The icon generator accepts Iconify aliases, which the new muse icons need.
- `GET /version` comes from the `VERSION` file, which the release commit bumps (still `v` plus the number). The `APP_VERSION` build arg and the Dokploy env no longer take part, so a stale env value cannot make the server report an old release. The image build no longer needs `.git`.

## [0.14.2] - 2026-11-07

### Changed

- Client: `@facile/muse` v0.14.2 to v0.14.4. The language chosen in the user menu now re-renders the whole UI without a reload, and the menu clamps itself at the screen edge.

## [0.14.1] - 2026-11-07

### Security

- Registre tokens are sealed at rest. porte to v0.13.0: `porte_identities.access_token` and `refresh_token`
  are encrypted with AES-256-GCM under `PORTE_TOKEN_KEY`. Existing rows are sealed once, in the background,
  after boot, and the count is logged. An invalid `PORTE_TOKEN_KEY` stops the boot. With the key unset nothing changes.
- Losing `PORTE_TOKEN_KEY` makes the stored tokens unreadable: people sign in again, nothing else is lost.
  Do not roll back to a build older than porte v0.13.0 after the first reseal.

## [0.14.0] - 2026-11-07

### Removed

- **Breaking:** `PUT /api/spaces/by-facile-id/{facile_id}` and the find-or-create behind it. Restricted tokens can no longer create spaces, and the route is no longer on their allowlist. Spaces come from `POST /api/spaces` or Registre.
- `POST /api/users/me/api-token` refuses `provision: true` with 400. A restricted token needs `app` and `space_facile_ids`. A token minted with `provision` before this release keeps reaching the spaces its app created, and can no longer create any.

## [0.13.2] - 2026-11-07

### Changed

- `@facile/muse` v0.14.1 to v0.14.2: the user menu clamps itself inside the viewport at the right edge of a header.
- The user menu's Subscription row always shows and opens `https://sso.facile.studio/spaces`, the same address as the plan upgrade link.
- API tokens: the create drawer has no expiry field and the list has no expiry or last-used column, because Nuage tokens carry neither.

## [0.13.1] - 2026-11-07

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
