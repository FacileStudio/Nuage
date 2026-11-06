package appfolders

import (
	"context"
	"fmt"

	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/internal/tombstone"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// move copies the file's bytes into the shared bucket, then in one
// transaction records the object (unless it deduplicates) and deletes the
// file row with a tombstone for sync clients. The source bytes go last, once
// nothing references them.
func (r *spaceRun) move(ctx context.Context, file schemas.File, entry FileReport) error {
	source := r.service.store
	if file.Bucket != "" {
		source = r.service.store.For(file.Bucket)
	}
	object, err := r.stage(ctx, source, file, entry)
	if err != nil {
		return err
	}
	moved, err := r.commit(ctx, file, entry.Key, object)
	if err == nil && !moved {
		err = fmt.Errorf("the file changed since it was planned; rerun to retry")
	}
	if err != nil {
		if object != nil {
			r.service.removeObjects(ctx, []string{object.BucketKey})
		}
		return err
	}
	r.service.dropSource(ctx, source, file)
	r.service.shiftUsage(ctx, file, r.space.ChargedTo, object != nil)
	return nil
}

// stage copies a file that moves into the app's area of the shared bucket and
// returns the object to record, or nil for a deduplication, which copies
// nothing.
func (r *spaceRun) stage(ctx context.Context, source *storage.Client, file schemas.File, entry FileReport) (*schemas.AppObject, error) {
	if entry.Action != ActionMove {
		return nil, nil
	}
	destKey := fmt.Sprintf("apps/%d/%s/%s", r.space.ChargedTo, r.app.Key, file.FacileID)
	if err := r.service.store.CopyFromKey(ctx, source, file.BucketKey, destKey); err != nil {
		return nil, fmt.Errorf("copy the bytes: %w", err)
	}
	return &schemas.AppObject{Key: entry.Key, Name: file.Name, MimeType: file.MimeType,
		Size: file.Size, Hash: file.Hash, BucketKey: destKey, SourceFileID: &file.ID}, nil
}

// commit is move's transaction; object is nil for a deduplication. It
// reports false when the file is no longer a live row of the space, or the key
// changed hands since it was planned.
func (r *spaceRun) commit(ctx context.Context, file schemas.File, key string, object *schemas.AppObject) (bool, error) {
	moved := false
	err := r.service.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		folderID, live, err := r.lockMove(tx, file)
		if err != nil || live == nil {
			return err
		}
		taken, err := lockedObject(tx, folderID, key)
		if err != nil || (object != nil) != (taken == nil) {
			return err
		}
		moved = true
		return transfer(ctx, tx, *live, folderID, object)
	})
	return moved, err
}

// transfer records the object for a move and deletes the file row, leaving a
// tombstone for sync clients.
func transfer(ctx context.Context, tx *gorm.DB, file schemas.File, folderID int64, object *schemas.AppObject) error {
	if object != nil {
		object.FolderID = folderID
		if err := tx.Create(object).Error; err != nil {
			return err
		}
	}
	if err := tombstone.RecordFiles(ctx, tx, []schemas.File{file}); err != nil {
		return err
	}
	return tx.Unscoped().Delete(&schemas.File{}, file.ID).Error
}

// lockMove ensures the target folder and locks the file row before the key
// lock commit takes next, returning nil
// for the row when it is no longer a live file of the space.
func (r *spaceRun) lockMove(tx *gorm.DB, file schemas.File) (int64, *schemas.File, error) {
	folderID, err := ensureFolder(tx, r.space.ChargedTo, r.app.ID)
	if err != nil {
		return 0, nil, err
	}
	var live []schemas.File
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND space_id = ? AND deleted_at IS NULL", file.ID, r.space.ID).Limit(1).Find(&live).Error; err != nil {
		return 0, nil, err
	}
	if len(live) == 0 {
		return folderID, nil, nil
	}
	return folderID, &live[0], nil
}

// dropSource removes the moved file's bytes and cached thumbnails from its
// space bucket. A failure leaves an unreferenced object, never lost data.
func (s *Service) dropSource(ctx context.Context, source *storage.Client, file schemas.File) {
	if err := source.DeleteObject(ctx, file.BucketKey); err != nil {
		s.logger.Warn("migration: failed to remove a moved file's source bytes", "file_id", file.ID, "error", err)
	}
	if err := source.DeletePrefix(ctx, fmt.Sprintf("thumbs/%d-", file.ID)); err != nil {
		s.logger.Warn("migration: failed to remove a moved file's thumbnails", "file_id", file.ID, "error", err)
	}
}

// shiftUsage moves the legacy counter: the uploader stops paying for the file
// and, unless it deduplicated, the folder's owner starts.
func (s *Service) shiftUsage(ctx context.Context, file schemas.File, owner int64, moved bool) {
	if err := s.quota.UpdateUsage(ctx, file.UploadedBy, -file.Size); err != nil {
		s.logger.Warn("migration: failed to update the legacy usage counter", "user_id", file.UploadedBy, "error", err)
	}
	if !moved {
		return
	}
	if err := s.quota.UpdateUsage(ctx, owner, file.Size); err != nil {
		s.logger.Warn("migration: failed to update the legacy usage counter", "user_id", owner, "error", err)
	}
}
