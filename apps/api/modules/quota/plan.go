package quota

import (
	"context"
	"fmt"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// usageSQL sums the live bytes of one scope: every file not in the trash plus
// the versions it keeps. The scope filter is spliced in twice.
const usageSQL = `SELECT COALESCE((SELECT SUM(f.size) FROM files f WHERE f.deleted_at IS NULL AND %[1]s), 0)
	+ COALESCE((SELECT SUM(v.size) FROM file_versions v JOIN files f ON f.id = v.file_id
		WHERE f.deleted_at IS NULL AND %[1]s), 0)`

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

// enforce applies the admin override, then the plan of the scope the bytes
// land in, against that scope's usage counted under its lock.
func (s *Service) enforce(tx *gorm.DB, userID int64, spaceID *int64, adding int64) error {
	overridden, err := checkOverride(tx, userID, adding)
	if err != nil || (overridden && spaceID == nil) {
		return err
	}
	sc, err := s.scopeOf(tx, userID, spaceID)
	if err != nil {
		return err
	}
	used, err := lockedUsage(tx, userID, spaceID)
	if err != nil {
		return err
	}
	return s.plans.CheckBytes(sc, used, adding)
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

// scopeOf names the plan scope the bytes count against: the Registre space a
// space is linked to, or the person's Registre subject for personal files. A
// space that is not linked has no plan and is refused, never read as the
// uploader's personal plan.
func (s *Service) scopeOf(tx *gorm.DB, userID int64, spaceID *int64) (forfait.Scope, error) {
	if spaceID == nil {
		subject, err := s.subjectOf(tx, userID)
		return forfait.Scope{Subject: subject}, err
	}
	var linked struct{ RegistreSpaceID *string }
	if err := tx.Table("spaces").Select("registre_space_id").Where("id = ?", *spaceID).Limit(1).Scan(&linked).Error; err != nil {
		return forfait.Scope{}, errors.Internal("failed to load the space", err)
	}
	if linked.RegistreSpaceID == nil || *linked.RegistreSpaceID == "" {
		return forfait.Scope{}, errors.Failed("this space is not linked to Registre yet, so it has no plan to store new files under")
	}
	return forfait.Scope{SpaceID: *linked.RegistreSpaceID}, nil
}

// subjectOf is the user's Registre subject, or empty for an account with no
// identity under the issuer, which then reads the free plan.
func (s *Service) subjectOf(tx *gorm.DB, userID int64) (string, error) {
	var subject string
	err := tx.Table("porte_identities").Select("subject").
		Where("user_id = ? AND provider = ?", userID, s.plans.Issuer()).Limit(1).Scan(&subject).Error
	if err != nil {
		return "", errors.Internal("failed to load the identity", err)
	}
	return subject, nil
}

// lockedUsage takes the scope's transactional lock and sums its live bytes.
func lockedUsage(tx *gorm.DB, userID int64, spaceID *int64) (int64, error) {
	key := fmt.Sprintf("nuage:plan:user:%d", userID)
	if spaceID != nil {
		key = fmt.Sprintf("nuage:plan:space:%d", *spaceID)
	}
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error; err != nil {
		return 0, errors.Internal("failed to lock the storage scope", err)
	}
	return scopeUsage(tx, userID, spaceID)
}

// scopeUsage sums the live bytes of a space, or of the user's personal files.
func scopeUsage(db *gorm.DB, userID int64, spaceID *int64) (int64, error) {
	filter, arg := "f.space_id IS NULL AND f.uploaded_by = ?", userID
	if spaceID != nil {
		filter, arg = "f.space_id = ?", *spaceID
	}
	var used int64
	if err := db.Raw(fmt.Sprintf(usageSQL, filter), arg, arg).Scan(&used).Error; err != nil {
		return 0, errors.Internal("failed to measure storage usage", err)
	}
	return used, nil
}
