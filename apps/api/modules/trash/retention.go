package trash

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// RetentionDays reports how many days a trashed item is kept. It falls back to
// the default while the setting has never been saved or holds garbage, so a bad
// value cannot turn the purge into "delete everything".
func (s *Service) RetentionDays(ctx context.Context) (int, error) {
	var record schemas.Setting
	err := s.orm.WithContext(ctx).Where("key = ?", schemas.TrashRetentionKey).First(&record).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return schemas.DefaultTrashRetentionDays, nil
	}
	if err != nil {
		return 0, errors.Internal("failed to read trash retention", err)
	}
	days, convErr := strconv.Atoi(record.Value)
	if convErr != nil || days < 0 || days > schemas.MaxTrashRetentionDays {
		return schemas.DefaultTrashRetentionDays, nil
	}
	return days, nil
}

func (s *Service) withExpiry(ctx context.Context, items []TrashItem) ([]TrashItem, int, error) {
	days, err := s.RetentionDays(ctx)
	if err != nil || days == 0 {
		return items, 0, err
	}
	for i, item := range items {
		deleted, parseErr := time.Parse(time.RFC3339, item.DeletedAt)
		if parseErr != nil {
			continue
		}
		items[i].ExpiresAt = deleted.AddDate(0, 0, days).UTC().Format(time.RFC3339)
	}
	return items, days, nil
}
