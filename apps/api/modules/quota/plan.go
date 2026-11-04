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

// enforce checks the bytes against the ceiling of whatever they are charged
// to, using that target's usage counted under its lock: for a person, their
// personal files, every app-provisioned space charged to them and their open
// uploads. An admin limit on the person replaces only the plan's ceiling,
// never the usage, so co-members of a charged space stay capped too.
func (s *Service) enforce(tx *gorm.DB, userID int64, spaceID *int64, adding int64) error {
	charged, err := s.targetOf(tx, userID, spaceID)
	if err != nil {
		return err
	}
	used, err := lockedUsage(tx, charged)
	if err != nil {
		return err
	}
	if charged.personID != 0 {
		override, err := overrideOf(tx, charged.personID)
		if err != nil {
			return err
		}
		if override != 0 {
			return withinOverride(override, used, adding)
		}
	}
	return s.plans.CheckBytes(charged.scope, used, adding)
}

// overrideOf is the storage limit an admin set on the user in bytes: 0 when
// there is none, -1 for unlimited.
func overrideOf(tx *gorm.DB, userID int64) (int64, error) {
	var limit int64
	err := tx.Table("user_quotas").Select("storage_limit").Where("user_id = ?", userID).Limit(1).Scan(&limit).Error
	if err != nil {
		return 0, errors.Internal("failed to load quota", err)
	}
	return limit, nil
}

// withinOverride compares used+adding with an admin limit without overflow.
func withinOverride(limit, used, adding int64) error {
	if limit < 0 || used <= limit-adding {
		return nil
	}
	return errors.TooLarge("storage quota exceeded")
}
