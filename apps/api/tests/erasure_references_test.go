package tests

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func userReferenceColumns(t *testing.T, ts *testServer) [][2]string {
	t.Helper()
	var rows []struct{ TableName, ColumnName string }
	require.NoError(t, ts.db.Raw(`SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = current_schema() AND data_type IN ('bigint', 'integer')
		  AND column_name IN ('user_id', 'uploaded_by', 'created_by', 'owner_id', 'shared_by', 'charged_to', 'billing_offer_to')`).
		Scan(&rows).Error)
	columns := make([][2]string, 0, len(rows)+1)
	for _, row := range rows {
		columns = append(columns, [2]string{row.TableName, row.ColumnName})
	}
	return append(columns, [2]string{"users", "id"})
}

func textColumns(t *testing.T, ts *testServer) [][2]string {
	t.Helper()
	var rows []struct{ TableName, ColumnName string }
	require.NoError(t, ts.db.Raw(`SELECT c.table_name, c.column_name FROM information_schema.columns c
		JOIN information_schema.tables t ON t.table_schema = c.table_schema AND t.table_name = c.table_name
		WHERE c.table_schema = current_schema() AND t.table_type = 'BASE TABLE'
		  AND c.data_type IN ('text', 'character varying', 'jsonb', 'json')`).Scan(&rows).Error)
	columns := make([][2]string, 0, len(rows))
	for _, row := range rows {
		columns = append(columns, [2]string{row.TableName, row.ColumnName})
	}
	return columns
}

func seedEveryReference(t *testing.T, f erasureFixture) {
	t.Helper()
	db := f.ts.db
	gone := f.gone
	var private schemas.File
	require.NoError(t, db.Where("name = ?", "private.txt").First(&private).Error)
	require.NoError(t, db.Create(&schemas.Share{Token: "tok-gone", FileID: &private.ID, SharedBy: gone}).Error)
	require.NoError(t, db.Create(&schemas.ActivityLog{UserID: gone, EventType: "file.upload", ResourceType: "file",
		ResourceID: private.ID, ResourceName: "private.txt"}).Error)
	require.NoError(t, db.Create(&schemas.UploadSession{ID: "sess-gone", FileName: "half.bin", UserID: gone,
		TotalSize: 10, ExpiresAt: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.Exec(`UPDATE spaces SET charged_to = ?, billing_offer_to = ? WHERE id = ?`, gone, gone, f.space).Error)
	require.NoError(t, db.Exec(`INSERT INTO mcp_actions (id, user_id, kind, target, digest, expires_at)
		VALUES ('act-gone', ?, 'delete', '{}', 'd', now() + interval '1 hour')`, strconv.FormatInt(gone, 10)).Error)
	require.NoError(t, db.Exec(`INSERT INTO nook_deliveries (event_type, payload) VALUES ('user.created', ?)`,
		`{"user":{"email":"gone@example.com"}}`).Error)
}

func TestEraseLeavesNoRowThatReferencesThePerson(t *testing.T) {
	f := seedErasure(t)
	seedEveryReference(t, f)
	require.NoError(t, f.deletions.ErasePerson(context.Background(), testIssuer+"/", "sub-gone@example.com"))

	for _, column := range userReferenceColumns(t, f.ts) {
		query := `SELECT count(*) FROM "` + column[0] + `" WHERE "` + column[1] + `" = ?`
		assert.Zero(t, count(t, f.ts, query, f.gone), column[0]+"."+column[1])
	}
	for _, column := range textColumns(t, f.ts) {
		query := `SELECT count(*) FROM "` + column[0] + `" WHERE "` + column[1] + `"::text LIKE '%gone@example.com%'`
		assert.Zero(t, count(t, f.ts, query), column[0]+"."+column[1])
	}
	assert.Zero(t, count(t, f.ts, `SELECT count(*) FROM mcp_actions WHERE user_id = ?`, strconv.FormatInt(f.gone, 10)))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM spaces WHERE id = ?`, f.space))
}

func TestEraseHandsASoleOwnersSpaceToItsLongestStandingMember(t *testing.T) {
	f := seedErasure(t)
	_, goneToken := registerUser(f.ts, "gone@example.com", "")
	owned := createSpace(t, f.ts, goneToken, "Gone's team")
	alone := createSpace(t, f.ts, goneToken, "Gone alone")
	require.Equal(t, 201, uploadFileToSpace(f.ts, goneToken, "kept.txt", "kept", alone).StatusCode)
	addSpaceMember(t, f.ts, goneToken, owned, strconv.FormatInt(f.other, 10))

	require.NoError(t, f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone@example.com"))

	var role string
	require.NoError(t, f.ts.db.Raw(`SELECT role FROM space_members WHERE space_id = ? AND user_id = ?`, owned, f.other).
		Scan(&role).Error)
	assert.Equal(t, "owner", role)
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM spaces WHERE id = ?`, alone))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM files WHERE name = 'kept.txt'`))
}

func TestAFailedEraseRollsBackEveryRow(t *testing.T) {
	f := seedErasure(t)
	require.NoError(t, f.ts.db.Exec(`CREATE FUNCTION refuse() RETURNS trigger LANGUAGE plpgsql AS
		$$ BEGIN RAISE EXCEPTION 'refused'; END $$`).Error)
	require.NoError(t, f.ts.db.Exec(`CREATE TRIGGER refuse BEFORE DELETE ON user_quotas FOR EACH ROW EXECUTE FUNCTION refuse()`).Error)
	require.NoError(t, f.ts.db.Exec(`INSERT INTO user_quotas (user_id, storage_used, storage_limit) VALUES (?, 0, 0)
		ON CONFLICT (user_id) DO NOTHING`, f.gone).Error)

	require.Error(t, f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone@example.com"))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM users WHERE id = ?`, f.gone))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM files WHERE name = 'private.txt'`))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM app_objects WHERE key = 'doc'`))
}
