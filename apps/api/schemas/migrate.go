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

func Migrate(db *gorm.DB) error {
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", migrationLockID).Error; err != nil {
			return err
		}
		return migrateSchema(tx)
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
		&Session{},
		&ApiToken{},
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
		&NookDelivery{},
		&Tombstone{},
	); err != nil {
		return err
	}
	return ensureAdmin(db)
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

	return nil
}
