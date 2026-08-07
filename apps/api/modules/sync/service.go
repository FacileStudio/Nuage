package sync

import (
	"context"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// cursorLag is subtracted from the cursor handed back to clients. A row written
// by a transaction that commits just after the cursor is read would otherwise
// carry an updated_at below the cursor and never be observed again.
const cursorLag = 2 * time.Second

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

// reachable restricts a query to the caller's personal items plus every item
// belonging to a space they are a member of.
func reachable(orm *gorm.DB, ownerColumn string, userID int64, spaceIDs []int64) *gorm.DB {
	personal := orm.Where(ownerColumn+" = ? AND space_id IS NULL", userID)
	if len(spaceIDs) == 0 {
		return personal
	}
	return personal.Or("space_id IN ?", spaceIDs)
}

func (s *Service) changes(ctx context.Context, userID int64, since time.Time) (*ChangesResponse, error) {
	cursor := time.Now().UTC()

	spaceIDs, err := spaceaccess.MemberIDs(ctx, s.orm, userID)
	if err != nil {
		return nil, err
	}

	var changedFiles []schemas.File
	if err := s.orm.WithContext(ctx).
		Where("deleted_at IS NULL AND updated_at > ?", since).
		Where(reachable(s.orm, "uploaded_by", userID, spaceIDs)).
		Find(&changedFiles).Error; err != nil {
		return nil, errors.Internal("failed to query changed files", err)
	}

	var changedFolders []schemas.Folder
	if err := s.orm.WithContext(ctx).
		Where("deleted_at IS NULL AND updated_at > ?", since).
		Where(reachable(s.orm, "owner_id", userID, spaceIDs)).
		Find(&changedFolders).Error; err != nil {
		return nil, errors.Internal("failed to query changed folders", err)
	}

	var trashedFiles []schemas.File
	if err := s.orm.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_at > ?", since).
		Where(reachable(s.orm, "uploaded_by", userID, spaceIDs)).
		Find(&trashedFiles).Error; err != nil {
		return nil, errors.Internal("failed to query deleted files", err)
	}

	var trashedFolders []schemas.Folder
	if err := s.orm.WithContext(ctx).
		Where("deleted_at IS NOT NULL AND deleted_at > ?", since).
		Where(reachable(s.orm, "owner_id", userID, spaceIDs)).
		Find(&trashedFolders).Error; err != nil {
		return nil, errors.Internal("failed to query deleted folders", err)
	}

	var markers []schemas.Tombstone
	if err := s.orm.WithContext(ctx).
		Where("deleted_at > ?", since).
		Where(reachable(s.orm, "user_id", userID, spaceIDs)).
		Find(&markers).Error; err != nil {
		return nil, errors.Internal("failed to query deletion markers", err)
	}

	deletedFiles := mapTrashedFiles(trashedFiles)
	deletedFolders := mapTrashedFolders(trashedFolders)
	for _, marker := range markers {
		item := DeletedItem{
			ID:        marker.ResourceID,
			FacileID:  marker.FacileID,
			Name:      marker.Name,
			SpaceID:   marker.SpaceID,
			DeletedAt: marker.DeletedAt.UTC().Format(time.RFC3339Nano),
			Permanent: true,
		}
		if marker.ResourceType == "folder" {
			deletedFolders = append(deletedFolders, item)
		} else {
			deletedFiles = append(deletedFiles, item)
		}
	}

	resp := &ChangesResponse{
		Files: ChangedItems{
			Changed: mapFiles(changedFiles),
			Deleted: deletedFiles,
		},
		Folders: ChangedItems{
			Changed: mapFolders(changedFolders),
			Deleted: deletedFolders,
		},
		ServerTime: cursor.Add(-cursorLag).Format(time.RFC3339Nano),
	}
	return resp, nil
}

func (s *Service) state(ctx context.Context, userID int64) (*StateResponse, error) {
	cursor := time.Now().UTC()

	spaceIDs, err := spaceaccess.MemberIDs(ctx, s.orm, userID)
	if err != nil {
		return nil, err
	}

	var files []schemas.File
	if err := s.orm.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where(reachable(s.orm, "uploaded_by", userID, spaceIDs)).
		Find(&files).Error; err != nil {
		return nil, errors.Internal("failed to query files", err)
	}

	var folders []schemas.Folder
	if err := s.orm.WithContext(ctx).
		Where("deleted_at IS NULL").
		Where(reachable(s.orm, "owner_id", userID, spaceIDs)).
		Find(&folders).Error; err != nil {
		return nil, errors.Internal("failed to query folders", err)
	}

	resp := &StateResponse{
		Files:      mapFiles(files),
		Folders:    mapFolders(folders),
		ServerTime: cursor.Add(-cursorLag).Format(time.RFC3339Nano),
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
			SpaceID:   r.SpaceID,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339Nano),
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
			SpaceID:   r.SpaceID,
			CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return items
}

func mapTrashedFiles(records []schemas.File) []DeletedItem {
	items := make([]DeletedItem, 0, len(records))
	for _, r := range records {
		items = append(items, DeletedItem{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			SpaceID:   r.SpaceID,
			DeletedAt: r.DeletedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return items
}

func mapTrashedFolders(records []schemas.Folder) []DeletedItem {
	items := make([]DeletedItem, 0, len(records))
	for _, r := range records {
		items = append(items, DeletedItem{
			ID:        r.ID,
			FacileID:  r.FacileID,
			Name:      r.Name,
			SpaceID:   r.SpaceID,
			DeletedAt: r.DeletedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return items
}
