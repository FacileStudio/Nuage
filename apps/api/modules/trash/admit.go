package trash

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// restoredBytes is what a restore adds to its scope's usage: nothing, since
// trashed files keep counting until they are purged. AdmitTx returns at once
// for zero bytes and takes no plan lock; the call stays so a restore is checked
// if that accounting ever changes.
const restoredBytes int64 = 0

// restoreFile takes one file out of the trash. It locks the file row before
// anything else, so if restores ever take the plan lock they do so after the
// row lock, the order writeVersion and restoreVersion use.
func (s *Service) restoreFile(ctx context.Context, userID int64, record *schemas.File, updates map[string]any) error {
	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.admitRestore(tx, userID, record.SpaceID, "id = ?", record.ID); err != nil {
			return err
		}
		return tx.Model(record).Updates(updates).Error
	})
	return restoreFailure(err, "failed to restore file")
}

// admitRestore locks the trashed files a restore brings back, in id order,
// and then checks the scope's plan for what the restore adds, which is
// restoredBytes.
func (s *Service) admitRestore(tx *gorm.DB, userID int64, spaceID *int64, where string, args ...any) error {
	var locked []int64
	if err := tx.Model(&schemas.File{}).Where(where, args...).Where("deleted_at IS NOT NULL").
		Order("id").Clauses(clause.Locking{Strength: "UPDATE"}).Pluck("id", &locked).Error; err != nil {
		return err
	}
	return s.quota.AdmitTx(tx, userID, spaceID, restoredBytes)
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
