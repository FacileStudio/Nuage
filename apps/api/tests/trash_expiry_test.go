package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/modules/trash"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTrashService(t *testing.T, ts *testServer) *trash.Service {
	t.Helper()
	return trash.NewService(ts.db, testStorageClient(t), activity.NewLogger(ts.db), quota.NewService(ts.db))
}

func backdateTrashed(t *testing.T, ts *testServer, model any, id int64, age time.Duration) {
	t.Helper()
	require.NoError(t, ts.db.Model(model).Where("id = ?", id).Update("deleted_at", time.Now().Add(-age)).Error)
}

func trashFile(t *testing.T, ts *testServer, token string, id int64) {
	t.Helper()
	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/files/%d", id), token).StatusCode)
}

func fileExists(ts *testServer, id int64) bool {
	var count int64
	ts.db.Model(&schemas.File{}).Where("id = ?", id).Count(&count)
	return count == 1
}

func TestPurgeExpiredRemovesOnlyOldTrash(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "expire@example.com", "password12345")
	old := uploadTestFile(t, ts, token, "old.txt", "old content here", nil)
	young := uploadTestFile(t, ts, token, "young.txt", "young content", nil)
	var oldKey string
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("id = ?", old).Pluck("bucket_key", &oldKey).Error)
	before := storageUsed(t, ts, token)
	trashFile(t, ts, token, old)
	trashFile(t, ts, token, young)
	backdateTrashed(t, ts, &schemas.File{}, old, 31*24*time.Hour)
	backdateTrashed(t, ts, &schemas.File{}, young, 29*24*time.Hour)

	purged, err := newTrashService(t, ts).PurgeExpired(context.Background(), time.Now())

	require.NoError(t, err)
	assert.Equal(t, 1, purged)
	assert.False(t, fileExists(ts, old))
	assert.True(t, fileExists(ts, young))
	_, statErr := testStorageClient(t).StatObject(context.Background(), oldKey)
	assert.Error(t, statErr, "the purged object should be gone from storage")
	assert.Equal(t, before-int64(len("old content here")), storageUsed(t, ts, token))
	var tombstones int64
	require.NoError(t, ts.db.Model(&schemas.Tombstone{}).Where("resource_id = ?", old).Count(&tombstones).Error)
	assert.Equal(t, int64(1), tombstones)
}

func TestPurgeExpiredRemovesTrashedFolderWithItsFiles(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "expirefolder@example.com", "password12345")
	root := createTestFolder(t, ts, token, "Projects", nil)
	child := createTestFolder(t, ts, token, "Sub", &root)
	fileID := uploadTestFile(t, ts, token, "a.txt", "nested file content", &child)
	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/folders/%d", root), token).StatusCode)
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("id = ?", fileID).Update("deleted_at", time.Now().Add(-40*24*time.Hour)).Error)
	require.NoError(t, ts.db.Model(&schemas.Folder{}).Where("id IN ?", []int64{root, child}).Update("deleted_at", time.Now().Add(-40*24*time.Hour)).Error)

	purged, err := newTrashService(t, ts).PurgeExpired(context.Background(), time.Now())

	require.NoError(t, err)
	assert.Equal(t, 3, purged)
	var folders int64
	require.NoError(t, ts.db.Model(&schemas.Folder{}).Count(&folders).Error)
	assert.Equal(t, int64(0), folders)
	assert.False(t, fileExists(ts, fileID))
	assert.Equal(t, int64(0), storageUsed(t, ts, token))
}

func TestPurgeExpiredRefundsTheUploaderNotTheMemberWhoTrashedIt(t *testing.T) {
	ts := setupSettledServer(t)
	_, owner := registerUser(ts, "expire-owner@example.com", "password12345")
	memberID, member := registerUser(ts, "expire-member@example.com", "password12345")
	spaceID := createSpace(t, ts, owner, "Expiring Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)
	resp := uploadFileToSpace(ts, owner, "shared.txt", "shared bytes", spaceID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)
	require.Greater(t, storageUsed(t, ts, owner), int64(0))
	trashFile(t, ts, member, file.ID)
	backdateTrashed(t, ts, &schemas.File{}, file.ID, 45*24*time.Hour)

	purged, err := newTrashService(t, ts).PurgeExpired(context.Background(), time.Now())

	require.NoError(t, err)
	assert.Equal(t, 1, purged)
	assert.Equal(t, int64(0), storageUsed(t, ts, owner))
	assert.Equal(t, int64(0), storageUsed(t, ts, member))
}

func TestPurgeExpiredIsDisabledByZeroRetention(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "expirezero@example.com", "password12345")
	id := uploadTestFile(t, ts, token, "keep.txt", "keep me forever", nil)
	trashFile(t, ts, token, id)
	backdateTrashed(t, ts, &schemas.File{}, id, 400*24*time.Hour)
	require.NoError(t, ts.db.Create(&schemas.Setting{Key: schemas.TrashRetentionKey, Value: "0"}).Error)

	purged, err := newTrashService(t, ts).PurgeExpired(context.Background(), time.Now())

	require.NoError(t, err)
	assert.Equal(t, 0, purged)
	assert.True(t, fileExists(ts, id))
}

func TestTrashListReportsRetentionAndExpiry(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "expirelist@example.com", "password12345")
	id := uploadTestFile(t, ts, token, "listed.txt", "listed", nil)
	trashFile(t, ts, token, id)

	resp := doGet(ts, "/trash", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list struct {
		RetentionDays int `json:"retention_days"`
		Items         []struct {
			DeletedAt string `json:"deleted_at"`
			ExpiresAt string `json:"expires_at"`
		} `json:"items"`
	}
	parseJSON(resp, &list)

	require.Len(t, list.Items, 1)
	assert.Equal(t, 30, list.RetentionDays)
	deleted, err := time.Parse(time.RFC3339, list.Items[0].DeletedAt)
	require.NoError(t, err)
	expires, err := time.Parse(time.RFC3339, list.Items[0].ExpiresAt)
	require.NoError(t, err)
	assert.Equal(t, deleted.AddDate(0, 0, 30), expires)
}

func TestRetentionSettingIsAdminOnlyAndValidated(t *testing.T) {
	ts := setupSettledServer(t)
	_, admin := registerUser(ts, "retention-admin@example.com", "password12345")
	_, other := registerUser(ts, "retention-other@example.com", "password12345")
	put := func(token, value string) int {
		body := map[string]any{"settings": map[string]string{schemas.TrashRetentionKey: value}}
		return doJSON(ts, "PUT", "/settings", body, token).StatusCode
	}

	assert.Equal(t, http.StatusForbidden, put(other, "7"))
	assert.Equal(t, http.StatusBadRequest, put(admin, "-1"))
	assert.Equal(t, http.StatusBadRequest, put(admin, "soon"))
	assert.Equal(t, http.StatusBadRequest, put(admin, "99999"))
	assert.Equal(t, http.StatusOK, put(admin, "7"))
}
