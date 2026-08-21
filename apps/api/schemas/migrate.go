package schemas

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"

	"gorm.io/gorm"
)

// migrationLockID namespaces the advisory lock that serialises schema
// migration across concurrently starting instances.
const migrationLockID = 4919

// Migrate brings the schema up to date and then hands authentication to porte.
func Migrate(db *gorm.DB) error {
	return MigrateWithIssuer(db, "")
}

// MigrateWithIssuer is Migrate with the OIDC issuer, which the identity
// backfill needs: porte matches an account on (provider, subject) and the
// provider is the issuer, so backfilling with a placeholder would leave every
// existing SSO user unmatched and quietly fall through to the email path.
//
// AdoptPorte runs inside the same advisory lock as the rest of the schema, so
// two instances starting together do not both try to move the credentials.
func MigrateWithIssuer(db *gorm.DB, issuer string) error {
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", migrationLockID).Error; err != nil {
			return err
		}
		if err := migrateSchema(tx); err != nil {
			return err
		}
		return AdoptPorte(tx, issuer)
	}); err != nil {
		return err
	}
	return usercolor.BackfillMissing(context.Background(), db)
}

func migrateSchema(db *gorm.DB) error {
	if err := preMigrate(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&User{},
		&Space{},
		&SpaceMember{},
		&File{},
		&Folder{},
		&Share{},
		&Setting{},
		&FileVersion{},
		&UploadSession{},
		&UploadChunk{},
		&UserQuota{},
		&ActivityLog{},
		&AntenneDelivery{},
		&Tombstone{},
	); err != nil {
		return err
	}
	if err := backfillAvatarUploadPath(db); err != nil {
		return err
	}
	return ensureAdmin(db)
}

// backfillAvatarUploadPath moves the uploaded avatars onto the column that now owns them,
// and clears the stored copies of the SSO photo that nothing reads any more.
//
// The filename decides, not avatar_source. That column was added after the upload feature,
// so the oldest uploaded avatars have it empty — on Sablier's production database two of
// the four rows were exactly that, and keying on avatar_source = 'upload' would silently
// have dropped their picture. persistAvatarFile has always named uploads "user-<id>-<nanos>"
// and the old OIDC download named its copies "oidc-<id>-<nanos>", so anything that is not
// an oidc- copy is somebody's upload and is kept.
//
// The prefix strip is anchored. Nuage serves avatars from /api/avatars/, not the /files/
// the rest of the suite uses, and an unanchored replace() would mangle any filename that
// happened to contain the prefix.
//
// backfillAvatarUploadPath back-fills avatar_upload_path and cleans
// oidc_picture_url on rows created by older releases.
//
// avatar_url and avatar_source stay in the table, unread, until a later
// release drops them. Expanding first means a rollback is redeploying the
// old binary rather than restoring a backup.
//
// The old code stored profile.Picture verbatim, so every user without a
// photo in Authentik carries a data: URI of their own initials here. Under
// the new rule this column means "there is an SSO photo", so leaving the
// placeholder would suppress the upload fallback for those users forever.
//
// A NULL in a freshly added column would fail to scan into the plain string
// the model declares.
func backfillAvatarUploadPath(db *gorm.DB) error {
	if db.Migrator().HasColumn(&User{}, "avatar_url") {
		if err := db.Exec(
			`UPDATE users SET avatar_upload_path = regexp_replace(avatar_url, '^/api/avatars/', '')
			 WHERE coalesce(avatar_url, '') <> ''
			   AND avatar_url LIKE '/api/avatars/%'
			   AND avatar_url NOT LIKE '/api/avatars/oidc-%'
			   AND coalesce(avatar_upload_path, '') = ''`).Error; err != nil {
			return err
		}
	}

	if err := db.Exec(
		`UPDATE users SET oidc_picture_url = ''
		 WHERE coalesce(oidc_picture_url, '') <> ''
		   AND lower(oidc_picture_url) NOT LIKE 'https://%'`).Error; err != nil {
		return err
	}

	return db.Exec(`UPDATE users SET avatar_upload_path = '' WHERE avatar_upload_path IS NULL`).Error
}

// ensureAdmin promotes the earliest account when no administrator exists, so a
// fresh instance is always manageable. It never runs once an admin is present.
func ensureAdmin(db *gorm.DB) error {
	var adminCount int64
	if err := db.Model(&User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
		return err
	}
	if adminCount > 0 {
		return nil
	}
	var firstUser User
	if err := db.Order("id asc").First(&firstUser).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return db.Model(&firstUser).Update("is_admin", true).Error
}

func preMigrate(db *gorm.DB) error {
	if db.Migrator().HasTable("api_tokens") {
		if err := db.Exec(`
			DO $$ BEGIN
				IF EXISTS (
					SELECT 1 FROM information_schema.table_constraints
					WHERE table_name = 'api_tokens'
					AND constraint_type = 'PRIMARY KEY'
					AND constraint_name IN (
						SELECT constraint_name FROM information_schema.key_column_usage
						WHERE table_name = 'api_tokens' AND column_name = 'token'
					)
				) THEN
					ALTER TABLE api_tokens DROP CONSTRAINT IF EXISTS api_tokens_pkey;
					IF NOT EXISTS (
						SELECT 1 FROM information_schema.columns
						WHERE table_name = 'api_tokens' AND column_name = 'id'
					) THEN
						ALTER TABLE api_tokens ADD COLUMN id BIGSERIAL PRIMARY KEY;
					ELSE
						ALTER TABLE api_tokens ADD PRIMARY KEY (id);
					END IF;
				END IF;
			END $$;
		`).Error; err != nil {
			return err
		}
	}

	if db.Migrator().HasTable("shares") {
		if err := db.Exec(`
			DO $$ BEGIN
				IF EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = 'shares' AND column_name = 'shared_with'
				) THEN
					DROP INDEX IF EXISTS idx_shares_shared_with;
					ALTER TABLE shares DROP COLUMN shared_with;
				END IF;
			END $$;
		`).Error; err != nil {
			return err
		}
	}

	return renameNookToAntenne(db)
}

// renameNookToAntenne carries the delivery queue and its settings over from the
// name the alert bus had before it was renamed to Antenne. The table rename is
// what keeps a pending queue alive across the deploy: without it AutoMigrate
// creates an empty antenne_deliveries beside the old table and every event
// still waiting to be delivered is stranded.
//
// Postgres keeps every dependent object's name across ALTER TABLE ... RENAME,
// so they are renamed one by one. The two indexes have to be: left alone,
// AutoMigrate finds no index under the new name and builds a second copy of
// each on the same columns. The primary key and the identity sequence keep
// working either way and are renamed only so that nothing in the database is
// still named nook_*.
//
// The settings are rows here, not columns as in Sablier: the keys live in
// settings.key, which is the primary key, so an already-renamed row would make
// the update collide with itself rather than be a no-op. The NOT EXISTS guard
// is what makes a second run harmless.
func renameNookToAntenne(db *gorm.DB) error {
	migrator := db.Migrator()

	if migrator.HasTable("nook_deliveries") && !migrator.HasTable("antenne_deliveries") {
		if err := migrator.RenameTable("nook_deliveries", "antenne_deliveries"); err != nil {
			return err
		}
		renames := []string{
			"ALTER INDEX IF EXISTS idx_nook_status_retry RENAME TO idx_antenne_status_retry",
			"ALTER INDEX IF EXISTS idx_nook_deliveries_event_type RENAME TO idx_antenne_deliveries_event_type",
			"ALTER INDEX IF EXISTS nook_deliveries_pkey RENAME TO antenne_deliveries_pkey",
			"ALTER SEQUENCE IF EXISTS nook_deliveries_id_seq RENAME TO antenne_deliveries_id_seq",
		}
		for _, stmt := range renames {
			if err := db.Exec(stmt).Error; err != nil {
				return err
			}
		}
	}

	if !migrator.HasTable(&Setting{}) {
		return nil
	}
	return db.Exec(`
		UPDATE settings SET key = 'antenne_' || substr(key, 6)
		WHERE left(key, 5) = 'nook_'
		  AND NOT EXISTS (
			SELECT 1 FROM settings existing
			WHERE existing.key = 'antenne_' || substr(settings.key, 6)
		  )`).Error
}
