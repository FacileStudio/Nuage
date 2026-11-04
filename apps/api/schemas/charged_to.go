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
	WHERE s.id = o.space_id AND s.provisioned_by <> '' AND s.charged_to IS NULL
	RETURNING s.id, s.provisioned_by, s.charged_to`

// backfillChargedTo gives each provisioned space that predates charged_to the
// earliest owner as the person whose plan pays for it, and logs every space it
// charged. A provisioned space with no owner stays uncharged and is refused new
// files once plans apply. It matches nothing once every row is set.
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
