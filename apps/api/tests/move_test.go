package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func createFolder(t *testing.T, ts *testServer, token string, body map[string]any) int64 {
	resp := doJSON(ts, "POST", "/folders", body, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)
	return folder.ID
}

func TestFolderCannotMoveIntoItsOwnSubtree(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "cycle@example.com", "password12345")

	parent := createFolder(t, ts, token, map[string]any{"name": "parent"})
	child := createFolder(t, ts, token, map[string]any{"name": "child", "parent_id": parent})
	grandchild := createFolder(t, ts, token, map[string]any{"name": "grandchild", "parent_id": child})

	resp := doJSON(ts, "PUT", fmt.Sprintf("/folders/%d", parent), map[string]any{"parent_id": grandchild}, token)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = doJSON(ts, "PUT", fmt.Sprintf("/folders/%d", grandchild), map[string]any{"parent_id": 0}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFileCannotMoveAcrossSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "acrossspace@example.com", "password12345")

	spaceID := createSpace(t, ts, token, "team")
	spaceFolder := createFolder(t, ts, token, map[string]any{"name": "shared", "space_id": spaceID})

	resp := uploadFile(ts, token, "personal.txt", "personal bytes", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	resp = doJSON(ts, "PUT", fmt.Sprintf("/files/%d", file.ID), map[string]any{"folder_id": spaceFolder}, token)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestWebDAVFolderCannotMoveIntoItsOwnSubtree(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-cycle@example.com", "password12345")

	require.Equal(t, http.StatusCreated, davRequest(ts, "MKCOL", "/webdav/a", token, "").StatusCode)
	require.Equal(t, http.StatusCreated, davRequest(ts, "MKCOL", "/webdav/a/b", token, "").StatusCode)

	req := httptest.NewRequest("MOVE", "/webdav/a", nil)
	req.SetBasicAuth("user@example.com", token)
	req.Header.Set("Destination", "/webdav/a/b/a")
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	require.GreaterOrEqual(t, w.Result().StatusCode, 400)

	require.Equal(t, http.StatusMultiStatus, davRequest(ts, "PROPFIND", "/webdav/a/b", token, "").StatusCode)
}

func TestSpaceMemberCanMoveTeammateFileButOutsiderCannot(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "owner-move@example.com", "password12345")
	memberID, member := registerUser(ts, "member-move@example.com", "password12345")
	_, outsider := registerUser(ts, "outsider-move@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Move Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)
	folder := createFolder(t, ts, owner, map[string]any{"name": "archive", "space_id": spaceID})

	resp := uploadFileToSpace(ts, owner, "plan.txt", "plan", spaceID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	path := fmt.Sprintf("/files/%d", file.ID)
	require.Equal(t, http.StatusNotFound, doJSON(ts, "PUT", path, map[string]any{"folder_id": folder}, outsider).StatusCode)
	require.Equal(t, http.StatusOK, doJSON(ts, "PUT", path, map[string]any{"folder_id": folder}, member).StatusCode)
	require.Equal(t, http.StatusOK, doJSON(ts, "PUT", path, map[string]any{"folder_id": 0}, member).StatusCode)
}
