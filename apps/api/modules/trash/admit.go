package trash

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// restoredBytesSQL sums what restoring the matching trashed files adds back
// to their scope: each file and the versions it keeps.
const restoredBytesSQL = `SELECT COALESCE(SUM(f.size + COALESCE(
	(SELECT SUM(v.size) FROM file_versions v WHERE v.file_id = f.id), 0)), 0) FROM files f WHERE `

// restoreFile takes one file out of the trash, inside the transaction that
// checks its bytes still fit the scope's plan.
func (s *Service) restoreFile(ctx context.Context, userID int64, record *schemas.File, updates map[string]any) error {
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.admitRestore(tx, userID, record.SpaceID, "f.id = ?", record.ID); err != nil {
			return err
		}
		return tx.Model(record).Updates(updates).Error
	})
	return restoreFailure(err, "failed to restore file")
}

// admitRestore refuses a restore that would take the scope over its plan.
// Restored files count again, since trashed ones do not.
func (s *Service) admitRestore(tx *gorm.DB, userID int64, spaceID *int64, where string, args ...any) error {
	if s.quota == nil {
		return nil
	}
	var adding int64
	if err := tx.Raw(restoredBytesSQL+where, args...).Scan(&adding).Error; err != nil {
		return err
	}
	return s.quota.AdmitTx(tx, userID, spaceID, adding)
}

// restoreFailure keeps a plan refusal or an API error as it is and wraps any
// other failure as an internal error.
func restoreFailure(err error, message string) error {
	if err == nil {
		return nil
	}
	var apiErr *errors.Error
	if stderrors.Is(err, forfait.ErrLimit) || stderrors.As(err, &apiErr) {
		return err
	}
	return errors.Internal(message, err)
}
