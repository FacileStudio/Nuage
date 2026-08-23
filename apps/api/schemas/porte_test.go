package schemas

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

const testIssuer = "https://porte.test/application/o/nuage/"

// seedPrePorte rebuilds the shape production is in before this deploy: the old
// sessions table, the separate api_tokens table, a federated identity recorded
// on the user row, and a local password hash in users.password_hash.
//
// The legacy tables are created here in SQL because the models are gone. That
// is the point: after this migration they exist only in databases that predate
// it, and the only thing that still has to understand them is AdoptPorte.
func seedPrePorte(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`DROP TABLE IF EXISTS sessions`,
		`DROP TABLE IF EXISTS api_tokens`,
		`CREATE TABLE sessions (token text PRIMARY KEY, user_id bigint NOT NULL, expires_at timestamptz, created_at timestamptz)`,
		`CREATE TABLE api_tokens (token text PRIMARY KEY, user_id bigint NOT NULL, name text, created_at timestamptz)`,
		`INSERT INTO users (id, email, name, oidc_subject, oidc_access_token, oidc_refresh_token, profile_synced_at, password_hash, created_at)
		 VALUES (1, 'camille@facile.studio', 'Camille', 'sub-1', 'ciphertext', 'ciphertext', now(), '', now())`,
		`INSERT INTO users (id, email, name, oidc_subject, password_hash, created_at)
		 VALUES (2, 'Noah@Facile.Studio', 'Noah', NULL, '$argon2id$fake', now())`,
		`INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES
			('live', 1, now() + interval '10 days', now() - interval '40 days'),
			('dead', 1, now() - interval '1 day', now() - interval '31 days')`,
		`INSERT INTO api_tokens (token, user_id, name, created_at) VALUES ('cli-token', 2, 'CLI', now())`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, statement)
		}
	}
	t.Cleanup(func() {
		db.Exec(`DROP TABLE IF EXISTS sessions`)
		db.Exec(`DROP TABLE IF EXISTS api_tokens`)
		db.Exec(`DELETE FROM users`)
	})
}

// Nobody may be signed out by this deploy. Both tables store the SHA-256 hex of
// a token and nothing else, which is exactly what porte stores, so the rows
// move and the cookie already in somebody's browser keeps authenticating.
//
// Carrying created_at over would put this session 40 days into the
// seven-day idle window and sign the user out on the deploy meant to
// keep them. The API token becomes a labelled session with no expiry,
// which is what porte.Session.Label exists for. It is issued once and
// never re-issued, so a row that does not survive is a credential the
// holder cannot get back.
func TestAdoptPorteKeepsEverybodySignedIn(t *testing.T) {
	db := openTestDatabase(t)
	seedPrePorte(t, db)

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	var carried struct {
		UserID     int64
		Label      string
		LastUsedAt time.Time
	}
	if err := db.Raw(`SELECT user_id, label, last_used_at FROM porte_sessions WHERE token_hash = 'live'`).Scan(&carried).Error; err != nil {
		t.Fatalf("read the carried session: %v", err)
	}
	if carried.UserID != 1 || carried.Label != "" {
		t.Fatalf("the browser session did not survive as an unlabelled session: %+v", carried)
	}
	if time.Since(carried.LastUsedAt) > time.Hour {
		t.Fatalf("last_used_at was copied instead of stamped: %v", carried.LastUsedAt)
	}

	var expired int64
	if err := db.Raw(`SELECT count(*) FROM porte_sessions WHERE token_hash = 'dead'`).Scan(&expired).Error; err != nil {
		t.Fatalf("count expired: %v", err)
	}
	if expired != 0 {
		t.Fatal("an already-expired session was carried over")
	}

	var token struct {
		UserID    int64
		Label     string
		ExpiresAt *time.Time
	}
	if err := db.Raw(`SELECT user_id, label, expires_at FROM porte_sessions WHERE token_hash = 'cli-token'`).Scan(&token).Error; err != nil {
		t.Fatalf("read the carried api token: %v", err)
	}
	if token.UserID != 2 || token.Label != "CLI" {
		t.Fatalf("the api token did not survive: %+v", token)
	}
	if token.ExpiresAt != nil {
		t.Fatalf("the api token was given an expiry: %v", token.ExpiresAt)
	}

	for _, table := range []string{"sessions", "api_tokens"} {
		var remaining *string
		if err := db.Raw(`SELECT to_regclass(?)::text`, table).Scan(&remaining).Error; err != nil {
			t.Fatalf("check %s: %v", table, err)
		}
		if remaining != nil {
			t.Fatalf("the legacy %s table survived", table)
		}
	}

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("adopt is not idempotent: %v", err)
	}
}

// The password hash moves into the identity row porte/local reads. Without it
// the login form answers "invalid credentials" to a correct password, with the
// hash still sitting in the users table and no error anywhere.
//
// The subject is the account id, which is porte.LocalSubject. v0.2 keyed it on
// the lowercased address, and this user registered with a mixed-case one — so
// an adoption that still wrote the address would leave a row porte cannot find
// *and* one the re-key in porteSchema has already run past.
func TestAdoptPorteMovesThePasswords(t *testing.T) {
	db := openTestDatabase(t)
	seedPrePorte(t, db)

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	var identity struct {
		UserID       int64
		PasswordHash string
	}
	err := db.Raw(
		`SELECT user_id, password_hash FROM porte_identities WHERE provider = 'local' AND subject = '2'`,
	).Scan(&identity).Error
	if err != nil {
		t.Fatalf("read the local identity: %v", err)
	}
	if identity.UserID != 2 || identity.PasswordHash != "$argon2id$fake" {
		t.Fatalf("the password did not move: %+v", identity)
	}

	var withoutPassword int64
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE provider = 'local' AND user_id = 1`).Scan(&withoutPassword).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if withoutPassword != 0 {
		t.Fatal("an account with no password gained a local identity, which is a login that cannot be used and an account that cannot be registered")
	}
}

// TestAdoptPorteReKeysAnAddressKeyedIdentity is the migration porte v0.3.0
// ships and this app does not receive.
//
// The re-key lives in porte/pg's own Schema, which nothing here calls: Nuage
// runs its own copy of that schema against its own users table. Left uncopied,
// every account that adopted porte before this deploy keeps an identity keyed
// on its address, porte/local looks one up by account id, finds nothing, and
// the login form answers "invalid credentials" to the right password with no
// error anywhere.
//
// The second run is not a formality. AdoptPorte runs on every boot, so a
// migration that is not idempotent breaks the instance that restarts.
//
// The database is put back into the shape v0.2 left it in — the local identity
// keyed on the lowercased address, exactly as the old adoptExistingPasswords
// wrote it — and the federated identity is then checked to have stayed put:
// re-keying that one would orphan the SSO login instead of repairing it.
func TestAdoptPorteReKeysAnAddressKeyedIdentity(t *testing.T) {
	db := openTestDatabase(t)
	seedPrePorte(t, db)

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("adopt: %v", err)
	}
	if err := db.Exec(
		`UPDATE porte_identities SET subject = lower(btrim((SELECT email FROM users WHERE id = 2)))
		  WHERE provider = 'local' AND user_id = 2`).Error; err != nil {
		t.Fatalf("rebuild the v0.2 shape: %v", err)
	}

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("re-key: %v", err)
	}

	var subject string
	if err := db.Raw(`SELECT subject FROM porte_identities WHERE provider = 'local' AND user_id = 2`).Scan(&subject).Error; err != nil {
		t.Fatalf("read the local identity: %v", err)
	}
	if subject != "2" {
		t.Fatalf("the local identity is still keyed on %q, so this password cannot sign in any more", subject)
	}

	var locals int64
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE provider = 'local'`).Scan(&locals).Error; err != nil {
		t.Fatalf("count the local identities: %v", err)
	}
	if locals != 1 {
		t.Fatalf("expected one local identity, got %d — a second password on one account is what the old key made reachable", locals)
	}

	var federated string
	if err := db.Raw(`SELECT subject FROM porte_identities WHERE provider = ?`, testIssuer).Scan(&federated).Error; err != nil {
		t.Fatalf("read the federated identity: %v", err)
	}
	if federated != "sub-1" {
		t.Fatalf("the federated subject was re-keyed to %q", federated)
	}

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("the re-key is not idempotent: %v", err)
	}
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE provider = 'local'`).Scan(&locals).Error; err != nil {
		t.Fatalf("recount the local identities: %v", err)
	}
	if locals != 1 {
		t.Fatalf("a second run left %d local identities", locals)
	}
}

// The federated identity moves off the user row. Without it porte finds no
// identity, falls back to matching the verified email and relinks on the next
// login — which works, but leans the whole existing user base on the weaker of
// the two matching paths, on the one deploy where nobody would notice.
//
// Nuage encrypts the provider tokens with ENCRYPTION_KEY and porte
// stores them as it will send them, so the ciphertext deliberately
// stays behind: handing porte a refresh token that is not one makes the
// first profile sync fail and look like the provider revoked it.
func TestAdoptPorteMovesTheOIDCSubject(t *testing.T) {
	db := openTestDatabase(t)
	seedPrePorte(t, db)

	if err := AdoptPorte(db, testIssuer); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	var identity struct {
		UserID       int64
		AccessToken  string
		RefreshToken string
	}
	err := db.Raw(
		`SELECT user_id, access_token, refresh_token FROM porte_identities WHERE provider = ? AND subject = 'sub-1'`,
		testIssuer,
	).Scan(&identity).Error
	if err != nil {
		t.Fatalf("read the identity: %v", err)
	}
	if identity.UserID != 1 {
		t.Fatal("the oidc subject was not adopted")
	}
	if identity.AccessToken != "" || identity.RefreshToken != "" {
		t.Fatalf("encrypted provider tokens were carried across: %+v", identity)
	}

	var rows int64
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE provider = ?`, testIssuer).Scan(&rows).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected exactly one federated identity, got %d", rows)
	}
}

// An empty issuer is a deployment with SSO switched off. The sessions, the API
// tokens and the passwords still have to move — they are what keeps people
// signed in and able to sign in — but there is no provider to key a federated
// identity against.
func TestAdoptPorteWithoutAnIssuerStillMovesTheCredentials(t *testing.T) {
	db := openTestDatabase(t)
	seedPrePorte(t, db)

	if err := AdoptPorte(db, ""); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	var sessions, federated int64
	if err := db.Raw(`SELECT count(*) FROM porte_sessions`).Scan(&sessions).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if err := db.Raw(`SELECT count(*) FROM porte_identities WHERE provider <> 'local'`).Scan(&federated).Error; err != nil {
		t.Fatalf("count identities: %v", err)
	}
	if sessions != 2 {
		t.Fatalf("expected the live session and the api token, got %d", sessions)
	}
	if federated != 0 {
		t.Fatalf("an identity was keyed against no provider: %d rows", federated)
	}
}
