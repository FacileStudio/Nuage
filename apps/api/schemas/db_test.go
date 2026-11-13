package schemas

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/migrations"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// openTestDatabase hands back a connection scoped to a schema of its own, built by
// the migrations, so these tests can write `users` without disturbing the integration
// suite sharing the database. The backfills are written in PostgreSQL (regexp_replace,
// anchored), so testing them anywhere else would test a different statement than the
// one that ships.
func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db := openScratchSchema(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := migrations.Up(context.Background(), migrations.Config(sqlDB, slog.New(slog.DiscardHandler))); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// openScratchSchema hands back a connection scoped to a new, empty schema that is
// dropped when the test ends.
//
// A developer without Postgres still gets a usable suite; CI, where the
// database is always there, would fail on the same condition rather than
// skip silently.
func openScratchSchema(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://nuage:nuage-internal-db@localhost:5432/nuage_test?sslmode=disable"
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		databaseUnavailable(t, err)
	}
	schema := fmt.Sprintf("avatar_test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		databaseUnavailable(t, err)
	}
	t.Cleanup(func() {
		admin.Exec("DROP SCHEMA " + schema + " CASCADE")
		if handle, err := admin.DB(); err == nil {
			_ = handle.Close()
		}
	})
	return openScoped(t, dsn, schema)
}

// openScoped connects with schema first on the search_path. The schema travels in
// the connection string rather than in a SET: tronc/migrate holds one pooled
// connection for its lock and migrates through another, and a SET binds only the
// connection that ran it.
func openScoped(t *testing.T, dsn, schema string) *gorm.DB {
	t.Helper()
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	db, err := gorm.Open(postgres.Open(dsn+separator+"search_path="+schema), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatalf("open %s: %v", schema, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open %s: %v", schema, err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// databaseUnavailable skips without a database and fails in CI, where one is required.
func databaseUnavailable(t *testing.T, err error) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("integration test infrastructure required in CI: %v", err)
	}
	t.Skipf("skipping: database not available: %v", err)
}
