package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileVersioning(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "version@example.com", "password12345")

	resp := uploadFile(ts, token, "versioned.txt", "version 1", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID   int64  `json:"id"`
		Hash string `json:"hash"`
		Size int64  `json:"size"`
	}
	parseJSON(resp, &file)
	originalHash := file.Hash

	reupResp := reuploadFile(ts, token, file.ID, "version 2 content")
	require.Equal(t, http.StatusOK, reupResp.StatusCode)

	var updated struct {
		Hash string `json:"hash"`
		Size int64  `json:"size"`
	}
	parseJSON(reupResp, &updated)
	assert.NotEqual(t, originalHash, updated.Hash)

	versionsResp := doGet(ts, fmt.Sprintf("/files/%d/versions", file.ID), token)
	require.Equal(t, http.StatusOK, versionsResp.StatusCode)

	var versionList struct {
		Versions []struct {
			ID      int64 `json:"id"`
			Version int   `json:"version"`
			Size    int64 `json:"size"`
		} `json:"versions"`
	}
	parseJSON(versionsResp, &versionList)
	assert.Len(t, versionList.Versions, 1)
	assert.Equal(t, 1, versionList.Versions[0].Version)
}

func TestRestoreVersion(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "restore-ver@example.com", "password12345")

	resp := uploadFile(ts, token, "restore-ver.txt", "original", nil)
	var file struct {
		ID   int64  `json:"id"`
		Hash string `json:"hash"`
	}
	parseJSON(resp, &file)
	originalHash := file.Hash

	reuploadFile(ts, token, file.ID, "modified content")

	versionsResp := doGet(ts, fmt.Sprintf("/files/%d/versions", file.ID), token)
	var versionList struct {
		Versions []struct {
			ID int64 `json:"id"`
		} `json:"versions"`
	}
	parseJSON(versionsResp, &versionList)
	require.Len(t, versionList.Versions, 1)

	restoreResp := doJSON(ts, "POST",
		fmt.Sprintf("/files/%d/versions/%d/restore", file.ID, versionList.Versions[0].ID),
		nil, token)
	require.Equal(t, http.StatusOK, restoreResp.StatusCode)

	var restored struct {
		Hash string `json:"hash"`
	}
	parseJSON(restoreResp, &restored)
	assert.Equal(t, originalHash, restored.Hash)
}

func TestVersionCleanupKeepsLiveObjects(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "version-clean@example.com", "password12345")

	require.NoError(t, ts.db.Create(&schemas.Setting{Key: "max_file_versions", Value: "1"}).Error)

	resp := uploadFile(ts, token, "clean.txt", "v1", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	for i := 2; i <= 4; i++ {
		r := reuploadFile(ts, token, file.ID, fmt.Sprintf("version %d content", i))
		require.Equal(t, http.StatusOK, r.StatusCode)
	}

	var versions []schemas.FileVersion
	require.Eventually(t, func() bool {
		versions = nil
		ts.db.Where("file_id = ?", file.ID).Find(&versions)
		return len(versions) == 1
	}, 5*time.Second, 100*time.Millisecond)

	storageClient := newTestStorageClient(t)
	for _, v := range versions {
		_, err := storageClient.StatObject(context.Background(), v.BucketKey)
		assert.NoError(t, err, "version row must point at an existing object")
	}

	var record schemas.File
	require.NoError(t, ts.db.Where("id = ?", file.ID).First(&record).Error)
	_, err := storageClient.StatObject(context.Background(), record.BucketKey)
	assert.NoError(t, err, "live file object must exist")

	dlResp := doGet(ts, fmt.Sprintf("/files/%d/download", file.ID), token)
	assert.Equal(t, http.StatusOK, dlResp.StatusCode)
}
