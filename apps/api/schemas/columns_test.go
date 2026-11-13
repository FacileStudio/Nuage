package schemas

import (
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/tronc/mcp"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var queryModels = []any{
	&User{}, &Space{}, &SpaceMember{}, &File{}, &Folder{}, &Share{}, &Setting{}, &FileVersion{},
	&UploadSession{}, &UploadChunk{}, &UserQuota{}, &ActivityLog{}, &NookDelivery{}, &Tombstone{},
	&EntitlementState{}, &DirectoryDeletionState{}, &RegisteredApp{}, &AppFolder{}, &AppObject{}, &TokenScope{},
}

var porteTables = []string{"porte_identities", "porte_sessions", "porte_login_codes"}

func compatibleTypes(field *schema.Field) []string {
	kind := field.FieldType
	for kind.Kind() == reflect.Pointer {
		kind = kind.Elem()
	}
	switch {
	case kind == reflect.TypeFor[time.Time]():
		return []string{"timestamp with time zone"}
	case kind == reflect.TypeFor[[]byte]():
		return []string{"bytea"}
	case kind.Kind() == reflect.Slice:
		return []string{"jsonb"}
	}
	switch kind.Kind() {
	case reflect.String:
		return []string{"text", "character varying"}
	case reflect.Bool:
		return []string{"boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
		return []string{"bigint"}
	case reflect.Int32, reflect.Uint32:
		return []string{"integer", "bigint"}
	default:
		return nil
	}
}

func migratedColumnTypes(t *testing.T, db *gorm.DB, table string) map[string]string {
	t.Helper()
	var rows []struct {
		ColumnName string
		DataType   string
	}
	err := db.Raw(`SELECT column_name, data_type FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = ?`, table).Scan(&rows).Error
	if err != nil {
		t.Fatalf("read the columns of %s: %v", table, err)
	}
	columns := map[string]string{}
	for _, row := range rows {
		columns[row.ColumnName] = row.DataType
	}
	return columns
}

// TestGORMModelsMatchTheMigratedSchema checks every column a GORM model reads
// or writes exists in the schema the migrations build, with a type the field
// can hold. AutoMigrate no longer adds a missing column, so a field added to a
// struct without a migration fails here instead of at query time.
func TestGORMModelsMatchTheMigratedSchema(t *testing.T) {
	db := openTestDatabase(t)
	for _, model := range queryModels {
		parsed, err := schema.Parse(model, &sync.Map{}, db.NamingStrategy)
		if err != nil {
			t.Fatalf("parse %T: %v", model, err)
		}
		columns := migratedColumnTypes(t, db, parsed.Table)
		if len(columns) == 0 {
			t.Errorf("%T: table %s does not exist", model, parsed.Table)
			continue
		}
		for _, field := range parsed.Fields {
			if field.DBName == "" {
				continue
			}
			got, ok := columns[field.DBName]
			if !ok {
				t.Errorf("%s.%s: no such column", parsed.Table, field.DBName)
				continue
			}
			if want := compatibleTypes(field); !slices.Contains(want, got) {
				t.Errorf("%s.%s: column is %s, field %s needs one of %v", parsed.Table, field.DBName, got, field.FieldType, want)
			}
		}
	}
}

func tableShape(t *testing.T, db *gorm.DB, table string) []string {
	t.Helper()
	var shape []string
	err := db.Raw(`SELECT column_name || ' ' || data_type || ' ' || is_nullable || ' ' || coalesce(column_default, '')
		FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ?
		UNION ALL SELECT replace(indexdef, current_schema() || '.', '') FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = ?
		ORDER BY 1`, table, table).Scan(&shape).Error
	if err != nil {
		t.Fatalf("read the shape of %s: %v", table, err)
	}
	return shape
}

// TestMCPPendingSchemaIsInTheMigrations checks the migrations already hold
// tronc/mcp's PendingSchema, which Nuage used to execute at every boot. A
// tronc bump that changes it makes this fail, and the change then needs a
// numbered migration of its own.
func TestMCPPendingSchemaIsInTheMigrations(t *testing.T) {
	db := openTestDatabase(t)
	before := tableShape(t, db, "mcp_actions")
	if err := db.Exec(mcp.PendingSchema).Error; err != nil {
		t.Fatalf("run PendingSchema: %v", err)
	}
	if after := tableShape(t, db, "mcp_actions"); !slices.Equal(before, after) {
		t.Fatalf("PendingSchema changed mcp_actions:\nbefore %v\nafter  %v", before, after)
	}
}

// TestPorteTablesMatchPorteSchema checks the porte tables the migrations build
// have the columns and indexes porte/pg's own Schema gives them, so porte's
// queries find what they expect. Nuage keeps its own copy, with the foreign
// keys on its users table instead of porte_users, and used to run it at every
// boot. A porte bump that changes one of these tables fails here, and the
// change then needs a numbered migration of its own.
func TestPorteTablesMatchPorteSchema(t *testing.T) {
	migrated := openTestDatabase(t)
	reference := openScratchSchema(t)
	if err := reference.Exec(portepg.Schema).Error; err != nil {
		t.Fatalf("run porte's Schema: %v", err)
	}
	for _, table := range porteTables {
		want := tableShape(t, reference, table)
		if got := tableShape(t, migrated, table); !slices.Equal(want, got) {
			t.Errorf("%s differs from porte's Schema:\nporte  %s\nnuage  %s", table, strings.Join(want, "; "), strings.Join(got, "; "))
		}
	}
}
