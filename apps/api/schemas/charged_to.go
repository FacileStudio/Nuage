package schemas

import (
	"log/slog"

	"gorm.io/gorm"
)

// chargedToBackfillSQL charges every app-provisioned space created before the
// charged_to column to its earliest owner, and returns what it set.
const chargedToBackfillSQL = `UPDATE spaces s SET charged_to = o.user_id
	FROM (SELECT DISTINCT ON (space_id) space_id, user_id FROM space_members
		WHERE role = 'owner' ORDER BY space_id, joined_at, id) o
	WHERE s.id = o.space_id AND s.provisioned_by <> ''
		AND (s.charged_to IS NULL OR NOT EXISTS (SELECT 1 FROM users u WHERE u.id = s.charged_to))
	RETURNING s.id, s.provisioned_by, s.charged_to`

// backfillChargedTo gives each provisioned space that predates charged_to, or
// whose payer's account was deleted, its earliest remaining owner as the
// person whose plan pays for it, and logs every space it charged. A space with
// no owner left keeps no valid payer and is refused new files once plans
// apply. It matches nothing once every row is set.
func backfillChargedTo(db *gorm.DB) error {
	var charged []struct {
		ID            int64
		ProvisionedBy string
		ChargedTo     int64
	}
	if err := db.Raw(chargedToBackfillSQL).Scan(&charged).Error; err != nil {
		return err
	}
	for _, row := range charged {
		slog.Info("forfait: provisioned space charged to its earliest owner",
			slog.Int64("space_id", row.ID), slog.String("app", row.ProvisionedBy), slog.Int64("user_id", row.ChargedTo))
	}
	return nil
}

// backfill runs every data backfill that follows the schema migration, in
// order. Each one matches nothing once its rows are fixed.
func backfill(db *gorm.DB) error {
	for _, step := range []func(*gorm.DB) error{backfillAvatarUploadPath, backfillMimeTypes, backfillChargedTo} {
		if err := step(db); err != nil {
			return err
		}
	}
	return nil
}
