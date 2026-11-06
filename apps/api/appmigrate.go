package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/database"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/appfolders"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/logger"
)

// appMigrationCommand is the subcommand that moves app-provisioned spaces'
// files into their users' app folders.
const appMigrationCommand = "migrate-app-folders"

// runAppMigration runs the app folder migration with the server's own
// configuration and writes its JSON report to out. It is a dry run unless
// --apply is given. It returns the process exit code: 1 on a setup error or
// when any file failed to move.
func runAppMigration(args []string, out io.Writer) int {
	appLogger := logger.New(logger.Config{})
	flags := flag.NewFlagSet(appMigrationCommand, flag.ContinueOnError)
	apply := flags.Bool("apply", false, "move the files; without it nothing changes")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	report, err := migrateAppFolders(context.Background(), *apply, appLogger)
	if err != nil {
		appLogger.Error("app folder migration failed", slog.Any("error", err))
	}
	if report != nil {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(report)
	}
	if err != nil || (report != nil && report.Failed > 0) {
		return 1
	}
	return 0
}

// migrateAppFolders opens the database and storage from the environment,
// brings the schema and the registered apps up to date, and runs the
// migration.
func migrateAppFolders(ctx context.Context, apply bool, appLogger *slog.Logger) (*appfolders.MigrationReport, error) {
	appEnv, err := env.Load()
	if err != nil {
		return nil, err
	}
	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := schemas.MigrateWithIssuer(db, appEnv.IssuerForMigration()); err != nil {
		return nil, err
	}
	if err := appregistry.Sync(ctx, db, appEnv.Apps); err != nil {
		return nil, err
	}
	store, err := storage.NewClient(storage.MinIOConfig{
		Endpoint: appEnv.MinIO.Endpoint, AccessKey: appEnv.MinIO.AccessKey, SecretKey: appEnv.MinIO.SecretKey,
		Bucket: appEnv.MinIO.Bucket, UseSSL: appEnv.MinIO.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	service := appfolders.NewService(db, store, quota.NewService(db), appLogger)
	return service.Migrate(ctx, apply)
}
