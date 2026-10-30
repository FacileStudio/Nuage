package spaces

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

// MigrateBuckets moves the objects of every space that still lives in the shared
// bucket into the space's own bucket, and gives spaces without files their
// bucket. It is safe to run on every boot: a space with nothing left to move
// costs one query.
//
// For each space it copies every object first, then repoints the rows in one
// transaction, then deletes the originals. A crash before the repoint redoes
// the copy over identical bytes; a crash after it leaves originals behind in the
// shared bucket, which costs space and breaks nothing.
func (s *Service) MigrateBuckets(ctx context.Context) error {
	var all []schemas.Space
	if err := s.orm.WithContext(ctx).Select("id", "facile_id").Find(&all).Error; err != nil {
		return fmt.Errorf("list spaces: %w", err)
	}
	for _, space := range all {
		if err := s.migrateSpaceBucket(ctx, space); err != nil {
			return fmt.Errorf("space %d: %w", space.ID, err)
		}
	}
	return nil
}

func (s *Service) migrateSpaceBucket(ctx context.Context, space schemas.Space) error {
	name := storage.SpaceBucket(space.FacileID)
	dest := s.storage.For(name)
	if err := dest.EnsureBucket(ctx); err != nil {
		return err
	}

	var fileKeys, versionKeys []string
	if err := s.orm.WithContext(ctx).Model(&schemas.File{}).
		Where("space_id = ? AND bucket = ''", space.ID).
		Pluck("bucket_key", &fileKeys).Error; err != nil {
		return err
	}
	if err := s.orm.WithContext(ctx).Model(&schemas.FileVersion{}).
		Where("bucket = '' AND file_id IN (?)", s.orm.Model(&schemas.File{}).Select("id").Where("space_id = ?", space.ID)).
		Pluck("bucket_key", &versionKeys).Error; err != nil {
		return err
	}

	keys := append(fileKeys, versionKeys...)
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if err := dest.CopyFrom(ctx, s.storage, key); err != nil {
			return fmt.Errorf("copy %s: %w", key, err)
		}
	}

	if err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&schemas.FileVersion{}).
			Where("bucket = '' AND file_id IN (?)", tx.Model(&schemas.File{}).Select("id").Where("space_id = ?", space.ID)).
			Update("bucket", name).Error; err != nil {
			return err
		}
		return tx.Model(&schemas.File{}).
			Where("space_id = ? AND bucket = ''", space.ID).
			Update("bucket", name).Error
	}); err != nil {
		return err
	}

	for _, key := range keys {
		if err := s.storage.DeleteObject(ctx, key); err != nil {
			slog.Warn("spaces: failed to delete migrated object from the shared bucket", slog.String("key", key), slog.Any("error", err))
		}
	}
	slog.Info("spaces: moved space files to their own bucket", slog.Int64("space_id", space.ID), slog.String("bucket", name), slog.Int("objects", len(keys)))
	return nil
}
