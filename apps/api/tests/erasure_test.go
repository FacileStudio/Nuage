package tests

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/erasure"
	"github.com/FacileStudio/Nuage/apps/api/modules/trash"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"
	portepg "github.com/FacileStudio/porte/pg"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type erasureFixture struct {
	ts        *testServer
	gone      int64
	other     int64
	space     int64
	deletions *erasure.Deletions
}

func count(t *testing.T, ts *testServer, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, ts.db.Raw(query, args...).Scan(&n).Error)
	return n
}

func seedErasure(t *testing.T) erasureFixture {
	t.Helper()
	ts := setupTestServer(t)
	_, adminToken := registerUser(ts, "admin@example.com", "")
	goneID, goneToken := registerUser(ts, "gone@example.com", "")
	otherID, otherToken := registerUser(ts, "other@example.com", "")
	gone, _ := strconv.ParseInt(goneID, 10, 64)
	other, _ := strconv.ParseInt(otherID, 10, 64)

	link(t, ts, gone, "sub-gone@example.com")
	space := createSpace(t, ts, adminToken, "Studio")
	addSpaceMember(t, ts, adminToken, space, goneID)
	require.Equal(t, 201, uploadFileToSpace(ts, goneToken, "team.txt", "shared", space).StatusCode)
	require.Equal(t, 201, uploadFile(ts, goneToken, "private.txt", "secret", nil).StatusCode)
	require.Equal(t, 201, uploadFile(ts, otherToken, "theirs.txt", "mine", nil).StatusCode)
	require.NoError(t, ts.db.Create(&schemas.Folder{FacileID: "fac_gone_folder", Name: "Perso", OwnerID: gone}).Error)

	app := schemas.RegisteredApp{Key: "plume", Name: "Plume", ClientID: "plume-client", Active: true}
	require.NoError(t, ts.db.Create(&app).Error)
	folder := schemas.AppFolder{UserID: gone, AppID: app.ID}
	require.NoError(t, ts.db.Create(&folder).Error)
	require.NoError(t, ts.db.Create(&schemas.AppObject{FolderID: folder.ID, Key: "doc", Name: "doc", BucketKey: "apps/x/doc"}).Error)

	sqlDB, err := ts.db.DB()
	require.NoError(t, err)
	trashService := trash.NewService(ts.db, ts.store, nil, nil)
	deletions := erasure.New(erasure.Config{
		ORM: ts.db, Client: directory.Client{Issuer: testIssuer}, Erase: portepg.New(sqlDB).EraseIdentity,
		Files: trashService, Apps: ts.appFolders, StorageDir: t.TempDir(),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	return erasureFixture{ts: ts, gone: gone, other: other, space: space, deletions: deletions}
}

func TestErasePersonDeletesPersonalDataAndKeepsSpaceContent(t *testing.T) {
	f := seedErasure(t)
	for range 2 {
		require.NoError(t, f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone@example.com"))
	}

	checks := []struct {
		query string
		arg   int64
		want  int64
	}{
		{`SELECT count(*) FROM users WHERE id = ?`, f.gone, 0},
		{`SELECT count(*) FROM files WHERE name = 'private.txt'`, 0, 0},
		{`SELECT count(*) FROM folders WHERE name = 'Perso'`, 0, 0},
		{`SELECT count(*) FROM porte_sessions WHERE user_id = ?`, f.gone, 0},
		{`SELECT count(*) FROM porte_identities WHERE user_id = ?`, f.gone, 0},
		{`SELECT count(*) FROM space_members WHERE user_id = ?`, f.gone, 0},
		{`SELECT count(*) FROM app_folders WHERE user_id = ?`, f.gone, 0},
		{`SELECT count(*) FROM app_objects WHERE key = 'doc'`, 0, 0},
		{`SELECT count(*) FROM files WHERE name = 'team.txt'`, 0, 1},
		{`SELECT count(*) FROM files WHERE name = 'theirs.txt'`, 0, 1},
		{`SELECT count(*) FROM users WHERE id = ?`, f.other, 1},
	}
	for _, check := range checks {
		args := []any{}
		if check.arg != 0 {
			args = append(args, check.arg)
		}
		assert.Equal(t, check.want, count(t, f.ts, check.query, args...), check.query)
	}
}

func TestErasePersonAnonymisesSpaceAuthorship(t *testing.T) {
	f := seedErasure(t)
	require.NoError(t, f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone@example.com"))

	var ghost schemas.User
	require.NoError(t, f.ts.db.Where("email = ?", schemas.DeletedUserEmail).First(&ghost).Error)
	var file schemas.File
	require.NoError(t, f.ts.db.Where("name = ?", "team.txt").First(&file).Error)
	assert.Equal(t, ghost.ID, file.UploadedBy)

	_, adminToken := registerUser(f.ts, "admin@example.com", "")
	var list struct {
		Users []struct{ Email string } `json:"users"`
	}
	parseJSON(doGet(f.ts, "/users", adminToken), &list)
	for _, user := range list.Users {
		assert.NotEqual(t, schemas.DeletedUserEmail, user.Email)
	}
}

func TestEraseUnknownSubjectIsANoOp(t *testing.T) {
	f := seedErasure(t)
	require.NoError(t, f.deletions.ErasePerson(context.Background(), testIssuer, "sub-nobody"))
	assert.EqualValues(t, 3, count(t, f.ts, `SELECT count(*) FROM users`))
}

func TestDeletionStateSurvivesARestart(t *testing.T) {
	ts := setupTestServer(t)
	state := erasure.NewState(ts.db)
	ctx := context.Background()

	saved, err := state.Load(ctx)
	require.NoError(t, err)
	assert.Nil(t, saved)
	for _, blob := range []string{"one", "two"} {
		require.NoError(t, state.Save(ctx, []byte(blob)))
	}
	saved, err = state.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, "two", string(saved))
}

func TestDeletionHandlerWaitsForTheAdvisoryLock(t *testing.T) {
	f := seedErasure(t)
	sqlDB, err := f.ts.db.DB()
	require.NoError(t, err)
	conn, err := sqlDB.Conn(context.Background())
	require.NoError(t, err)
	defer conn.Close()
	_, err = conn.ExecContext(context.Background(), "SELECT pg_advisory_lock(4920)")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	request := httptest.NewRequest(http.MethodPost, directory.DeletionsPath, nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	f.deletions.Handler().ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)

	_, err = conn.ExecContext(context.Background(), "SELECT pg_advisory_unlock(4920)")
	require.NoError(t, err)
	recorder = httptest.NewRecorder()
	f.deletions.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, directory.DeletionsPath, nil))
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestConcurrentErasuresOfOneSubjectLeaveOneDeletedUser(t *testing.T) {
	f := seedErasure(t)
	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			_ = f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone@example.com")
		}()
	}
	group.Wait()
	require.NoError(t, f.deletions.ErasePerson(context.Background(), testIssuer, "sub-gone@example.com"))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM users WHERE email = ?`, schemas.DeletedUserEmail))
	assert.EqualValues(t, 0, count(t, f.ts, `SELECT count(*) FROM users WHERE id = ?`, f.gone))
}
