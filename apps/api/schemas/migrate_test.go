package schemas

import (
	"strings"
	"testing"

	"gorm.io/gorm"
)

// seedPreAntenne rebuilds the shape production is in before this deploy: the
// delivery queue and its settings under the name the alert bus had while it was
// called Nook.
//
// The DDL is written out here rather than taken from a model because the model
// is gone. That is the point: these names now exist only in databases that
// predate the rename, and the only thing that still has to understand them is
// renameNookToAntenne.
func seedPreAntenne(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		`CREATE TABLE nook_deliveries (
			id bigserial PRIMARY KEY,
			event_type text NOT NULL,
			payload text NOT NULL,
			status text NOT NULL DEFAULT 'pending',
			attempts bigint NOT NULL DEFAULT 0,
			next_retry_at timestamptz,
			response_code bigint,
			response_body text,
			error_message text,
			latency_ms bigint,
			created_at timestamptz,
			delivered_at timestamptz)`,
		`CREATE INDEX idx_nook_deliveries_event_type ON nook_deliveries (event_type)`,
		`CREATE INDEX idx_nook_status_retry ON nook_deliveries (status, next_retry_at)`,
		`INSERT INTO nook_deliveries (event_type, payload, status, created_at)
		 VALUES ('file.uploaded', '{}', 'pending', now())`,
		`CREATE TABLE settings (key text PRIMARY KEY, value text, updated_at timestamptz)`,
		`INSERT INTO settings (key, value, updated_at) VALUES
			('nook_webhook_url', 'https://antenne.facile.studio/webhook/nuage', now()),
			('nook_enabled', 'true', now()),
			('instance_name', 'Nuage', now())`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("seed: %v\n%s", err, statement)
		}
	}
}

// No queued event and no configured webhook may be lost by this deploy, and
// running it twice must change nothing — a second instance starting behind the
// first replays preMigrate against a database already carrying the new names.
//
// The index names are asserted because AutoMigrate is the thing that punishes
// getting them wrong: it finds no index under the new name and silently builds
// a second copy of each on the same columns.
func TestRenameNookToAntenneKeepsTheQueue(t *testing.T) {
	db := openTestDatabase(t)
	seedPreAntenne(t, db)

	for _, pass := range []string{"first run", "second run"} {
		t.Run(pass, func(t *testing.T) {
			if err := renameNookToAntenne(db); err != nil {
				t.Fatalf("rename: %v", err)
			}
			if err := db.AutoMigrate(&AntenneDelivery{}, &Setting{}); err != nil {
				t.Fatalf("automigrate: %v", err)
			}

			if db.Migrator().HasTable("nook_deliveries") {
				t.Error("nook_deliveries is still there, so AutoMigrate built a second queue")
			}

			var queued int64
			if err := db.Table("antenne_deliveries").Count(&queued).Error; err != nil {
				t.Fatalf("count deliveries: %v", err)
			}
			if queued != 1 {
				t.Errorf("queued deliveries = %d, want 1", queued)
			}

			var indexes []string
			db.Raw(`SELECT indexname FROM pg_indexes
				WHERE schemaname = current_schema() AND tablename = 'antenne_deliveries'
				ORDER BY indexname`).Scan(&indexes)
			want := []string{"antenne_deliveries_pkey", "idx_antenne_deliveries_event_type", "idx_antenne_status_retry"}
			if strings.Join(indexes, ",") != strings.Join(want, ",") {
				t.Errorf("indexes = %v, want %v", indexes, want)
			}

			var sequence string
			db.Raw(`SELECT pg_get_serial_sequence('antenne_deliveries', 'id')`).Scan(&sequence)
			if !strings.HasSuffix(sequence, ".antenne_deliveries_id_seq") {
				t.Errorf("sequence = %q, want one named antenne_deliveries_id_seq", sequence)
			}

			var keys []string
			db.Raw(`SELECT key FROM settings ORDER BY key`).Scan(&keys)
			wantKeys := []string{"antenne_enabled", "antenne_webhook_url", "instance_name"}
			if strings.Join(keys, ",") != strings.Join(wantKeys, ",") {
				t.Errorf("settings keys = %v, want %v", keys, wantKeys)
			}

			var webhook string
			db.Raw(`SELECT value FROM settings WHERE key = 'antenne_webhook_url'`).Scan(&webhook)
			if webhook != "https://antenne.facile.studio/webhook/nuage" {
				t.Errorf("webhook URL = %q, want the seeded one", webhook)
			}
		})
	}
}
