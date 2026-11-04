package quota

import (
	"context"

	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Admit reports whether userID may add adding bytes to the space, or to their
// personal files when spaceID is nil. It is the early check that runs before
// the bytes are uploaded; AdmitTx repeats it where the bytes are recorded.
//
// Until Registre has served an authoritative entitlements snapshot it is the
// per-user quota Nuage always had. After that the plan decides, except that
// an explicit per-user limit an admin set keeps applying as an override.
func (s *Service) Admit(ctx context.Context, userID int64, spaceID *int64, adding int64) error {
	if s == nil || adding <= 0 {
		return nil
	}
	if !s.plans.Enforcing() {
		return s.CheckQuota(ctx, userID, adding)
	}
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.enforce(tx, userID, spaceID, adding)
	})
}

// AdmitTx is Admit inside the transaction that records the bytes. It locks the
// scope until that transaction ends, so two uploads racing at the ceiling
// cannot both pass. Before enforcement starts it does nothing, as before.
func (s *Service) AdmitTx(tx *gorm.DB, userID int64, spaceID *int64, adding int64) error {
	if s == nil || adding <= 0 || !s.plans.Enforcing() {
		return nil
	}
	return s.enforce(tx, userID, spaceID, adding)
}

// enforce applies the admin override, then the plan of whatever the bytes are
// charged to, against its usage counted under its lock. For a person the
// override is that person's and replaces the plan; for a linked space the
// uploader's override still caps the upload and the space's plan applies too.
func (s *Service) enforce(tx *gorm.DB, userID int64, spaceID *int64, adding int64) error {
	charged, err := s.targetOf(tx, userID, spaceID)
	if err != nil {
		return err
	}
	overrideOf := userID
	if charged.personID != 0 {
		overrideOf = charged.personID
	}
	overridden, err := checkOverride(tx, overrideOf, adding)
	if err != nil || (overridden && charged.personID != 0) {
		return err
	}
	used, err := lockedUsage(tx, charged)
	if err != nil {
		return err
	}
	return s.plans.CheckBytes(charged.scope, used, adding)
}

// checkOverride applies the storage limit an admin set on the user, when there
// is one, and reports whether there is. The override counts every byte the
// user uploaded, as the per-user quota always has.
func checkOverride(tx *gorm.DB, userID int64, adding int64) (bool, error) {
	var row struct {
		StorageUsed  int64
		StorageLimit int64
	}
	err := tx.Table("user_quotas").Select("storage_used, storage_limit").
		Where("user_id = ?", userID).Limit(1).Scan(&row).Error
	if err != nil {
		return false, errors.Internal("failed to load quota", err)
	}
	if row.StorageLimit == 0 {
		return false, nil
	}
	if row.StorageLimit > 0 && row.StorageUsed > row.StorageLimit-adding {
		return true, errors.TooLarge("storage quota exceeded")
	}
	return true, nil
}
