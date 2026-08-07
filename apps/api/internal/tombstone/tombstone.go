package tombstone

import (
	"context"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// RecordFiles persists deletion markers for permanently removed files. Without
// them a sync client whose cursor predates the removal would never learn the
// files are gone and would re-upload its local copies.
func RecordFiles(ctx context.Context, orm *gorm.DB, records []schemas.File) error {
	if len(records) == 0 {
		return nil
	}

	now := time.Now().UTC()
	entries := make([]schemas.Tombstone, 0, len(records))
	for _, record := range records {
		entries = append(entries, schemas.Tombstone{
			ResourceType: "file",
			ResourceID:   record.ID,
			FacileID:     record.FacileID,
			Name:         record.Name,
			UserID:       record.UploadedBy,
			SpaceID:      record.SpaceID,
			DeletedAt:    now,
		})
	}
	return insert(ctx, orm, entries)
}

// RecordFolders persists deletion markers for permanently removed folders.
func RecordFolders(ctx context.Context, orm *gorm.DB, records []schemas.Folder) error {
	if len(records) == 0 {
		return nil
	}

	now := time.Now().UTC()
	entries := make([]schemas.Tombstone, 0, len(records))
	for _, record := range records {
		entries = append(entries, schemas.Tombstone{
			ResourceType: "folder",
			ResourceID:   record.ID,
			FacileID:     record.FacileID,
			Name:         record.Name,
			UserID:       record.OwnerID,
			SpaceID:      record.SpaceID,
			DeletedAt:    now,
		})
	}
	return insert(ctx, orm, entries)
}

func insert(ctx context.Context, orm *gorm.DB, entries []schemas.Tombstone) error {
	if err := orm.WithContext(ctx).CreateInBatches(entries, 200).Error; err != nil {
		return errors.Internal("failed to record deletion markers", err)
	}
	return nil
}

// Prune drops markers past the retention window so the table stays bounded.
func Prune(ctx context.Context, orm *gorm.DB) (int64, error) {
	cutoff := time.Now().UTC().Add(-schemas.TombstoneRetention)
	result := orm.WithContext(ctx).
		Where("deleted_at < ?", cutoff).
		Delete(&schemas.Tombstone{})
	if result.Error != nil {
		return 0, errors.Internal("failed to prune deletion markers", result.Error)
	}
	return result.RowsAffected, nil
}
