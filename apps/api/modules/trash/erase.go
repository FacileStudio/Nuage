package trash

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

// ErasePersonal deletes for good, inside tx, every file and folder userID
// keeps outside a space, trashed ones included. Space content is not touched:
// it belongs to the space. Quota is not refunded because the caller deletes
// the account. It returns the removal of the stored objects, which the caller
// runs once tx has committed, so a rolled-back erasure keeps every byte.
func (s *Service) ErasePersonal(ctx context.Context, tx *gorm.DB, userID int64) (func(context.Context), error) {
	var files []schemas.File
	if err := tx.Where("uploaded_by = ? AND space_id IS NULL", userID).Find(&files).Error; err != nil {
		return nil, err
	}
	objects, _, err := purgeFilesTx(ctx, tx, files)
	if err != nil {
		return nil, err
	}
	if err := purgePersonalFolders(ctx, tx, userID); err != nil {
		return nil, err
	}
	return func(ctx context.Context) { s.deleteObjects(ctx, objects) }, nil
}

func purgePersonalFolders(ctx context.Context, tx *gorm.DB, userID int64) error {
	var ids []int64
	if err := tx.Model(&schemas.Folder{}).
		Where("owner_id = ? AND space_id IS NULL", userID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	return purgeFoldersTx(ctx, tx, ids)
}
