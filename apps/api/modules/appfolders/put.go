package appfolders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Upload is one object an app sends.
type Upload struct {
	Key      string
	Name     string
	MimeType string
	Size     int64
	Body     io.Reader
}

// Put stores upload in app's folder for userID, creating the folder on first
// use and replacing an object already stored under the same key. created
// reports whether the key was new. The plan admits only the growth, checked
// early and again under the person's plan lock in the transaction that
// records the object.
func (s *Service) Put(ctx context.Context, userID int64, app *schemas.RegisteredApp, upload Upload) (*schemas.AppObject, bool, error) {
	if err := validKey(upload.Key); err != nil {
		return nil, false, err
	}
	name, err := objectName(upload.Name, upload.Key)
	if err != nil {
		return nil, false, err
	}
	if upload.Size > MaxObjectSize {
		return nil, false, errors.TooLarge("object is too large")
	}
	if err := s.admitEarly(ctx, userID, app.ID, upload); err != nil {
		return nil, false, err
	}
	bucketKey := fmt.Sprintf("apps/%d/%s/%s", userID, app.Key, facile.NewID())
	stored, err := s.storeBytes(ctx, bucketKey, upload)
	if err != nil {
		return nil, false, err
	}
	stored.Key, stored.Name, stored.MimeType = upload.Key, name, mediaType(upload.MimeType)
	object, replaced, err := s.record(ctx, userID, app.ID, stored)
	if err != nil {
		s.removeObjects(ctx, []string{bucketKey})
		return nil, false, err
	}
	s.settle(ctx, userID, object, replaced)
	return object, replaced == nil, nil
}

// settle removes the bytes a replacement superseded and moves the legacy
// usage counter by the growth.
func (s *Service) settle(ctx context.Context, userID int64, object, replaced *schemas.AppObject) {
	growth := object.Size
	if replaced != nil {
		growth -= replaced.Size
		s.removeObjects(ctx, []string{replaced.BucketKey})
	}
	if err := s.quota.UpdateUsage(ctx, userID, growth); err != nil {
		s.logger.Error("failed to update the legacy usage counter", slog.Int64("user_id", userID), slog.Any("error", err))
	}
}

// admitEarly refuses before any byte is stored when the declared size cannot
// fit, counting only the growth over an object the key already holds.
func (s *Service) admitEarly(ctx context.Context, userID, appID int64, upload Upload) error {
	if upload.Size <= 0 {
		return nil
	}
	var current int64
	if err := s.orm.WithContext(ctx).Table("app_objects o").Select("o.size").
		Joins("JOIN app_folders af ON af.id = o.folder_id").
		Where("af.user_id = ? AND af.app_id = ? AND o.key = ?", userID, appID, upload.Key).
		Limit(1).Scan(&current).Error; err != nil {
		return errors.Internal("failed to read the object", err)
	}
	return s.quota.Admit(ctx, userID, nil, upload.Size-current)
}

// storeBytes streams the body to bucketKey, hashing it on the way, and
// returns the row to record. It reads at most one byte past MaxObjectSize, so
// a body with no declared size cannot grow past the cap, and a body whose
// length differs from its declared size is refused.
func (s *Service) storeBytes(ctx context.Context, bucketKey string, upload Upload) (*schemas.AppObject, error) {
	hasher := sha256.New()
	counter := &countingReader{reader: io.TeeReader(io.LimitReader(upload.Body, MaxObjectSize+1), hasher)}
	size := upload.Size
	if size <= 0 {
		size = -1
	}
	if err := s.store.PutObject(ctx, bucketKey, counter, size, mediaType(upload.MimeType)); err != nil {
		var tooLarge *http.MaxBytesError
		if stderrors.As(err, &tooLarge) {
			return nil, errors.TooLarge("object is too large")
		}
		return nil, errors.Internal("failed to store the object", err)
	}
	if counter.read > MaxObjectSize || (upload.Size > 0 && counter.read != upload.Size) {
		s.removeObjects(ctx, []string{bucketKey})
		if counter.read > MaxObjectSize {
			return nil, errors.TooLarge("object is too large")
		}
		return nil, errors.Invalid("body size does not match Content-Length")
	}
	return &schemas.AppObject{Size: counter.read, Hash: hex.EncodeToString(hasher.Sum(nil)), BucketKey: bucketKey}, nil
}

// record writes object into the folder in one transaction: it creates the
// folder if needed, serialises writers of the same key, admits the growth
// under the person's plan lock (taken last) and inserts or replaces the row.
// It returns the previous row when one was replaced.
func (s *Service) record(ctx context.Context, userID, appID int64, object *schemas.AppObject) (*schemas.AppObject, *schemas.AppObject, error) {
	var replaced *schemas.AppObject
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		folderID, err := ensureFolder(tx, userID, appID)
		if err != nil {
			return err
		}
		existing, err := lockedObject(tx, folderID, object.Key)
		if err != nil {
			return err
		}
		growth := object.Size
		if existing != nil {
			growth -= existing.Size
		}
		if err := s.quota.AdmitTx(tx, userID, nil, growth); err != nil {
			return err
		}
		object.FolderID, replaced = folderID, existing
		return writeRow(tx, object, existing)
	})
	if err != nil {
		return nil, nil, err
	}
	return object, replaced, nil
}

// lockedObject takes the key's write lock and returns the row it holds, or nil.
func lockedObject(tx *gorm.DB, folderID int64, key string) (*schemas.AppObject, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", objectLock(folderID, key)).Error; err != nil {
		return nil, errors.Internal("failed to lock the object", err)
	}
	var existing []schemas.AppObject
	if err := tx.Where("folder_id = ? AND key = ?", folderID, key).Limit(1).Find(&existing).Error; err != nil {
		return nil, errors.Internal("failed to read the object", err)
	}
	if len(existing) == 0 {
		return nil, nil
	}
	return &existing[0], nil
}

// writeRow inserts object, or overwrites existing with it in place.
func writeRow(tx *gorm.DB, object, existing *schemas.AppObject) error {
	var err error
	if existing == nil {
		err = tx.Create(object).Error
	} else {
		object.ID, object.CreatedAt = existing.ID, existing.CreatedAt
		err = tx.Model(&schemas.AppObject{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"name": object.Name, "mime_type": object.MimeType, "size": object.Size,
			"hash": object.Hash, "bucket_key": object.BucketKey, "updated_at": gorm.Expr("NOW()"),
		}).Error
	}
	if err != nil {
		return errors.Internal("failed to record the object", err)
	}
	return nil
}

// ensureFolder returns the id of userID's folder for appID, creating it, and
// holds a key-share lock on the row until the transaction ends so the folder
// cannot be deleted under the write.
func ensureFolder(tx *gorm.DB, userID, appID int64) (int64, error) {
	folder := schemas.AppFolder{UserID: userID, AppID: appID}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&folder).Error; err != nil {
		return 0, errors.Internal("failed to create the app folder", err)
	}
	var ids []int64
	if err := tx.Model(&schemas.AppFolder{}).Clauses(clause.Locking{Strength: "KEY SHARE"}).
		Where("user_id = ? AND app_id = ?", userID, appID).Pluck("id", &ids).Error; err != nil {
		return 0, errors.Internal("failed to read the app folder", err)
	}
	if len(ids) == 0 {
		return 0, errors.Conflict("the app folder was deleted during the upload; retry")
	}
	return ids[0], nil
}
