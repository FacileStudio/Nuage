package quota

import (
	"fmt"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// usageSQL sums the stored bytes of one scope: every file, trashed ones
// included until they are purged, plus the versions it keeps. The scope filter
// is spliced in twice.
const usageSQL = `SELECT COALESCE((SELECT SUM(f.size) FROM files f WHERE %[1]s), 0)
	+ COALESCE((SELECT SUM(v.size) FROM file_versions v JOIN files f ON f.id = v.file_id WHERE %[1]s), 0)`

// reservedSQL sums the declared size of the scope's chunked uploads still
// receiving chunks. A session being assembled is not counted: its completion
// is checked on its own, inside the transaction that records the file.
const reservedSQL = `SELECT COALESCE(SUM(s.total_size), 0) FROM upload_sessions s
	WHERE s.status = 'pending' AND s.expires_at > now() AND %s`

// target is what an addition is charged to: a space linked to Registre, with
// its own plan, or a person. A person's plan covers their personal files and
// every app-provisioned space charged to them, all under one lock, so an app
// cannot give each space it provisions a fresh free plan.
type target struct {
	scope    forfait.Scope
	personID int64
	spaceID  int64
}

// lockKey names the transactional lock that serialises additions to target.
func (t target) lockKey() string {
	if t.spaceID != 0 {
		return fmt.Sprintf("nuage:plan:space:%d", t.spaceID)
	}
	return fmt.Sprintf("nuage:plan:user:%d", t.personID)
}

// filter is the WHERE clause, on the table aliased alias, selecting the rows
// charged to target, with its arguments. ownerColumn names the user column of
// personal rows.
func (t target) filter(alias, ownerColumn string) (string, []any) {
	if t.spaceID != 0 {
		return alias + ".space_id = ?", []any{t.spaceID}
	}
	clause := fmt.Sprintf(`((%[1]s.space_id IS NULL AND %[1]s.%[2]s = ?)
		OR %[1]s.space_id IN (SELECT sp.id FROM spaces sp WHERE sp.charged_to = ? AND sp.registre_space_id IS NULL))`, alias, ownerColumn)
	return clause, []any{t.personID, t.personID}
}

// targetOf resolves what bytes landing in spaceID, or in userID's personal
// files when it is nil, are charged to. Server-held state decides: a linked
// space pays for itself, a space another app provisioned is charged to the
// user recorded when it was created, and any other space has no plan and is
// refused rather than read as the uploader's personal plan.
func (s *Service) targetOf(tx *gorm.DB, userID int64, spaceID *int64) (target, error) {
	if spaceID == nil {
		return s.personTarget(tx, userID)
	}
	var row struct {
		RegistreSpaceID *string
		ProvisionedBy   string
		ChargedTo       *int64
	}
	if err := tx.Table("spaces").Select("registre_space_id, provisioned_by, charged_to").
		Where("id = ?", *spaceID).Limit(1).Scan(&row).Error; err != nil {
		return target{}, errors.Internal("failed to load the space", err)
	}
	if row.RegistreSpaceID != nil && *row.RegistreSpaceID != "" {
		return target{scope: forfait.Scope{SpaceID: *row.RegistreSpaceID}, spaceID: *spaceID}, nil
	}
	if row.ProvisionedBy != "" && row.ChargedTo != nil {
		return s.personTarget(tx, *row.ChargedTo)
	}
	return target{}, errors.Failed("this space is not linked to Registre yet, so it has no plan to store new files under")
}

// personTarget is the person's plan, read under their Registre subject, or
// the free plan for an account with no identity under the issuer.
func (s *Service) personTarget(tx *gorm.DB, personID int64) (target, error) {
	var subject string
	err := tx.Table("porte_identities").Select("subject").
		Where("user_id = ? AND provider = ?", personID, s.plans.Issuer()).Limit(1).Scan(&subject).Error
	if err != nil {
		return target{}, errors.Internal("failed to load the identity", err)
	}
	return target{scope: forfait.Scope{Subject: subject}, personID: personID}, nil
}

// usageOf sums the stored bytes charged to target.
func usageOf(db *gorm.DB, t target) (int64, error) {
	filter, args := t.filter("f", "uploaded_by")
	var used int64
	if err := db.Raw(fmt.Sprintf(usageSQL, filter), append(args, args...)...).Scan(&used).Error; err != nil {
		return 0, errors.Internal("failed to measure storage usage", err)
	}
	return used, nil
}

// lockedUsage takes target's transactional lock and sums its stored bytes and
// the bytes its open chunked uploads have reserved.
func lockedUsage(tx *gorm.DB, t target) (int64, error) {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", t.lockKey()).Error; err != nil {
		return 0, errors.Internal("failed to lock the storage scope", err)
	}
	used, err := usageOf(tx, t)
	if err != nil {
		return 0, err
	}
	filter, args := t.filter("s", "user_id")
	var reserved int64
	if err := tx.Raw(fmt.Sprintf(reservedSQL, filter), args...).Scan(&reserved).Error; err != nil {
		return 0, errors.Internal("failed to measure reserved uploads", err)
	}
	return used + reserved, nil
}
