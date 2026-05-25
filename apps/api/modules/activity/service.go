package activity

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

type ListParams struct {
	UserID       *int64
	EventType    string
	ResourceType string
	ResourceID   *int64
	Page         int
	PerPage      int
}

func (s *Service) List(ctx context.Context, params ListParams) ([]schemas.ActivityLog, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 100 {
		params.PerPage = 50
	}

	query := s.orm.WithContext(ctx).Model(&schemas.ActivityLog{})

	if params.UserID != nil {
		query = query.Where("user_id = ?", *params.UserID)
	}
	if params.EventType != "" {
		query = query.Where("event_type = ?", params.EventType)
	}
	if params.ResourceType != "" {
		query = query.Where("resource_type = ?", params.ResourceType)
	}
	if params.ResourceID != nil {
		query = query.Where("resource_id = ?", *params.ResourceID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.Internal("failed to count activity", err)
	}

	var records []schemas.ActivityLog
	offset := (params.Page - 1) * params.PerPage
	if err := query.Order("created_at desc").Offset(offset).Limit(params.PerPage).Find(&records).Error; err != nil {
		return nil, 0, errors.Internal("failed to list activity", err)
	}

	return records, total, nil
}

func (s *Service) ForFile(ctx context.Context, fileID int64, page, perPage int) ([]schemas.ActivityLog, int64, error) {
	rid := fileID
	return s.List(ctx, ListParams{
		ResourceType: "file",
		ResourceID:   &rid,
		Page:         page,
		PerPage:      perPage,
	})
}

func mapActivity(record schemas.ActivityLog) ActivityResponse {
	return ActivityResponse{
		ID:           record.ID,
		UserID:       record.UserID,
		EventType:    record.EventType,
		ResourceType: record.ResourceType,
		ResourceID:   record.ResourceID,
		ResourceName: record.ResourceName,
		Metadata:     record.Metadata,
		CreatedAt:    record.CreatedAt.UTC().Format(time.RFC3339),
	}
}
