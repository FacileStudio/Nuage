package trash

import (
	"context"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	expireBatch        = 500
	expireLockID int64 = 7311402
)

// PurgeExpired deletes for good every trashed file and folder whose deletion is
// older than the retention window, refunds each uploader's quota and removes the
// stored objects. It returns how many items it purged. A retention of zero
// disables it.
func (s *Service) PurgeExpired(ctx context.Context, now time.Time) (int, error) {
	days, err := s.RetentionDays(ctx)
	if err != nil || days == 0 {
		return 0, err
	}
	cutoff := now.AddDate(0, 0, -days)

	total, err := s.drain(func() (int, error) { return s.expireFilesBatch(ctx, cutoff) })
	if err != nil {
		return total, err
	}
	folders, err := s.drain(func() (int, error) { return s.expireFoldersBatch(ctx, cutoff) })
	return total + folders, err
}

func (s *Service) drain(batch func() (int, error)) (int, error) {
	total := 0
	for {
		purged, err := batch()
		total += purged
		if err != nil {
			return total, err
		}
		if purged < expireBatch {
			return total, nil
		}
	}
}

func (s *Service) expireFilesBatch(ctx context.Context, cutoff time.Time) (int, error) {
	var batch []schemas.File
	var keys []storedObject
	freed := map[int64]int64{}

	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockExpiry(tx); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
			Order("id").Limit(expireBatch).Find(&batch).Error; err != nil {
			return err
		}
		var err error
		keys, err = purgeByUploader(ctx, tx, batch, freed)
		return err
	})
	if err != nil {
		return 0, errors.Internal("failed to expire trashed files", err)
	}

	for userID, bytes := range freed {
		s.refundQuota(ctx, userID, bytes)
	}
	s.deleteObjects(ctx, keys)
	s.logExpired(ctx, batch, nil)
	return len(batch), nil
}

func lockExpiry(tx *gorm.DB) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(?)", expireLockID).Error
}

func purgeByUploader(ctx context.Context, tx *gorm.DB, files []schemas.File, freed map[int64]int64) ([]storedObject, error) {
	groups := map[int64][]schemas.File{}
	for _, f := range files {
		groups[f.UploadedBy] = append(groups[f.UploadedBy], f)
	}
	var keys []storedObject
	for userID, group := range groups {
		k, bytes, err := purgeFilesTx(ctx, tx, group)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k...)
		freed[userID] += bytes
	}
	return keys, nil
}

func (s *Service) expireFoldersBatch(ctx context.Context, cutoff time.Time) (int, error) {
	var batch []schemas.Folder

	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockExpiry(tx); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("deleted_at IS NOT NULL AND deleted_at < ?", cutoff).
			Order("id").Limit(expireBatch).Find(&batch).Error; err != nil {
			return err
		}
		ids := make([]int64, len(batch))
		for i, f := range batch {
			ids[i] = f.ID
		}
		return purgeFoldersTx(ctx, tx, ids)
	})
	if err != nil {
		return 0, errors.Internal("failed to expire trashed folders", err)
	}

	s.logExpired(ctx, nil, batch)
	return len(batch), nil
}

func (s *Service) logExpired(ctx context.Context, files []schemas.File, folders []schemas.Folder) {
	if s.activity == nil {
		return
	}
	for _, f := range files {
		s.activity.Log(ctx, activity.Entry{
			UserID: f.UploadedBy, EventType: "file.expired", ResourceType: "file",
			ResourceID: f.ID, ResourceName: f.Name,
		})
	}
	for _, f := range folders {
		s.activity.Log(ctx, activity.Entry{
			UserID: f.OwnerID, EventType: "folder.expired", ResourceType: "folder",
			ResourceID: f.ID, ResourceName: f.Name,
		})
	}
}
