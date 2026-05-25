package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileVersioning(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "version@example.com", "password123")

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
	_, token := registerUser(ts, "restore-ver@example.com", "password123")

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
		Versions []struct{ ID int64 `json:"id"` } `json:"versions"`
	}
	parseJSON(versionsResp, &versionList)
	require.Len(t, versionList.Versions, 1)

	restoreResp := doJSON(ts, "POST",
		fmt.Sprintf("/files/%d/versions/%d/restore", file.ID, versionList.Versions[0].ID),
		nil, token)
	require.Equal(t, http.StatusOK, restoreResp.StatusCode)

	var restored struct{ Hash string `json:"hash"` }
	parseJSON(restoreResp, &restored)
	assert.Equal(t, originalHash, restored.Hash)
}
