package trash

import (
	"context"
	stderrors "errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/internal/tombstone"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm      *gorm.DB
	storage  *storage.Client
	activity *activity.Logger
	quota    *quota.Service
}

func NewService(orm *gorm.DB, storageClient *storage.Client, actLogger *activity.Logger, quotaService *quota.Service) *Service {
	return &Service{orm: orm, storage: storageClient, activity: actLogger, quota: quotaService}
}

func (s *Service) listTrash(ctx context.Context, userID int64, hasSpace bool, spaceID *int64) ([]TrashItem, error) {
	if hasSpace && spaceID != nil {
		if err := spaceaccess.Require(ctx, s.orm, *spaceID, userID); err != nil {
			return nil, err
		}
	}

	fileQuery := s.orm.WithContext(ctx).Where("uploaded_by = ? AND deleted_at IS NOT NULL", userID)
	folderQuery := s.orm.WithContext(ctx).Where("owner_id = ? AND deleted_at IS NOT NULL", userID)

	if hasSpace {
		if spaceID != nil {
			fileQuery = fileQuery.Where("space_id = ?", *spaceID)
			folderQuery = folderQuery.Where("space_id = ?", *spaceID)
		} else {
			fileQuery = fileQuery.Where("space_id IS NULL")
			folderQuery = folderQuery.Where("space_id IS NULL")
		}
	}

	var files []schemas.File
	if err := fileQuery.Order("deleted_at desc").Find(&files).Error; err != nil {
		return nil, errors.Internal("failed to list trashed files", err)
	}

	var folders []schemas.Folder
	if err := folderQuery.Order("deleted_at desc").Find(&folders).Error; err != nil {
		return nil, errors.Internal("failed to list trashed folders", err)
	}

	items := make([]TrashItem, 0, len(files)+len(folders))
	for _, f := range files {
		items = append(items, TrashItem{
			Type:      "file",
			ID:        f.ID,
			FacileID:  f.FacileID,
			Name:      f.Name,
			MimeType:  f.MimeType,
			Size:      f.Size,
			DeletedAt: f.DeletedAt.UTC().Format(time.RFC3339),
		})
	}
	for _, f := range folders {
		items = append(items, TrashItem{
			Type:      "folder",
			ID:        f.ID,
			FacileID:  f.FacileID,
			Name:      f.Name,
			DeletedAt: f.DeletedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, nil
}

func (s *Service) folderTreeIDs(ctx context.Context, rootID int64) ([]int64, error) {
	var ids []int64
	if err := s.orm.WithContext(ctx).Raw(`
		WITH RECURSIVE folder_tree AS (
			SELECT id FROM folders WHERE id = ?
			UNION ALL
			SELECT f.id FROM folders f INNER JOIN folder_tree ft ON f.parent_id = ft.id
		)
		SELECT id FROM folder_tree
	`, rootID).Scan(&ids).Error; err != nil {
		return nil, errors.Internal("failed to resolve folder tree", err)
	}
	return ids, nil
}

func (s *Service) hasTrashedAncestor(ctx context.Context, folderID int64) (bool, error) {
	var trashed int64
	if err := s.orm.WithContext(ctx).Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id, deleted_at FROM folders WHERE id = ?
			UNION ALL
			SELECT f.id, f.parent_id, f.deleted_at FROM folders f INNER JOIN ancestors a ON f.id = a.parent_id
		)
		SELECT COUNT(*) FROM ancestors WHERE deleted_at IS NOT NULL
	`, folderID).Scan(&trashed).Error; err != nil {
		return false, errors.Internal("failed to check folder ancestors", err)
	}
	return trashed > 0, nil
}

func (s *Service) restore(ctx context.Context, userID int64, itemType string, itemID string) error {
	id, err := strconv.ParseInt(itemID, 10, 64)
	if err != nil {
		return errors.Invalid("invalid id")
	}

	switch itemType {
	case "file":
		var record schemas.File
		if err := s.orm.WithContext(ctx).Where("id = ? AND uploaded_by = ? AND deleted_at IS NOT NULL", id, userID).First(&record).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return errors.NotFound("file not found in trash")
			}
			return errors.Internal("failed to find file", err)
		}
		updates := map[string]any{"deleted_at": nil}
		if record.FolderID != nil {
			trashedAncestor, err := s.hasTrashedAncestor(ctx, *record.FolderID)
			if err != nil {
				return err
			}
			if trashedAncestor {
				updates["folder_id"] = nil
			}
		}
		if err := s.orm.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
			return errors.Internal("failed to restore file", err)
		}
		if s.activity != nil {
			s.activity.Log(ctx, activity.Entry{
				UserID: userID, EventType: "file.restored", ResourceType: "file",
				ResourceID: record.ID, ResourceName: record.Name,
			})
		}
	case "folder":
		var record schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ? AND owner_id = ? AND deleted_at IS NOT NULL", id, userID).First(&record).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return errors.NotFound("folder not found in trash")
			}
			return errors.Internal("failed to find folder", err)
		}
		folderIDs, err := s.folderTreeIDs(ctx, record.ID)
		if err != nil {
			return err
		}
		detach := false
		if record.ParentID != nil {
			detach, err = s.hasTrashedAncestor(ctx, *record.ParentID)
			if err != nil {
				return err
			}
		}
		deletedAt := *record.DeletedAt
		err = s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&schemas.File{}).Where("folder_id IN ? AND deleted_at = ?", folderIDs, deletedAt).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			if err := tx.Model(&schemas.Folder{}).Where("id IN ? AND deleted_at = ?", folderIDs, deletedAt).Update("deleted_at", nil).Error; err != nil {
				return err
			}
			if detach {
				return tx.Model(&schemas.Folder{}).Where("id = ?", record.ID).Update("parent_id", nil).Error
			}
			return nil
		})
		if err != nil {
			return errors.Internal("failed to restore folder", err)
		}
		if s.activity != nil {
			s.activity.Log(ctx, activity.Entry{
				UserID: userID, EventType: "folder.restored", ResourceType: "folder",
				ResourceID: record.ID, ResourceName: record.Name,
			})
		}
	default:
		return errors.Invalid("type must be file or folder")
	}
	return nil
}

func purgeFilesTx(ctx context.Context, tx *gorm.DB, files []schemas.File) ([]string, int64, error) {
	if len(files) == 0 {
		return nil, 0, nil
	}

	if err := tombstone.RecordFiles(ctx, tx, files); err != nil {
		return nil, 0, err
	}

	fileIDs := make([]int64, len(files))
	keys := make([]string, 0, len(files))
	var bytes int64
	for i, f := range files {
		fileIDs[i] = f.ID
		keys = append(keys, f.BucketKey)
		bytes += f.Size
	}

	var versions []schemas.FileVersion
	if err := tx.Where("file_id IN ?", fileIDs).Find(&versions).Error; err != nil {
		return nil, 0, err
	}
	for _, v := range versions {
		keys = append(keys, v.BucketKey)
		bytes += v.Size
	}

	if err := tx.Where("file_id IN ?", fileIDs).Delete(&schemas.FileVersion{}).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Where("file_id IN ?", fileIDs).Delete(&schemas.Share{}).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Unscoped().Where("id IN ?", fileIDs).Delete(&schemas.File{}).Error; err != nil {
		return nil, 0, err
	}
	return keys, bytes, nil
}

func purgeFoldersTx(ctx context.Context, tx *gorm.DB, folderIDs []int64) error {
	if len(folderIDs) == 0 {
		return nil
	}

	var folders []schemas.Folder
	if err := tx.Where("id IN ?", folderIDs).Find(&folders).Error; err != nil {
		return err
	}
	if err := tombstone.RecordFolders(ctx, tx, folders); err != nil {
		return err
	}
	if err := tx.Model(&schemas.File{}).Where("folder_id IN ?", folderIDs).Update("folder_id", nil).Error; err != nil {
		return err
	}
	if err := tx.Model(&schemas.Folder{}).Where("parent_id IN ? AND id NOT IN ?", folderIDs, folderIDs).Update("parent_id", nil).Error; err != nil {
		return err
	}
	if err := tx.Where("folder_id IN ?", folderIDs).Delete(&schemas.Share{}).Error; err != nil {
		return err
	}
	if err := tx.Model(&schemas.UploadSession{}).Where("folder_id IN ?", folderIDs).Update("folder_id", nil).Error; err != nil {
		return err
	}
	return tx.Unscoped().Where("id IN ?", folderIDs).Delete(&schemas.Folder{}).Error
}

func (s *Service) refundQuota(ctx context.Context, userID int64, bytes int64) {
	if s.quota == nil || bytes <= 0 {
		return
	}
	if err := s.quota.UpdateUsage(ctx, userID, -bytes); err != nil {
		slog.Warn("trash: failed to refund quota", slog.Int64("user_id", userID), slog.Any("error", err))
	}
}

func (s *Service) deleteObjects(ctx context.Context, keys []string) {
	for _, key := range keys {
		if err := s.storage.DeleteObject(ctx, key); err != nil {
			slog.Warn("trash: failed to delete object from storage", slog.String("key", key), slog.Any("error", err))
		}
	}
}

func (s *Service) permanentDelete(ctx context.Context, userID int64, itemType string, itemID string) error {
	id, err := strconv.ParseInt(itemID, 10, 64)
	if err != nil {
		return errors.Invalid("invalid id")
	}

	switch itemType {
	case "file":
		var record schemas.File
		if err := s.orm.WithContext(ctx).Where("id = ? AND uploaded_by = ? AND deleted_at IS NOT NULL", id, userID).First(&record).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return errors.NotFound("file not found in trash")
			}
			return errors.Internal("failed to find file", err)
		}
		var keys []string
		var freed int64
		err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			k, b, err := purgeFilesTx(ctx, tx, []schemas.File{record})
			if err != nil {
				return err
			}
			keys, freed = k, b
			return nil
		})
		if err != nil {
			return errors.Internal("failed to delete file record", err)
		}
		s.refundQuota(ctx, userID, freed)
		s.deleteObjects(ctx, keys)
		if s.activity != nil {
			s.activity.Log(ctx, activity.Entry{
				UserID: userID, EventType: "file.permanently_deleted", ResourceType: "file",
				ResourceID: record.ID, ResourceName: record.Name,
			})
		}
	case "folder":
		var record schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ? AND owner_id = ? AND deleted_at IS NOT NULL", id, userID).First(&record).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return errors.NotFound("folder not found in trash")
			}
			return errors.Internal("failed to find folder", err)
		}
		folderIDs, err := s.folderTreeIDs(ctx, record.ID)
		if err != nil {
			return err
		}
		var keys []string
		var freed int64
		err = s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var files []schemas.File
			if err := tx.Where("folder_id IN ? AND uploaded_by = ? AND deleted_at IS NOT NULL", folderIDs, userID).Find(&files).Error; err != nil {
				return err
			}
			k, b, err := purgeFilesTx(ctx, tx, files)
			if err != nil {
				return err
			}
			keys, freed = k, b
			return purgeFoldersTx(ctx, tx, folderIDs)
		})
		if err != nil {
			return errors.Internal("failed to delete folder record", err)
		}
		s.refundQuota(ctx, userID, freed)
		s.deleteObjects(ctx, keys)
		if s.activity != nil {
			s.activity.Log(ctx, activity.Entry{
				UserID: userID, EventType: "folder.permanently_deleted", ResourceType: "folder",
				ResourceID: record.ID, ResourceName: record.Name,
			})
		}
	default:
		return errors.Invalid("type must be file or folder")
	}
	return nil
}

func (s *Service) emptyTrash(ctx context.Context, userID int64) (int64, error) {
	var trashFiles []schemas.File
	var trashFolders []schemas.Folder
	var keys []string
	var freed int64

	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("uploaded_by = ? AND deleted_at IS NOT NULL", userID).Find(&trashFiles).Error; err != nil {
			return err
		}
		k, b, err := purgeFilesTx(ctx, tx, trashFiles)
		if err != nil {
			return err
		}
		keys, freed = k, b

		if err := tx.Where("owner_id = ? AND deleted_at IS NOT NULL", userID).Find(&trashFolders).Error; err != nil {
			return err
		}
		folderIDs := make([]int64, len(trashFolders))
		for i, f := range trashFolders {
			folderIDs[i] = f.ID
		}
		return purgeFoldersTx(ctx, tx, folderIDs)
	})
	if err != nil {
		return 0, errors.Internal("failed to empty trash", err)
	}

	s.refundQuota(ctx, userID, freed)
	s.deleteObjects(ctx, keys)

	if s.activity != nil {
		for _, record := range trashFiles {
			s.activity.Log(ctx, activity.Entry{
				UserID: userID, EventType: "file.permanently_deleted", ResourceType: "file",
				ResourceID: record.ID, ResourceName: record.Name,
			})
		}
		for _, record := range trashFolders {
			s.activity.Log(ctx, activity.Entry{
				UserID: userID, EventType: "folder.permanently_deleted", ResourceType: "folder",
				ResourceID: record.ID, ResourceName: record.Name,
			})
		}
	}

	return int64(len(trashFiles) + len(trashFolders)), nil
}
