package sharing

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

type Service struct {
	orm      *gorm.DB
	notifier *nook.Notifier
}

func NewService(orm *gorm.DB, notifier *nook.Notifier) *Service {
	return &Service{orm: orm, notifier: notifier}
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
