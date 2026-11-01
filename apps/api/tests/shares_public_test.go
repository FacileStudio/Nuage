package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createShare(t *testing.T, ts *testServer, token string, body map[string]any) string {
	resp := doJSON(ts, "POST", "/shares", body, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var share struct {
		Token string `json:"token"`
	}
	parseJSON(resp, &share)
	return share.Token
}

func uploadForShare(t *testing.T, ts *testServer, token, name, content string, folderID *int64) int64 {
	resp := uploadFile(ts, token, name, content, folderID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)
	return file.ID
}

func TestSharedTrashedFileAnswersNotFound(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "share-trashed@example.com", "password12345")

	fileID := uploadForShare(t, ts, token, "gone.txt", "bytes", nil)
	shared := createShare(t, ts, token, map[string]any{"file_id": fileID, "permission": "view"})
	require.Equal(t, http.StatusOK, doGet(ts, "/shared/"+shared, "").StatusCode)

	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/files/%d", fileID), token).StatusCode)

	assert.Equal(t, http.StatusNotFound, doGet(ts, "/shared/"+shared, "").StatusCode)
	download := doGet(ts, fmt.Sprintf("/shared/%s/download/%d", shared, fileID), "")
	assert.Equal(t, http.StatusNotFound, download.StatusCode)
}

func TestSharedTrashedFolderAnswersNotFound(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "share-trashed-folder@example.com", "password12345")

	folderID := createFolder(t, ts, token, map[string]any{"name": "album"})
	fileID := uploadForShare(t, ts, token, "inside.txt", "bytes", &folderID)
	shared := createShare(t, ts, token, map[string]any{"folder_id": folderID, "permission": "view"})
	require.Equal(t, http.StatusOK, doGet(ts, "/shared/"+shared+"/files", "").StatusCode)

	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/folders/%d", folderID), token).StatusCode)

	assert.Equal(t, http.StatusNotFound, doGet(ts, "/shared/"+shared, "").StatusCode)
	assert.Equal(t, http.StatusNotFound, doGet(ts, "/shared/"+shared+"/files", "").StatusCode)
	download := doGet(ts, fmt.Sprintf("/shared/%s/download/%d", shared, fileID), "")
	assert.Equal(t, http.StatusNotFound, download.StatusCode)
}

func TestSharedFolderRefusesFileOutsideIt(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "share-outside@example.com", "password12345")

	folderID := createFolder(t, ts, token, map[string]any{"name": "public"})
	outside := uploadForShare(t, ts, token, "private.txt", "secret", nil)
	shared := createShare(t, ts, token, map[string]any{"folder_id": folderID, "permission": "view"})

	resp := doGet(ts, fmt.Sprintf("/shared/%s/download/%d", shared, outside), "")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestSharedListingCarriesUpdatedAtAndDownloadRevalidates(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "share-listing@example.com", "password12345")

	folderID := createFolder(t, ts, token, map[string]any{"name": "pics"})
	fileID := uploadForShare(t, ts, token, "a.txt", "one", &folderID)
	shared := createShare(t, ts, token, map[string]any{"folder_id": folderID, "permission": "view"})

	resp := doGet(ts, "/shared/"+shared+"/files", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listing struct {
		Files []struct {
			ID        int64  `json:"id"`
			UpdatedAt string `json:"updated_at"`
		} `json:"files"`
	}
	parseJSON(resp, &listing)
	require.Len(t, listing.Files, 1)
	assert.Equal(t, fileID, listing.Files[0].ID)
	assert.NotEmpty(t, listing.Files[0].UpdatedAt)

	dl := doGet(ts, fmt.Sprintf("/shared/%s/download/%d", shared, fileID), "")
	require.Equal(t, http.StatusOK, dl.StatusCode)
	assert.Equal(t, "private, no-cache", dl.Header.Get("Cache-Control"))
}
