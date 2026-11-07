package erasure

import (
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

const (
	reassignActivity = `UPDATE activity_logs SET user_id = @ghost WHERE user_id = @id AND (
		(resource_type = 'file' AND resource_id IN (SELECT id FROM files WHERE space_id IS NOT NULL))
		OR (resource_type = 'folder' AND resource_id IN (SELECT id FROM folders WHERE space_id IS NOT NULL)))`
	deleteActivity    = `DELETE FROM activity_logs WHERE user_id = @id`
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
	deleteUser        = `DELETE FROM users WHERE id = @id`
)

var accountSteps = []string{
	reassignActivity, deleteActivity, deleteShares, reassignFiles, reassignVersions, reassignFolders,
	reassignUploads, reassignTombstone, deleteTombstones, deleteQuota, releaseCharges, releaseOffers,
	deleteWebhooks, deleteUser,
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
	return nil
}

func deletedUser(tx *gorm.DB) (int64, error) {
	placeholder := schemas.User{Email: schemas.DeletedUserEmail, Name: "Deleted user", Color: "AD9EF0"}
	err := tx.Where("email = ?", schemas.DeletedUserEmail).FirstOrCreate(&placeholder).Error
	return placeholder.ID, err
}
