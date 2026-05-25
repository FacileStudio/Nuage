package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadAndDownload(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "files@example.com", "password123")

	content := "hello world file content"
	resp := uploadFile(ts, token, "test.txt", content, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Size     int64  `json:"size"`
		Hash     string `json:"hash"`
		MimeType string `json:"mime_type"`
	}
	parseJSON(resp, &file)
	assert.Equal(t, "test.txt", file.Name)
	assert.Equal(t, int64(len(content)), file.Size)

	expectedHash := sha256.Sum256([]byte(content))
	assert.Equal(t, hex.EncodeToString(expectedHash[:]), file.Hash)

	dlResp := doGet(ts, fmt.Sprintf("/files/%d/download", file.ID), token)
	require.Equal(t, http.StatusOK, dlResp.StatusCode)

	body, _ := io.ReadAll(dlResp.Body)
	dlResp.Body.Close()
	assert.Equal(t, content, string(body))
}

func TestUploadToFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "folder-upload@example.com", "password123")

	folderResp := doJSON(ts, "POST", "/folders", map[string]string{"name": "docs"}, token)
	require.Equal(t, http.StatusCreated, folderResp.StatusCode)

	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(folderResp, &folder)

	resp := uploadFile(ts, token, "doc.pdf", "pdf content", &folder.ID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		FolderID *int64 `json:"folder_id"`
	}
	parseJSON(resp, &file)
	assert.NotNil(t, file.FolderID)
	assert.Equal(t, folder.ID, *file.FolderID)
}

func TestDeduplicateFileName(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dedup@example.com", "password123")

	uploadFile(ts, token, "report.txt", "first", nil)
	resp := uploadFile(ts, token, "report.txt", "second", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		Name string `json:"name"`
	}
	parseJSON(resp, &file)
	assert.Equal(t, "report (1).txt", file.Name)
}

func TestListFiles(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "list@example.com", "password123")

	uploadFile(ts, token, "a.txt", "a", nil)
	uploadFile(ts, token, "b.txt", "b", nil)

	resp := doGet(ts, "/files", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Files []struct{ Name string } `json:"files"`
	}
	parseJSON(resp, &result)
	assert.Len(t, result.Files, 2)
}

func TestUpdateFile(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "update@example.com", "password123")

	resp := uploadFile(ts, token, "old.txt", "content", nil)
	var file struct{ ID int64 `json:"id"` }
	parseJSON(resp, &file)

	updateResp := doJSON(ts, "PUT", fmt.Sprintf("/files/%d", file.ID), map[string]string{"name": "new.txt"}, token)
	require.Equal(t, http.StatusOK, updateResp.StatusCode)

	var updated struct{ Name string `json:"name"` }
	parseJSON(updateResp, &updated)
	assert.Equal(t, "new.txt", updated.Name)
}

func TestDeleteFile(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "delete@example.com", "password123")

	resp := uploadFile(ts, token, "deleteme.txt", "bye", nil)
	var file struct{ ID int64 `json:"id"` }
	parseJSON(resp, &file)

	delResp := doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token)
	require.Equal(t, http.StatusOK, delResp.StatusCode)

	getResp := doGet(ts, fmt.Sprintf("/files/%d", file.ID), token)
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestSearchFiles(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search@example.com", "password123")

	uploadFile(ts, token, "quarterly-report.pdf", "data", nil)
	uploadFile(ts, token, "invoice.pdf", "data", nil)

	resp := doGet(ts, "/files?search=quarterly", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Files []struct{ Name string } `json:"files"`
	}
	parseJSON(resp, &result)
	assert.Len(t, result.Files, 1)
	assert.Equal(t, "quarterly-report.pdf", result.Files[0].Name)
}
