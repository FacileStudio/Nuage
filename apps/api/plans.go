package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"

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
