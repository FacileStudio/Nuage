package trash

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// ErasePersonal deletes for good every file and folder userID keeps outside a
// space, trashed ones included, and removes their stored objects. Space content
// is not touched: it belongs to the space. Quota is not refunded because the
// caller deletes the account. Running it again finds nothing and does nothing.
func (s *Service) ErasePersonal(ctx context.Context, userID int64) error {
	var keys []storedObject
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var files []schemas.File
		if err := tx.Where("uploaded_by = ? AND space_id IS NULL", userID).Find(&files).Error; err != nil {
			return err
		}
		purged, _, err := purgeFilesTx(ctx, tx, files)
		if err != nil {
			return err
		}
		keys = purged
		return purgePersonalFolders(ctx, tx, userID)
	})
	if err != nil {
		return errors.Internal("failed to erase the personal files", err)
	}
	s.deleteObjects(ctx, keys)
	return nil
}

func purgePersonalFolders(ctx context.Context, tx *gorm.DB, userID int64) error {
	var ids []int64
	if err := tx.Model(&schemas.Folder{}).
		Where("owner_id = ? AND space_id IS NULL", userID).Pluck("id", &ids).Error; err != nil {
		return err
	}
	return purgeFoldersTx(ctx, tx, ids)
}
