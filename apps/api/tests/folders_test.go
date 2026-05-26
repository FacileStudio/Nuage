package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "folders@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "documents"}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var folder struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FacileID string `json:"facile_id"`
	}
	parseJSON(resp, &folder)
	assert.Equal(t, "documents", folder.Name)
	assert.NotEmpty(t, folder.FacileID)
}

func TestNestedFolders(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "nested@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "parent"}, token)
	var parent struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &parent)

	resp = doJSON(ts, "POST", "/folders", map[string]any{"name": "child", "parent_id": parent.ID}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var child struct {
		ParentID *int64 `json:"parent_id"`
	}
	parseJSON(resp, &child)
	assert.NotNil(t, child.ParentID)
	assert.Equal(t, parent.ID, *child.ParentID)
}

func TestGetFolderContents(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "contents@example.com", "password12345")

	folderResp := doJSON(ts, "POST", "/folders", map[string]string{"name": "stuff"}, token)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(folderResp, &folder)

	uploadFile(ts, token, "inside.txt", "content", &folder.ID)
	doJSON(ts, "POST", "/folders", map[string]any{"name": "subfolder", "parent_id": folder.ID}, token)

	resp := doGet(ts, fmt.Sprintf("/folders/%d", folder.ID), token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var detail struct {
		Files   []struct{ Name string } `json:"files"`
		Folders []struct{ Name string } `json:"folders"`
	}
	parseJSON(resp, &detail)
	assert.Len(t, detail.Files, 1)
	assert.Len(t, detail.Folders, 1)
}

func TestDeleteEmptyFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "delfolder@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "empty"}, token)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)

	delResp := doDelete(ts, fmt.Sprintf("/folders/%d", folder.ID), token)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)
}

func TestDeleteNonEmptyFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "nonempty@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "notempty"}, token)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)

	uploadFile(ts, token, "blocker.txt", "content", &folder.ID)

	delResp := doDelete(ts, fmt.Sprintf("/folders/%d", folder.ID), token)
	assert.Equal(t, http.StatusPreconditionFailed, delResp.StatusCode)
}
