package schemas

import (
	"strings"

	"gorm.io/gorm"
)

// AdoptPorte keeps Nuage's credentials on porte's identity table. It changes
// rows only: the porte tables themselves come from the migrations.
//
// issuer is the configured OIDC_ISSUER, and it has to be passed in because it
// is half of the account matching key: porte finds an identity by (provider,
// subject), and the provider is the issuer. Backfilling with the wrong value —
// or with a placeholder — would leave every existing SSO user unmatched, which
// degrades silently to the email fallback rather than failing.
//
// An empty issuer skips only the identity backfill. The local re-key and the
// passwords still move, because they are what keeps people able to sign in
// again.
func AdoptPorte(db *gorm.DB, issuer string) error {
	statements := []string{rekeyLocalSubjects, adoptExistingPasswords}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	if issuer == "" {
		return nil
	}
	if err := db.Exec(adoptOIDCSubjects, issuer).Error; err != nil {
		return err
	}
	return backfillPictureURLs(db, issuer)
}

// backfillPictureURLs points every SSO account at its Registre picture without
// waiting for a sign-in.
//
// Registre serves each person's current picture at the stable URL
// <issuer>/avatars/<subject> and answers 404 when there is none, which the
// client renders as initials. Accounts that last signed in before the claim
// existed hold nothing, or a copy from an earlier identity provider, so they
// showed initials until their next login. The URL is derived from the
// (provider, subject) row porte already keeps, so the update is idempotent and
// a row already on the right URL is left alone. A non-https issuer is skipped,
// matching kit.PictureURL.
func backfillPictureURLs(db *gorm.DB, issuer string) error {
	base := strings.TrimRight(issuer, "/")
	if !strings.HasPrefix(strings.ToLower(base), "https://") {
		return nil
	}
	return db.Exec(pictureURLBackfill, base, issuer, base).Error
}

const pictureURLBackfill = `
UPDATE users SET oidc_picture_url = ? || '/avatars/' || i.subject
  FROM porte_identities i
 WHERE i.user_id = users.id
   AND i.provider = ?
   AND coalesce(users.oidc_picture_url, '') NOT LIKE ? || '/avatars/%';
`

// rekeyLocalSubjects is porte v0.3.0's re-key of the local subject onto the
// user id. It lives in porte/pg's own Schema, which nothing here runs: Nuage's
// porte tables are in its migrations, repointed at its own users. Without it
// every account that adopted porte before v0.3.0 keeps an identity keyed on
// its address, porte/local looks one up by account id, and the login form
// answers "invalid credentials" to the right password.
const rekeyLocalSubjects = `
UPDATE porte_identities SET subject = user_id::text
 WHERE provider = 'local' AND subject <> user_id::text;
`

// adoptExistingPasswords moves the argon2 hashes from users.password_hash into
// the identity rows porte/local reads.
//
// Without it the deploy silently ends password login for every existing
// account: the hash is still in the users table, nothing reads it there any
// more, and the login form answers "invalid credentials" to a correct
// password with no error anywhere. The hashes are byte-identical — porte/local
// uses the parameters this app already used — so the move is a copy and nobody
// resets anything.
//
// The subject is the account id, which is what porte.LocalSubject returns.
// v0.2 keyed it on the lowercased address, and that key moved every time
// somebody edited their profile, so this app carried an UPDATE against
// porte_identities to chase it. The id does not move, so the chase is gone —
// and writing the address here would re-introduce a row the re-key above has
// already run past.
//
// users.password_hash is deliberately left in place. Blanking it in the same
// deploy makes the change unrollbackable for the sake of tidiness, and a
// column nothing reads can be dropped on any later day.
const adoptExistingPasswords = `
INSERT INTO porte_identities (user_id, provider, subject, password_hash)
SELECT id, 'local', id::text, password_hash
  FROM users
 WHERE coalesce(password_hash, '') <> ''
ON CONFLICT (provider, subject) DO NOTHING;
`

// adoptOIDCSubjects moves the federated identity off the user row and into the
// identity table porte reads.
//
// Without it nothing breaks loudly: porte finds no identity for (issuer,
// subject), falls back to matching the verified email, finds the same user and
// links it on the next login. But the email fallback is the weaker path by
// design — it is the one an identity provider that lets a user set any address
// can abuse — and relying on it for every existing account, on the one deploy
// where it would go unnoticed, is not a trade worth making.
//
// The OIDC tokens deliberately do **not** come across. Nuage stores them
// encrypted with ENCRYPTION_KEY and porte stores them as it will send them, so
// copying the ciphertext across would hand porte a refresh token that is not
// one: the first profile sync would fail, clear both, and look like the
// provider had revoked them. They are re-issued in full on the user's next SSO
// login, and the only thing missing until then is a background profile
// refresh.
const adoptOIDCSubjects = `
INSERT INTO porte_identities (user_id, provider, subject)
SELECT id, ?, oidc_subject
  FROM users
 WHERE oidc_subject IS NOT NULL AND oidc_subject <> ''
ON CONFLICT (provider, subject) DO NOTHING;
`
