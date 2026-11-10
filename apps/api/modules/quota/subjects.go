package quota

import (
	"context"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// subjectsOf lists the user's subjects under the issuer, most recently synced
// first, then most recently created.
func (s *Service) subjectsOf(tx *gorm.DB, userID int64) ([]string, error) {
	var subjects []string
	err := tx.Table("porte_identities").Where("user_id = ? AND provider = ?", userID, s.plans.Issuer()).
		Order("synced_at DESC NULLS LAST, created_at DESC NULLS LAST, subject").Pluck("subject", &subjects).Error
	if err != nil {
		return nil, errors.Internal("failed to load the identity", err)
	}
	return subjects, nil
}

// pickSubject chooses the subject a person's plan is read under. A user can
// hold several identities under one issuer, such as one left from an older
// identity provider: the one Registre named in its last snapshot wins, else
// the most recent one.
func (s *Service) pickSubject(subjects []string) string {
	for _, subject := range subjects {
		if s.plans.Known(subject) {
			return subject
		}
	}
	if len(subjects) == 0 {
		return ""
	}
	return subjects[0]
}

// PersonScope is the scope a person's plan is read under: their Registre
// subject, or the free plan for an account with no identity under the issuer.
func (s *Service) PersonScope(ctx context.Context, userID int64) (forfait.Scope, error) {
	person, err := s.personTarget(s.orm.WithContext(ctx), userID)
	return person.scope, err
}
