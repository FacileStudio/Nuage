package schemas

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/internal/usercolor"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := preMigrate(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&User{},
		&Session{},
		&ApiToken{},
		&File{},
		&Folder{},
		&Share{},
		&Setting{},
		&FileVersion{},
		&UploadSession{},
		&UploadChunk{},
		&UserQuota{},
		&ActivityLog{},
	); err != nil {
		return err
	}
	return usercolor.BackfillMissing(context.Background(), db)
}

func preMigrate(db *gorm.DB) error {
	if db.Migrator().HasTable("api_tokens") {
		db.Exec(`
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
		`)
	}

	if db.Migrator().HasTable("shares") {
		db.Exec(`
			DO $$ BEGIN
				IF EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = 'shares' AND column_name = 'shared_with'
				) THEN
					DROP INDEX IF EXISTS idx_shares_shared_with;
					ALTER TABLE shares DROP COLUMN shared_with;
				END IF;
			END $$;
		`)
	}

	return nil
}
