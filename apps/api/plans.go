package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"

	"gorm.io/gorm"
)

// newPlans builds the plan entitlements and restores the ones saved before the last
// restart, so a reboot never fails open again once Registre has served plans. A database
// error while reading the saved row is retried until it succeeds rather than read as "no
// plans", which would fail open.
func newPlans(db *gorm.DB, appEnv env.Config, appLogger *slog.Logger) *entitlements.Plans {
	issuer := ""
	if appEnv.OIDC != nil {
		issuer = appEnv.OIDC.Issuer
	}
	plans := entitlements.New(db, issuer, appLogger)
	for delay := time.Second; ; delay = min(delay*2, 30*time.Second) {
		err := plans.Restore(context.Background())
		if err == nil {
			return plans
		}
		appLogger.Error("failed to read the saved plan entitlements, retrying", slog.Any("error", err), slog.Duration("retry_in", delay))
		time.Sleep(delay)
	}
}

// reportStorage logs, once at boot, what an operator must settle before plans
// cut anyone off: users holding several identities under the issuer, everyone
// over the free plan (to cover with Ardoise overrides before its first push),
// and, once plans apply, everyone over the ceiling they now resolve to.
func reportStorage(quotaService *quota.Service, appLogger *slog.Logger) {
	ctx := context.Background()
	quotaService.ReportIdentities(ctx, appLogger)
	quotaService.ReportOverFreePlan(ctx, appLogger)
	quotaService.ReportOverPlan(ctx, appLogger)
}
