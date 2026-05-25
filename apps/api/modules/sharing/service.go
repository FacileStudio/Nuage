package sharing

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm      *gorm.DB
	notifier *nook.Notifier
	activity *activity.Logger
}

func NewService(orm *gorm.DB, notifier *nook.Notifier, actLogger *activity.Logger) *Service {
	return &Service{orm: orm, notifier: notifier, activity: actLogger}
}

func (s *Service) createShare(ctx context.Context, userID int64, req CreateShareRequest) (*schemas.Share, error) {
	if req.FileID == nil && req.FolderID == nil {
		return nil, errors.Invalid("file_id or folder_id is required")
	}
	if req.FileID != nil && req.FolderID != nil {
		return nil, errors.Invalid("only one of file_id or folder_id allowed")
	}

	permission := req.Permission
	if permission == "" {
		permission = "view"
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, errors.Invalid("invalid expires_at format")
		}
		expiresAt = &t
	}

	record := &schemas.Share{
		Token:      facile.NewID(),
		FileID:     req.FileID,
		FolderID:   req.FolderID,
		SharedBy:   userID,
		SharedWith: req.SharedWith,
		Permission: permission,
		ExpiresAt:  expiresAt,
	}

	if err := s.orm.WithContext(ctx).Create(record).Error; err != nil {
		return nil, errors.Internal("failed to create share", err)
	}

	var sharedWithEmail string
	if record.SharedWith != nil {
		var target schemas.User
		if err := s.orm.WithContext(ctx).Where("id = ?", *record.SharedWith).First(&target).Error; err == nil {
			sharedWithEmail = target.Email
		}
	}
	s.notifier.Notify(ctx, userID, "share.created", nook.EventData{
		Share: &nook.ShareData{ID: record.ID, SharedWithEmail: sharedWithEmail, Permission: record.Permission},
	})

	if s.activity != nil {
		s.activity.Log(ctx, activity.Entry{
			UserID: userID, EventType: "share.created", ResourceType: "share",
			ResourceID: record.ID, ResourceName: sharedWithEmail,
		})
	}

	return record, nil
}

func (s *Service) listSharedWithMe(ctx context.Context, userID int64) ([]schemas.Share, error) {
	var records []schemas.Share
	if err := s.orm.WithContext(ctx).
		Preload("File").Preload("Folder").
		Where("shared_with = ?", userID).
		Order("created_at desc").
		Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list shares", err)
	}
	return records, nil
}

func (s *Service) listSharedByMe(ctx context.Context, userID int64) ([]schemas.Share, error) {
	var records []schemas.Share
	if err := s.orm.WithContext(ctx).
		Preload("File").Preload("Folder").
		Where("shared_by = ?", userID).
		Order("created_at desc").
		Find(&records).Error; err != nil {
		return nil, errors.Internal("failed to list shares", err)
	}
	return records, nil
}

func (s *Service) deleteShare(ctx context.Context, userID int64, shareID string) error {
	id, err := strconv.ParseInt(shareID, 10, 64)
	if err != nil {
		return errors.Invalid("invalid share id")
	}

	var share schemas.Share
	if err := s.orm.WithContext(ctx).Where("id = ? AND shared_by = ?", id, userID).First(&share).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("share not found")
		}
		return errors.Internal("failed to find share", err)
	}

	if err := s.orm.WithContext(ctx).Delete(&share).Error; err != nil {
		return errors.Internal("failed to delete share", err)
	}

	s.notifier.Notify(ctx, userID, "share.revoked", nook.EventData{
		Share: &nook.ShareData{ID: share.ID, Permission: share.Permission},
	})

	if s.activity != nil {
		s.activity.Log(ctx, activity.Entry{
			UserID: userID, EventType: "share.revoked", ResourceType: "share",
			ResourceID: share.ID,
		})
	}

	return nil
}

func (s *Service) getByToken(ctx context.Context, token string) (*schemas.Share, error) {
	var record schemas.Share
	if err := s.orm.WithContext(ctx).
		Preload("File").Preload("Folder").
		Where("token = ?", token).
		First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("share not found")
		}
		return nil, errors.Internal("failed to find share", err)
	}

	if record.ExpiresAt != nil && record.ExpiresAt.Before(time.Now()) {
		return nil, errors.NotFound("share has expired")
	}

	return &record, nil
}

func (s *Service) checkPermission(ctx context.Context, token string, requiredPermission string) (*schemas.Share, error) {
	share, err := s.getByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if requiredPermission == "edit" && share.Permission != "edit" {
		return nil, errors.Forbidden("insufficient share permission")
	}

	return share, nil
}

func (s *Service) getSharedFile(ctx context.Context, token string, fileID int64) (*schemas.File, *schemas.Share, error) {
	share, err := s.getByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	var file schemas.File
	if share.FileID != nil && *share.FileID == fileID {
		if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", fileID).First(&file).Error; err != nil {
			return nil, nil, errors.NotFound("file not found")
		}
		return &file, share, nil
	}

	if share.FolderID != nil {
		if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", fileID).First(&file).Error; err != nil {
			return nil, nil, errors.NotFound("file not found")
		}
		if s.isFileInSharedFolder(ctx, file, *share.FolderID) {
			return &file, share, nil
		}
	}

	return nil, nil, errors.Forbidden("file not accessible via this share")
}

func (s *Service) isFileInSharedFolder(ctx context.Context, file schemas.File, sharedFolderID int64) bool {
	if file.FolderID == nil {
		return false
	}

	folderID := *file.FolderID
	for i := 0; i < 50; i++ {
		if folderID == sharedFolderID {
			return true
		}
		var folder schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", folderID).First(&folder).Error; err != nil {
			return false
		}
		if folder.ParentID == nil {
			return false
		}
		folderID = *folder.ParentID
	}
	return false
}

func (s *Service) listSharedFolderContents(ctx context.Context, token string, folderID int64) ([]schemas.File, []schemas.Folder, *schemas.Share, error) {
	share, err := s.getByToken(ctx, token)
	if err != nil {
		return nil, nil, nil, err
	}

	if share.FolderID == nil {
		return nil, nil, nil, errors.Invalid("this share is not a folder share")
	}

	targetID := folderID
	if targetID == 0 {
		targetID = *share.FolderID
	} else {
		if !s.isFolderInSharedFolder(ctx, targetID, *share.FolderID) {
			return nil, nil, nil, errors.Forbidden("folder not accessible via this share")
		}
	}

	var files []schemas.File
	s.orm.WithContext(ctx).Where("folder_id = ? AND deleted_at IS NULL", targetID).Order("created_at desc").Find(&files)

	var folders []schemas.Folder
	s.orm.WithContext(ctx).Where("parent_id = ? AND deleted_at IS NULL", targetID).Order("name asc").Find(&folders)

	return files, folders, share, nil
}

func (s *Service) isFolderInSharedFolder(ctx context.Context, folderID, sharedFolderID int64) bool {
	if folderID == sharedFolderID {
		return true
	}

	currentID := folderID
	for i := 0; i < 50; i++ {
		var folder schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", currentID).First(&folder).Error; err != nil {
			return false
		}
		if folder.ParentID == nil {
			return false
		}
		if *folder.ParentID == sharedFolderID {
			return true
		}
		currentID = *folder.ParentID
	}
	return false
}

func mapShare(record schemas.Share) ShareResponse {
	resp := ShareResponse{
		ID:         record.ID,
		Token:      record.Token,
		FileID:     record.FileID,
		FolderID:   record.FolderID,
		SharedBy:   record.SharedBy,
		SharedWith: record.SharedWith,
		Permission: record.Permission,
		CreatedAt:  record.CreatedAt.UTC().Format(time.RFC3339),
	}
	if record.ExpiresAt != nil {
		formatted := record.ExpiresAt.UTC().Format(time.RFC3339)
		resp.ExpiresAt = &formatted
	}
	return resp
}
