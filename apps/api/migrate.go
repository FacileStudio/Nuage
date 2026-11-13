package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"

	"github.com/FacileStudio/Nuage/apps/api/migrations"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/migrate"

	"gorm.io/gorm"
)

// migrateOrAnswer answers the migrate subcommand (status, up, version,
// baseline <n>) when it was asked for, and otherwise applies the pending
// migrations and the boot data steps. It reports whether the process must exit
// now, and with which code: a failed migration stops the boot with 1, so a
// deploy never serves a half-migrated schema.
func migrateOrAnswer(ctx context.Context, db *gorm.DB, sqlDB *sql.DB, issuer string, appLogger *slog.Logger) (bool, int) {
	if handled, err := migrate.Command(ctx, os.Args, migrations.Config(sqlDB, appLogger)); handled {
		if err != nil {
			appLogger.Error("migrate", slog.Any("error", err))
			return true, 1
		}
		return true, 0
	}
	if err := schemas.MigrateWithIssuer(ctx, db, issuer, appLogger); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return true, 1
	}
	return false, 0
}
