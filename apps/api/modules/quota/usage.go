package quota

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/porte/forfait"

	"gorm.io/gorm"
)

// WithPlans makes the Service enforce the plan entitlements once Registre has
// served them, and returns it. It must be called before the Service is shared.
func (s *Service) WithPlans(plans *entitlements.Plans) *Service {
	s.plans = plans
	return s
}

// percentOf is used as a share of limit, capped at 100, and 0 when unlimited.
func percentOf(used, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	return min(float64(used)/float64(limit)*100, 100)
}

// planUsage is the personal usage the plan counts, app folders and
// app-provisioned spaces charged to the person included, against the ceiling
// uploads are held to: the admin override when there is one (override != 0),
// else the person's plan, with the Registre page where it is upgraded.
func (s *Service) planUsage(ctx context.Context, userID int64, override int64) (*UsageResponse, error) {
	db := s.orm.WithContext(ctx)
	person, err := s.personTarget(db, userID)
	if err != nil {
		return nil, err
	}
	used, err := usageOf(db, person)
	if err != nil {
		return nil, err
	}
	source, limit := SourceOverride, override
	if override == 0 {
		source, limit = SourcePlan, s.plans.LimitBytes(person.scope)
	}
	response := &UsageResponse{
		UserID: userID, StorageUsed: used, StorageLimit: limit,
		Percentage: percentOf(used, limit), Source: source,
	}
	if issuer := s.plans.Issuer(); issuer != "" && source == SourcePlan {
		response.UpgradeURL = forfait.UpgradeURL(issuer, "")
	}
	return response, nil
}

// Reserve opens a chunked upload of size bytes. Once plans apply it admits the
// declared size inside the transaction that runs create, under the scope's
// lock and counting what the scope's other open uploads reserved, so parallel
// sessions cannot stage more than the plan allows. Before that it is the old
// per-user check, followed by create.
func (s *Service) Reserve(ctx context.Context, userID int64, spaceID *int64, size int64, create func(tx *gorm.DB) error) error {
	if !s.plans.Enforcing() {
		if err := s.CheckQuota(ctx, userID, size); err != nil {
			return err
		}
	}
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.AdmitTx(tx, userID, spaceID, size); err != nil {
			return err
		}
		return create(tx)
	})
}

// Readmit runs touch in a transaction that first admits size bytes for a
// lapsed chunked upload under the plan lock. Before plans apply it only runs
// touch, as nothing was reserved.
func (s *Service) Readmit(ctx context.Context, userID int64, spaceID *int64, size int64, touch func(tx *gorm.DB) error) error {
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.AdmitTx(tx, userID, spaceID, size); err != nil {
			return err
		}
		return touch(tx)
	})
}
