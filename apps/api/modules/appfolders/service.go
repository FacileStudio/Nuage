// Package appfolders stores each registered app's private data per user, like
// an app's iCloud container. An app reaches only the folder of the user its
// delegated token names, and only its own app's folder; the user sees the
// usage of each and can delete one. Folders are never spaces and never show in
// the file tree, and their bytes count toward the user's personal plan.
package appfolders

import (
	"context"
	"io"
	"log/slog"

	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service reads and writes app folders.
type Service struct {
	orm    *gorm.DB
	store  *storage.Client
	quota  *quota.Service
	logger *slog.Logger
}

// NewService builds the app folder Service. Objects live in store's bucket,
// the shared one personal files use.
func NewService(orm *gorm.DB, store *storage.Client, quotaService *quota.Service, logger *slog.Logger) *Service {
	return &Service{orm: orm, store: store, quota: quotaService, logger: logger}
}

// Find returns the object stored under key in app's folder for userID.
func (s *Service) Find(ctx context.Context, userID, appID int64, key string) (*schemas.AppObject, error) {
	if err := validKey(key); err != nil {
		return nil, err
	}
	var rows []schemas.AppObject
	if err := s.orm.WithContext(ctx).Table("app_objects o").Select("o.*").
		Joins("JOIN app_folders af ON af.id = o.folder_id").
		Where("af.user_id = ? AND af.app_id = ? AND o.key = ?", userID, appID, key).
		Limit(1).Find(&rows).Error; err != nil {
		return nil, errors.Internal("failed to read the object", err)
	}
	if len(rows) == 0 {
		return nil, errors.NotFound("object not found")
	}
	return &rows[0], nil
}

// Open returns the stored bytes of object.
func (s *Service) Open(ctx context.Context, object *schemas.AppObject) (io.ReadSeekCloser, error) {
	content, err := s.store.GetObject(ctx, object.BucketKey)
	if err != nil {
		return nil, errors.Internal("failed to open the object", err)
	}
	return content, nil
}

// ListQuery selects a page of objects: keys starting with Prefix, after the
// key After, at most Limit of them.
type ListQuery struct {
	Prefix string
	After  string
	Limit  int
}

// List returns a page of app's folder for userID, in byte order of the keys.
func (s *Service) List(ctx context.Context, userID, appID int64, page ListQuery) ([]schemas.AppObject, error) {
	prefix, after, limit := page.Prefix, page.After, page.Limit
	if err := validPrefix(prefix); err != nil {
		return nil, err
	}
	if err := validPrefix(after); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultListLimit
	}
	limit = min(limit, maxListLimit+1)
	query := s.orm.WithContext(ctx).Table("app_objects o").Select("o.*").
		Joins("JOIN app_folders af ON af.id = o.folder_id").
		Where("af.user_id = ? AND af.app_id = ?", userID, appID)
	if prefix != "" {
		query = query.Where("left(o.key, ?) = ?", len([]rune(prefix)), prefix)
	}
	if after != "" {
		query = query.Where("o.key COLLATE \"C\" > ?", after)
	}
	objects := []schemas.AppObject{}
	if err := query.Order("o.key COLLATE \"C\"").Limit(limit).Find(&objects).Error; err != nil {
		return nil, errors.Internal("failed to list the objects", err)
	}
	return objects, nil
}

// Delete removes the object stored under key in app's folder for userID. It
// takes the same per-key lock as Put, so a delete never interleaves with a
// replacement of the same key.
func (s *Service) Delete(ctx context.Context, userID, appID int64, key string) error {
	if err := validKey(key); err != nil {
		return err
	}
	var removed []schemas.AppObject
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var folderIDs []int64
		if err := tx.Model(&schemas.AppFolder{}).Where("user_id = ? AND app_id = ?", userID, appID).
			Pluck("id", &folderIDs).Error; err != nil {
			return errors.Internal("failed to read the app folder", err)
		}
		if len(folderIDs) == 0 {
			return nil
		}
		if _, err := lockedObject(tx, folderIDs[0], key); err != nil {
			return err
		}
		return tx.Raw("DELETE FROM app_objects WHERE folder_id = ? AND key = ? RETURNING *", folderIDs[0], key).
			Scan(&removed).Error
	})
	if err != nil {
		return errors.Internal("failed to delete the object", err)
	}
	if len(removed) == 0 {
		return errors.NotFound("object not found")
	}
	s.refund(ctx, userID, removed)
	return nil
}

// refund removes the stored bytes of the deleted rows and gives them back to
// the legacy usage counter.
func (s *Service) refund(ctx context.Context, userID int64, removed []schemas.AppObject) {
	keys := make([]string, 0, len(removed))
	var bytes int64
	for _, object := range removed {
		keys = append(keys, object.BucketKey)
		bytes += object.Size
	}
	s.removeObjects(ctx, keys)
	if err := s.quota.UpdateUsage(ctx, userID, -bytes); err != nil {
		s.logger.Error("failed to update the legacy usage counter", slog.Int64("user_id", userID), slog.Any("error", err))
	}
}

// removeObjects deletes stored bytes, logging what it could not remove: the
// rows are already gone, so a leftover is an orphan, never visible data.
func (s *Service) removeObjects(ctx context.Context, keys []string) {
	for _, key := range keys {
		if err := s.store.DeleteObject(context.WithoutCancel(ctx), key); err != nil {
			s.logger.Warn("failed to remove an app object's bytes", slog.String("bucket_key", key), slog.Any("error", err))
		}
	}
}
