package quota

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/porte/forfait"
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

// planUsage is the personal usage the plan counts, against the person's plan
// ceiling in bytes, with the Registre page where it is upgraded.
func (s *Service) planUsage(ctx context.Context, userID int64) (*UsageResponse, error) {
	db := s.orm.WithContext(ctx)
	used, err := scopeUsage(db, userID, nil)
	if err != nil {
		return nil, err
	}
	sc, err := s.scopeOf(db, userID, nil)
	if err != nil {
		return nil, err
	}
	limit := s.plans.LimitBytes(sc)
	response := &UsageResponse{
		UserID: userID, StorageUsed: used, StorageLimit: limit,
		Percentage: percentOf(used, limit), Source: SourcePlan,
	}
	if issuer := s.plans.Issuer(); issuer != "" {
		response.UpgradeURL = forfait.UpgradeURL(issuer, "")
	}
	return response, nil
}
