package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSettledServer(t *testing.T) *testServer {
	ts := setupTestServer(t)
	t.Cleanup(func() {
		prev := int64(-1)
		for i := 0; i < 50; i++ {
			var count int64
			if err := ts.db.Table("activity_logs").Count(&count).Error; err != nil {
				return
			}
			if count == prev {
				return
			}
			prev = count
			time.Sleep(20 * time.Millisecond)
		}
	})
	return ts
}

func createTestFolder(t *testing.T, ts *testServer, token, name string, parentID *int64) int64 {
	t.Helper()
	body := map[string]any{"name": name}
	if parentID != nil {
		body["parent_id"] = *parentID
	}
	resp := doJSON(ts, "POST", "/folders", body, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)
	return folder.ID
}

func uploadTestFile(t *testing.T, ts *testServer, token, name, content string, folderID *int64) int64 {
	t.Helper()
	resp := uploadFile(ts, token, name, content, folderID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)
	return file.ID
}

func storageUsed(t *testing.T, ts *testServer, token string) int64 {
	t.Helper()
	resp := doGet(ts, "/quota/me", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(resp, &usage)
	return usage.StorageUsed
}

func testStorageClient(t *testing.T) *storage.Client {
	t.Helper()
	endpoint := os.Getenv("TEST_MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	accessKey := os.Getenv("TEST_MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "nuage-minio"
	}
	secretKey := os.Getenv("TEST_MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "nuage-internal-storage"
	}
	client, err := storage.NewClient(storage.MinIOConfig{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    "nuage-test",
		UseSSL:    false,
	})
	require.NoError(t, err)
	return client
}

func TestTrashAndRestore(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "trash@example.com", "password12345")

	resp := uploadFile(ts, token, "trashme.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token)

	trashResp := doGet(ts, "/trash", token)
	require.Equal(t, http.StatusOK, trashResp.StatusCode)

	var trashList struct {
		Items []struct {
			Type string `json:"type"`
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	parseJSON(trashResp, &trashList)
	assert.Len(t, trashList.Items, 1)
	assert.Equal(t, "file", trashList.Items[0].Type)

	restoreResp := doJSON(ts, "POST", fmt.Sprintf("/trash/file/%d/restore", file.ID), nil, token)
	assert.Equal(t, http.StatusOK, restoreResp.StatusCode)

	getResp := doGet(ts, fmt.Sprintf("/files/%d", file.ID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
}

func TestPermanentDelete(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "permdelete@example.com", "password12345")

	resp := uploadFile(ts, token, "goodbye.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token)

	delResp := doDelete(ts, fmt.Sprintf("/trash/file/%d", file.ID), token)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)

	trashResp := doGet(ts, "/trash", token)
	var trashList struct {
		Items []struct{} `json:"items"`
	}
	parseJSON(trashResp, &trashList)
	assert.Empty(t, trashList.Items)
}

func TestPermanentDeleteFolderPurgesDescendants(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "permfolder@example.com", "password12345")

	root := createTestFolder(t, ts, token, "Projects", nil)
	child := createTestFolder(t, ts, token, "Sub", &root)
	uploadTestFile(t, ts, token, "a.txt", "root file content", &root)
	uploadTestFile(t, ts, token, "b.txt", "nested file content", &child)

	var keys []string
	require.NoError(t, ts.db.Model(&schemas.File{}).Pluck("bucket_key", &keys).Error)
	require.Len(t, keys, 2)
	require.Greater(t, storageUsed(t, ts, token), int64(0))

	delResp := doDelete(ts, fmt.Sprintf("/folders/%d", root), token)
	require.Equal(t, http.StatusOK, delResp.StatusCode)

	permResp := doDelete(ts, fmt.Sprintf("/trash/folder/%d", root), token)
	require.Equal(t, http.StatusOK, permResp.StatusCode)

	var fileCount, folderCount, versionCount int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Count(&fileCount).Error)
	require.NoError(t, ts.db.Model(&schemas.Folder{}).Count(&folderCount).Error)
	require.NoError(t, ts.db.Model(&schemas.FileVersion{}).Count(&versionCount).Error)
	assert.Equal(t, int64(0), fileCount)
	assert.Equal(t, int64(0), folderCount)
	assert.Equal(t, int64(0), versionCount)

	assert.Equal(t, int64(0), storageUsed(t, ts, token))

	client := testStorageClient(t)
	for _, key := range keys {
		_, err := client.StatObject(context.Background(), key)
		assert.Error(t, err, "object %s should be deleted from storage", key)
	}
}

func TestTrashRestoreFolderRestoresDescendants(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "restorefolder@example.com", "password12345")

	root := createTestFolder(t, ts, token, "Projects", nil)
	child := createTestFolder(t, ts, token, "Sub", &root)
	fileA := uploadTestFile(t, ts, token, "a.txt", "root file", &root)
	fileB := uploadTestFile(t, ts, token, "b.txt", "nested file", &child)

	delResp := doDelete(ts, fmt.Sprintf("/folders/%d", root), token)
	require.Equal(t, http.StatusOK, delResp.StatusCode)

	restoreResp := doJSON(ts, "POST", fmt.Sprintf("/trash/folder/%d/restore", root), nil, token)
	require.Equal(t, http.StatusOK, restoreResp.StatusCode)

	assert.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/files/%d", fileA), token).StatusCode)
	assert.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/files/%d", fileB), token).StatusCode)
	assert.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/folders/%d", child), token).StatusCode)

	trashResp := doGet(ts, "/trash", token)
	var trashList struct {
		Items []struct{} `json:"items"`
	}
	parseJSON(trashResp, &trashList)
	assert.Empty(t, trashList.Items)
}

func TestTrashRestoreFileDetachesFromTrashedFolder(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "restoredetach@example.com", "password12345")

	root := createTestFolder(t, ts, token, "Projects", nil)
	fileID := uploadTestFile(t, ts, token, "orphan.txt", "file content", &root)

	delResp := doDelete(ts, fmt.Sprintf("/folders/%d", root), token)
	require.Equal(t, http.StatusOK, delResp.StatusCode)

	restoreResp := doJSON(ts, "POST", fmt.Sprintf("/trash/file/%d/restore", fileID), nil, token)
	require.Equal(t, http.StatusOK, restoreResp.StatusCode)

	getResp := doGet(ts, fmt.Sprintf("/files/%d", fileID), token)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
	var file struct {
		FolderID *int64 `json:"folder_id"`
	}
	parseJSON(getResp, &file)
	assert.Nil(t, file.FolderID)

	trashResp := doGet(ts, "/trash", token)
	var trashList struct {
		Items []struct {
			Type string `json:"type"`
			ID   int64  `json:"id"`
		} `json:"items"`
	}
	parseJSON(trashResp, &trashList)
	require.Len(t, trashList.Items, 1)
	assert.Equal(t, "folder", trashList.Items[0].Type)
	assert.Equal(t, root, trashList.Items[0].ID)
}

func TestTrashRestoreCycleDoesNotInflateQuota(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "quotacycle@example.com", "password12345")

	fileID := uploadTestFile(t, ts, token, "cycle.txt", "cycle file content", nil)
	used := storageUsed(t, ts, token)
	require.Greater(t, used, int64(0))

	for i := 0; i < 3; i++ {
		delResp := doDelete(ts, fmt.Sprintf("/files/%d", fileID), token)
		require.Equal(t, http.StatusOK, delResp.StatusCode)
		assert.Equal(t, used, storageUsed(t, ts, token))
		restoreResp := doJSON(ts, "POST", fmt.Sprintf("/trash/file/%d/restore", fileID), nil, token)
		require.Equal(t, http.StatusOK, restoreResp.StatusCode)
		assert.Equal(t, used, storageUsed(t, ts, token))
	}
}

func TestPermanentDeleteRefundsQuota(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "permrefund@example.com", "password12345")

	fileID := uploadTestFile(t, ts, token, "refund.txt", "refund file content", nil)
	used := storageUsed(t, ts, token)
	require.Greater(t, used, int64(0))

	doDelete(ts, fmt.Sprintf("/files/%d", fileID), token)
	assert.Equal(t, used, storageUsed(t, ts, token))

	delResp := doDelete(ts, fmt.Sprintf("/trash/file/%d", fileID), token)
	require.Equal(t, http.StatusOK, delResp.StatusCode)
	assert.Equal(t, int64(0), storageUsed(t, ts, token))
}

func TestEmptyTrashPurgesFilesAndFolders(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "emptytrash@example.com", "password12345")

	root := createTestFolder(t, ts, token, "Projects", nil)
	child := createTestFolder(t, ts, token, "Sub", &root)
	uploadTestFile(t, ts, token, "a.txt", "root file", &root)
	uploadTestFile(t, ts, token, "b.txt", "nested file", &child)
	standalone := uploadTestFile(t, ts, token, "loose.txt", "loose file", nil)

	doDelete(ts, fmt.Sprintf("/folders/%d", root), token)
	doDelete(ts, fmt.Sprintf("/files/%d", standalone), token)

	emptyResp := doDelete(ts, "/trash", token)
	require.Equal(t, http.StatusOK, emptyResp.StatusCode)
	var result struct {
		Deleted int64 `json:"deleted"`
	}
	parseJSON(emptyResp, &result)
	assert.Equal(t, int64(5), result.Deleted)

	var fileCount, folderCount int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Count(&fileCount).Error)
	require.NoError(t, ts.db.Model(&schemas.Folder{}).Count(&folderCount).Error)
	assert.Equal(t, int64(0), fileCount)
	assert.Equal(t, int64(0), folderCount)

	assert.Equal(t, int64(0), storageUsed(t, ts, token))
}
