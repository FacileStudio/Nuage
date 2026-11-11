package erasure

import (
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	mcpkit "github.com/FacileStudio/tronc/mcp"

	"gorm.io/gorm"
)

const (
	reassignActivity = `UPDATE activity_logs SET user_id = @ghost WHERE user_id = @id AND (
		(resource_type = 'file' AND resource_id IN (SELECT id FROM files WHERE space_id IS NOT NULL))
		OR (resource_type = 'folder' AND resource_id IN (SELECT id FROM folders WHERE space_id IS NOT NULL)))`
	deleteActivity   = `DELETE FROM activity_logs WHERE user_id = @id`
	promoteSuccessor = `UPDATE space_members SET role = 'owner' WHERE id IN (
		SELECT DISTINCT ON (m.space_id) m.id FROM space_members m JOIN spaces s ON s.id = m.space_id
		WHERE m.user_id <> @id AND s.registre_space_id IS NULL
		  AND m.space_id IN (SELECT space_id FROM space_members WHERE user_id = @id AND role = 'owner')
		  AND NOT EXISTS (SELECT 1 FROM space_members o
		                  WHERE o.space_id = m.space_id AND o.user_id <> @id AND o.role = 'owner')
		ORDER BY m.space_id, m.joined_at, m.id)`
	deleteMemberships = `DELETE FROM space_members WHERE user_id = @id`
	deleteShares      = `DELETE FROM shares WHERE shared_by = @id`
	reassignFiles     = `UPDATE files SET uploaded_by = @ghost WHERE uploaded_by = @id`
	reassignVersions  = `UPDATE file_versions SET created_by = @ghost WHERE created_by = @id`
	reassignFolders   = `UPDATE folders SET owner_id = @ghost WHERE owner_id = @id`
	reassignUploads   = `UPDATE upload_sessions SET user_id = @ghost WHERE user_id = @id`
	reassignTombstone = `UPDATE tombstones SET user_id = @ghost WHERE user_id = @id AND space_id IS NOT NULL`
	deleteTombstones  = `DELETE FROM tombstones WHERE user_id = @id`
	deleteQuota       = `DELETE FROM user_quotas WHERE user_id = @id`
	releaseCharges    = `UPDATE spaces SET charged_to = NULL WHERE charged_to = @id`
	releaseOffers     = `UPDATE spaces SET billing_offer_to = NULL WHERE billing_offer_to = @id`
	deleteWebhooks    = `DELETE FROM nook_deliveries WHERE position(@email in payload) > 0`
)

var accountSteps = []string{
	reassignActivity, deleteActivity, promoteSuccessor, deleteMemberships, deleteShares,
	reassignFiles, reassignVersions, reassignFolders, reassignUploads, reassignTombstone,
	deleteTombstones, deleteQuota, releaseCharges, releaseOffers, deleteWebhooks,
}

func deleteAccount(tx *gorm.DB, found account) error {
	ghost, err := deletedUser(tx)
	if err != nil {
		return err
	}
	args := map[string]any{"id": found.ID, "ghost": ghost, "email": found.Email}
	for _, statement := range accountSteps {
		if err := tx.Exec(statement, args).Error; err != nil {
			return err
		}
	}
	return mcpkit.ErasePending(tx.Statement.Context, tx.Statement.ConnPool, strconv.FormatInt(found.ID, 10))
}

func deletedUser(tx *gorm.DB) (int64, error) {
	placeholder := schemas.User{Email: schemas.DeletedUserEmail, Name: "Deleted user", Color: "AD9EF0"}
	err := tx.Where("email = ?", schemas.DeletedUserEmail).FirstOrCreate(&placeholder).Error
	return placeholder.ID, err
}
