package sync

import (
	"context"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) changes(ctx context.Context, userID int64, since time.Time) (*ChangesResponse, error) {
	var changedFiles []schemas.File
	if err := s.orm.WithContext(ctx).
		Where("uploaded_by = ? AND deleted_at IS NULL AND updated_at > ?", userID, since).
		Find(&changedFiles).Error; err != nil {
		return nil, errors.Internal("failed to query changed files", err)
	}

	var changedFolders []schemas.Folder
	if err := s.orm.WithContext(ctx).
		Where("owner_id = ? AND deleted_at IS NULL AND (created_at > ? OR updated_at > ?)", userID, since, since).
		Find(&changedFolders).Error; err != nil {
		return nil, errors.Internal("failed to query changed folders", err)
	}

	var deletedFiles []schemas.File
	if err := s.orm.WithContext(ctx).
		Where("uploaded_by = ? AND deleted_at IS NOT NULL AND deleted_at > ?", userID, since).
		Select("id", "facile_id", "name", "deleted_at").
		Find(&deletedFiles).Error; err != nil {
		return nil, errors.Internal("failed to query deleted files", err)
	}

	var deletedFolders []schemas.Folder
	if err := s.orm.WithContext(ctx).
		Where("owner_id = ? AND deleted_at IS NOT NULL AND deleted_at > ?", userID, since).
		Select("id", "facile_id", "name", "deleted_at").
		Find(&deletedFolders).Error; err != nil {
		return nil, errors.Internal("failed to query deleted folders", err)
	}

	resp := &ChangesResponse{
		Files: ChangedItems{
			Changed: mapFiles(changedFiles),
			Deleted: mapDeletedFiles(deletedFiles),
		},
		Folders: ChangedItems{
			Changed: mapFolders(changedFolders),
			Deleted: mapDeletedFolders(deletedFolders),
		},
		ServerTime: time.Now().UTC().Format(time.RFC3339),
	}
	return resp, nil
}

func (s *Service) state(ctx context.Context, userID int64) (*StateResponse, error) {
	var files []schemas.File
	if err := s.orm.WithContext(ctx).
		Where("uploaded_by = ? AND deleted_at IS NULL", userID).
		Find(&files).Error; err != nil {
		return nil, errors.Internal("failed to query files", err)
	}

	var folders []schemas.Folder
	if err := s.orm.WithContext(ctx).
		Where("owner_id = ? AND deleted_at IS NULL", userID).
		Find(&folders).Error; err != nil {
		return nil, errors.Internal("failed to query folders", err)
	}

	resp := &StateResponse{
		Files:      mapFiles(files),
		Folders:    mapFolders(folders),
		ServerTime: time.Now().UTC().Format(time.RFC3339),
	}
	return resp, nil
}

func mapFiles(records []schemas.File) []ItemResponse {
	items := make([]ItemResponse, 0, len(records))
	for _, r := range records {
		items = append(items, ItemResponse{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			MimeType:  r.MimeType,
			Size:      r.Size,
			Hash:      r.Hash,
			FolderID:  r.FolderID,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items
}

func mapFolders(records []schemas.Folder) []ItemResponse {
	items := make([]ItemResponse, 0, len(records))
	for _, r := range records {
		items = append(items, ItemResponse{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			ParentID:  r.ParentID,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items
}

func mapDeletedFiles(records []schemas.File) []DeletedItem {
	items := make([]DeletedItem, 0, len(records))
	for _, r := range records {
		items = append(items, DeletedItem{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			DeletedAt: r.DeletedAt.UTC().Format(time.RFC3339),
		})
	}
	return items
}

func mapDeletedFolders(records []schemas.Folder) []DeletedItem {
	items := make([]DeletedItem, 0, len(records))
	for _, r := range records {
		items = append(items, DeletedItem{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			DeletedAt: r.DeletedAt.UTC().Format(time.RFC3339),
		})
	}
	return items
}
