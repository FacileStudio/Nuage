package files

import (
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm     *gorm.DB
	storage *storage.Client
}

func NewService(orm *gorm.DB, storageClient *storage.Client) *Service {
	return &Service{orm: orm, storage: storageClient}
}

func (s *Service) uploadFile(ctx context.Context, userID int64, name string, mimeType string, _ int64, reader io.Reader, folderID *int64, originApp string) (*schemas.File, error) {
	if folderID != nil {
		var folder schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ?", *folderID).First(&folder).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NotFound("folder not found")
			}
			return nil, errors.Internal("failed to verify folder", err)
		}
	}

	facileID := facile.NewID()
	bucketKey := fmt.Sprintf("%d/%s/%s", userID, facileID, name)

	if err := s.storage.PutObject(ctx, bucketKey, reader, -1, mimeType); err != nil {
		return nil, errors.Internal("failed to upload file", err)
	}

	info, err := s.storage.StatObject(ctx, bucketKey)
	if err != nil {
		return nil, errors.Internal("failed to stat uploaded file", err)
	}

	record := &schemas.File{
		FacileID:   facileID,
		Name:       name,
		MimeType:   mimeType,
		Size:       info.Size,
		BucketKey:  bucketKey,
		FolderID:   folderID,
		OriginApp:  originApp,
		UploadedBy: userID,
	}

	if err := s.orm.WithContext(ctx).Create(record).Error; err != nil {
		_ = s.storage.DeleteObject(ctx, bucketKey)
		return nil, errors.Internal("failed to save file record", err)
	}

	return record, nil
}

func (s *Service) listFiles(ctx context.Context, folderID *int64, search string, linkedTo string, originApp string) ([]schemas.File, error) {
	query := s.orm.WithContext(ctx).Where("deleted_at IS NULL").Order("created_at desc")

	if folderID != nil {
		query = query.Where("folder_id = ?", *folderID)
	} else if search == "" && linkedTo == "" && originApp == "" {
		query = query.Where("folder_id IS NULL")
	}
	if search != "" {
		query = query.Where("lower(name) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	if linkedTo != "" {
		query = query.Where("linked_to = ?", linkedTo)
	}
	if originApp != "" {
		query = query.Where("origin_app = ?", originApp)
	}

	var records []schemas.File
	if err := query.Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list files", err)
	}
	return records, nil
}

func (s *Service) getFile(ctx context.Context, fileID string) (*schemas.File, error) {
	id, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid file id")
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("file not found")
		}
		return nil, errors.Internal("failed to read file", err)
	}
	return &record, nil
}

func (s *Service) downloadFile(ctx context.Context, fileID string) (io.ReadCloser, *schemas.File, error) {
	record, err := s.getFile(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}

	reader, err := s.storage.GetObject(ctx, record.BucketKey)
	if err != nil {
		return nil, nil, errors.Internal("failed to read file from storage", err)
	}
	return reader, record, nil
}

func (s *Service) deleteFile(ctx context.Context, fileID string) error {
	record, err := s.getFile(ctx, fileID)
	if err != nil {
		return err
	}

	now := time.Now()
	if err := s.orm.WithContext(ctx).Model(record).Update("deleted_at", now).Error; err != nil {
		return errors.Internal("failed to soft delete file", err)
	}
	return nil
}

func (s *Service) updateFile(ctx context.Context, fileID string, name *string, folderID *int64) (*schemas.File, error) {
	id, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid file id")
	}

	updates := map[string]any{}
	if name != nil {
		updates["name"] = *name
	}
	if folderID != nil {
		if *folderID == 0 {
			updates["folder_id"] = nil
		} else {
			var folder schemas.Folder
			if err := s.orm.WithContext(ctx).Where("id = ?", *folderID).First(&folder).Error; err != nil {
				if stderrors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.NotFound("folder not found")
				}
				return nil, errors.Internal("failed to verify folder", err)
			}
			updates["folder_id"] = *folderID
		}
	}

	if len(updates) == 0 {
		return nil, errors.Invalid("at least one field must be provided")
	}

	if err := s.orm.WithContext(ctx).Model(&schemas.File{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, errors.Internal("failed to update file", err)
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return nil, errors.Internal("failed to read file", err)
	}
	return &record, nil
}

func (s *Service) linkFile(ctx context.Context, fileID string, linkedTo string) (*schemas.File, error) {
	id, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid file id")
	}

	if err := s.orm.WithContext(ctx).Model(&schemas.File{}).Where("id = ?", id).Update("linked_to", linkedTo).Error; err != nil {
		return nil, errors.Internal("failed to link file", err)
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return nil, errors.Internal("failed to read file", err)
	}
	return &record, nil
}

func (s *Service) createFolder(ctx context.Context, userID int64, name string, parentID *int64) (*schemas.Folder, error) {
	if parentID != nil {
		var parent schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ?", *parentID).First(&parent).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.NotFound("parent folder not found")
			}
			return nil, errors.Internal("failed to verify parent folder", err)
		}
	}

	record := &schemas.Folder{
		FacileID: facile.NewID(),
		Name:     name,
		ParentID: parentID,
		OwnerID:  userID,
	}
	if err := s.orm.WithContext(ctx).Create(record).Error; err != nil {
		return nil, errors.Internal("failed to create folder", err)
	}
	return record, nil
}

func (s *Service) listFolders(ctx context.Context, parentID *int64) ([]schemas.Folder, error) {
	query := s.orm.WithContext(ctx).Where("deleted_at IS NULL").Order("name asc")

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	var records []schemas.Folder
	if err := query.Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list folders", err)
	}
	return records, nil
}

func (s *Service) getFolder(ctx context.Context, folderID string) (*schemas.Folder, []schemas.File, []schemas.Folder, error) {
	id, err := strconv.ParseInt(folderID, 10, 64)
	if err != nil {
		return nil, nil, nil, errors.Invalid("invalid folder id")
	}

	var folder schemas.Folder
	if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&folder).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, errors.NotFound("folder not found")
		}
		return nil, nil, nil, errors.Internal("failed to read folder", err)
	}

	var childFiles []schemas.File
	if err := s.orm.WithContext(ctx).Where("folder_id = ? AND deleted_at IS NULL", id).Order("created_at desc").Find(&childFiles).Error; err != nil {
		return nil, nil, nil, errors.Internal("failed to list folder files", err)
	}

	var childFolders []schemas.Folder
	if err := s.orm.WithContext(ctx).Where("parent_id = ? AND deleted_at IS NULL", id).Order("name asc").Find(&childFolders).Error; err != nil {
		return nil, nil, nil, errors.Internal("failed to list subfolders", err)
	}

	return &folder, childFiles, childFolders, nil
}

func (s *Service) updateFolder(ctx context.Context, folderID string, name *string, parentID *int64) (*schemas.Folder, error) {
	id, err := strconv.ParseInt(folderID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid folder id")
	}

	updates := map[string]any{}
	if name != nil {
		updates["name"] = *name
	}
	if parentID != nil {
		if *parentID == 0 {
			updates["parent_id"] = nil
		} else {
			if *parentID == id {
				return nil, errors.Invalid("folder cannot be its own parent")
			}
			var parent schemas.Folder
			if err := s.orm.WithContext(ctx).Where("id = ?", *parentID).First(&parent).Error; err != nil {
				if stderrors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.NotFound("parent folder not found")
				}
				return nil, errors.Internal("failed to verify parent folder", err)
			}
			updates["parent_id"] = *parentID
		}
	}

	if len(updates) == 0 {
		return nil, errors.Invalid("at least one field must be provided")
	}

	if err := s.orm.WithContext(ctx).Model(&schemas.Folder{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, errors.Internal("failed to update folder", err)
	}

	var record schemas.Folder
	if err := s.orm.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return nil, errors.Internal("failed to read folder", err)
	}
	return &record, nil
}

func (s *Service) deleteFolder(ctx context.Context, folderID string) error {
	id, err := strconv.ParseInt(folderID, 10, 64)
	if err != nil {
		return errors.Invalid("invalid folder id")
	}

	var folder schemas.Folder
	if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&folder).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("folder not found")
		}
		return errors.Internal("failed to read folder", err)
	}

	var fileCount int64
	if err := s.orm.WithContext(ctx).Model(&schemas.File{}).Where("folder_id = ? AND deleted_at IS NULL", id).Count(&fileCount).Error; err != nil {
		return errors.Internal("failed to check folder contents", err)
	}
	if fileCount > 0 {
		return errors.Failed("folder is not empty")
	}

	var subfolderCount int64
	if err := s.orm.WithContext(ctx).Model(&schemas.Folder{}).Where("parent_id = ? AND deleted_at IS NULL", id).Count(&subfolderCount).Error; err != nil {
		return errors.Internal("failed to check subfolders", err)
	}
	if subfolderCount > 0 {
		return errors.Failed("folder contains subfolders")
	}

	now := time.Now()
	if err := s.orm.WithContext(ctx).Model(&folder).Update("deleted_at", now).Error; err != nil {
		return errors.Internal("failed to soft delete folder", err)
	}
	return nil
}

func mapFile(record schemas.File) FileResponse {
	return FileResponse{
		ID:         record.ID,
		FacileID:   record.FacileID,
		Name:       record.Name,
		MimeType:   record.MimeType,
		Size:       record.Size,
		FolderID:   record.FolderID,
		OriginApp:  record.OriginApp,
		LinkedTo:   record.LinkedTo,
		UploadedBy: record.UploadedBy,
		CreatedAt:  record.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  record.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func mapFolder(record schemas.Folder) FolderResponse {
	return FolderResponse{
		ID:        record.ID,
		FacileID:  record.FacileID,
		Name:      record.Name,
		ParentID:  record.ParentID,
		OwnerID:   record.OwnerID,
		CreatedAt: record.CreatedAt.UTC().Format(time.RFC3339),
	}
}
