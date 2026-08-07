package quota

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

const defaultQuotaBytes int64 = 50 * 1024 * 1024 * 1024

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
}

func (s *Service) getDefaultLimit() int64 {
	var setting schemas.Setting
	if err := s.orm.Where("key = ?", "default_storage_quota").First(&setting).Error; err == nil {
		if n, err := strconv.ParseInt(setting.Value, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return defaultQuotaBytes
}

func (s *Service) ensureQuota(ctx context.Context, userID int64) (*schemas.UserQuota, error) {
	var quota schemas.UserQuota
	if err := s.orm.WithContext(ctx).
		Where(schemas.UserQuota{UserID: userID}).
		Attrs(schemas.UserQuota{StorageUsed: 0, StorageLimit: 0}).
		FirstOrCreate(&quota).Error; err != nil {
		return nil, errors.Internal("failed to load quota", err)
	}
	return &quota, nil
}

func (s *Service) GetUsage(ctx context.Context, userID int64) (*UsageResponse, error) {
	quota, err := s.ensureQuota(ctx, userID)
	if err != nil {
		return nil, err
	}
	limit := quota.StorageLimit
	if limit == 0 {
		limit = s.getDefaultLimit()
	}

	var pct float64
	if limit > 0 {
		pct = float64(quota.StorageUsed) / float64(limit) * 100
		if pct > 100 {
			pct = 100
		}
	}

	return &UsageResponse{
		UserID:       userID,
		StorageUsed:  quota.StorageUsed,
		StorageLimit: limit,
		Percentage:   pct,
	}, nil
}

func (s *Service) CheckQuota(ctx context.Context, userID int64, additionalBytes int64) error {
	quota, err := s.ensureQuota(ctx, userID)
	if err != nil {
		return err
	}
	limit := quota.StorageLimit
	if limit == 0 {
		limit = s.getDefaultLimit()
	}
	if limit < 0 {
		return nil
	}

	if quota.StorageUsed+additionalBytes > limit {
		return errors.TooLarge("storage quota exceeded")
	}
	return nil
}

func (s *Service) UpdateUsage(ctx context.Context, userID int64, delta int64) error {
	if _, err := s.ensureQuota(ctx, userID); err != nil {
		return err
	}
	if err := s.orm.WithContext(ctx).Model(&schemas.UserQuota{}).
		Where("user_id = ?", userID).
		Update("storage_used", gorm.Expr("GREATEST(storage_used + ?, 0)", delta)).Error; err != nil {
		return errors.Internal("failed to update usage", err)
	}
	return nil
}

func (s *Service) RecalculateUsage(ctx context.Context, userID int64) error {
	var totalSize int64
	if err := s.orm.WithContext(ctx).Raw(`
		SELECT COALESCE((SELECT SUM(size) FROM files WHERE uploaded_by = ?), 0)
			+ COALESCE((SELECT SUM(v.size) FROM file_versions v INNER JOIN files f ON f.id = v.file_id WHERE f.uploaded_by = ?), 0)
	`, userID, userID).Scan(&totalSize).Error; err != nil {
		return errors.Internal("failed to calculate usage", err)
	}

	if _, err := s.ensureQuota(ctx, userID); err != nil {
		return err
	}
	if err := s.orm.WithContext(ctx).Model(&schemas.UserQuota{}).
		Where("user_id = ?", userID).
		Update("storage_used", totalSize).Error; err != nil {
		return errors.Internal("failed to update usage", err)
	}
	return nil
}

func (s *Service) SetLimit(ctx context.Context, userID int64, limit int64) error {
	var user schemas.User
	if err := s.orm.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("user not found")
		}
		return errors.Internal("failed to find user", err)
	}

	if _, err := s.ensureQuota(ctx, userID); err != nil {
		return err
	}
	if err := s.orm.WithContext(ctx).Model(&schemas.UserQuota{}).Where("user_id = ?", userID).Update("storage_limit", limit).Error; err != nil {
		return errors.Internal("failed to set quota", err)
	}
	return nil
}

func (s *Service) ListAllUsage(ctx context.Context) ([]UsageResponse, error) {
	var users []schemas.User
	if err := s.orm.WithContext(ctx).Order("id asc").Find(&users).Error; err != nil {
		return nil, errors.Internal("failed to list users", err)
	}

	userIDs := make([]int64, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}

	var quotas []schemas.UserQuota
	if err := s.orm.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&quotas).Error; err != nil {
		return nil, errors.Internal("failed to list quotas", err)
	}
	quotaMap := make(map[int64]*schemas.UserQuota, len(quotas))
	for i := range quotas {
		quotaMap[quotas[i].UserID] = &quotas[i]
	}

	defaultLimit := s.getDefaultLimit()
	results := make([]UsageResponse, 0, len(users))

	for _, u := range users {
		q, ok := quotaMap[u.ID]
		if !ok {
			q = &schemas.UserQuota{UserID: u.ID, StorageUsed: 0, StorageLimit: 0}
		}
		limit := q.StorageLimit
		if limit == 0 {
			limit = defaultLimit
		}

		var pct float64
		if limit > 0 {
			pct = float64(q.StorageUsed) / float64(limit) * 100
			if pct > 100 {
				pct = 100
			}
		}

		results = append(results, UsageResponse{
			UserID:       u.ID,
			StorageUsed:  q.StorageUsed,
			StorageLimit: limit,
			Percentage:   pct,
		})
	}

	return results, nil
}
