package trash

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
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

func (s *Service) listTrash(ctx context.Context, userID int64) ([]TrashItem, error) {
	var files []schemas.File
	if err := s.orm.WithContext(ctx).Where("uploaded_by = ? AND deleted_at IS NOT NULL", userID).Order("deleted_at desc").Find(&files).Error; err != nil {
		return nil, errors.Internal("failed to list trashed files", err)
	}

	var folders []schemas.Folder
	if err := s.orm.WithContext(ctx).Where("owner_id = ? AND deleted_at IS NOT NULL", userID).Order("deleted_at desc").Find(&folders).Error; err != nil {
		return nil, errors.Internal("failed to list trashed folders", err)
	}

	items := make([]TrashItem, 0, len(files)+len(folders))
	for _, f := range files {
		items = append(items, TrashItem{
			Type:      "file",
			ID:        f.ID,
			FacileID:  f.FacileID,
			Name:      f.Name,
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
		if err := s.orm.WithContext(ctx).Model(&record).Update("deleted_at", nil).Error; err != nil {
			return errors.Internal("failed to restore file", err)
		}
		if s.quota != nil {
			s.quota.UpdateUsage(ctx, userID, record.Size)
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
		if err := s.orm.WithContext(ctx).Model(&record).Update("deleted_at", nil).Error; err != nil {
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
		if err := s.storage.DeleteObject(ctx, record.BucketKey); err != nil {
			return errors.Internal("failed to delete file from storage", err)
		}
		var versions []schemas.FileVersion
		s.orm.WithContext(ctx).Where("file_id = ?", record.ID).Find(&versions)
		var versionBytes int64
		for _, v := range versions {
			_ = s.storage.DeleteObject(ctx, v.BucketKey)
			versionBytes += v.Size
		}
		s.orm.WithContext(ctx).Where("file_id = ?", record.ID).Delete(&schemas.FileVersion{})
		if err := s.orm.WithContext(ctx).Unscoped().Delete(&record).Error; err != nil {
			return errors.Internal("failed to delete file record", err)
		}
		if s.quota != nil {
			s.quota.UpdateUsage(ctx, userID, -(record.Size + versionBytes))
		}
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
		if err := s.orm.WithContext(ctx).Unscoped().Delete(&record).Error; err != nil {
			return errors.Internal("failed to delete folder record", err)
		}
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
