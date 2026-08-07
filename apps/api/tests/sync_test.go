package tests

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type syncItem struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Hash      string `json:"hash"`
	FolderID  *int64 `json:"folder_id"`
	SpaceID   *int64 `json:"space_id"`
	UpdatedAt string `json:"updated_at"`
}

type syncDeleted struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	SpaceID   *int64 `json:"space_id"`
	Permanent bool   `json:"permanent"`
}

type syncChanges struct {
	Files struct {
		Changed []syncItem    `json:"changed"`
		Deleted []syncDeleted `json:"deleted"`
	} `json:"files"`
	Folders struct {
		Changed []syncItem    `json:"changed"`
		Deleted []syncDeleted `json:"deleted"`
	} `json:"folders"`
	ServerTime string `json:"server_time"`
}

type syncState struct {
	Files      []syncItem `json:"files"`
	Folders    []syncItem `json:"folders"`
	ServerTime string     `json:"server_time"`
}

func fetchSyncState(t *testing.T, ts *testServer, token string) syncState {
	t.Helper()
	resp := doGet(ts, "/sync/state", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var state syncState
	parseJSON(resp, &state)
	require.NotEmpty(t, state.ServerTime)
	return state
}

func fetchSyncChanges(t *testing.T, ts *testServer, token, since string) syncChanges {
	t.Helper()
	resp := doGet(ts, "/sync/changes?since="+url.QueryEscape(since), token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var changes syncChanges
	parseJSON(resp, &changes)
	require.NotEmpty(t, changes.ServerTime)
	return changes
}

func TestSyncState(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "sync@example.com", "password12345")

	require.Equal(t, http.StatusCreated, uploadFile(ts, token, "synced.txt", "data", nil).StatusCode)

	state := fetchSyncState(t, ts, token)
	require.Len(t, state.Files, 1)
	assert.Equal(t, "synced.txt", state.Files[0].Name)
	assert.NotEmpty(t, state.Files[0].Hash)
}

func TestSyncChangesEmptyForFreshAccount(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "changes@example.com", "password12345")

	changes := fetchSyncChanges(t, ts, token, "2020-01-01T00:00:00Z")
	assert.Empty(t, changes.Files.Changed)
	assert.Empty(t, changes.Files.Deleted)
	assert.Empty(t, changes.Folders.Changed)
}

func TestSyncCursorRoundTripSeesLaterWrite(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "cursor@example.com", "password12345")

	require.Equal(t, http.StatusCreated, uploadFile(ts, token, "first.txt", "one", nil).StatusCode)

	first := fetchSyncChanges(t, ts, token, "2020-01-01T00:00:00Z")
	require.Len(t, first.Files.Changed, 1)

	require.Equal(t, http.StatusCreated, uploadFile(ts, token, "second.txt", "two", nil).StatusCode)

	second := fetchSyncChanges(t, ts, token, first.ServerTime)
	names := map[string]bool{}
	for _, item := range second.Files.Changed {
		names[item.Name] = true
	}
	assert.True(t, names["second.txt"], "a write after the cursor must appear in the next poll")
}

func TestSyncChangesReportsRename(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "rename-sync@example.com", "password12345")

	resp := uploadFile(ts, token, "before.txt", "data", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	cursor := fetchSyncChanges(t, ts, token, "2020-01-01T00:00:00Z").ServerTime

	renameResp := doJSON(ts, "PUT", fmt.Sprintf("/files/%d", file.ID), map[string]string{"name": "after.txt"}, token)
	require.Equal(t, http.StatusOK, renameResp.StatusCode)

	changes := fetchSyncChanges(t, ts, token, cursor)
	require.NotEmpty(t, changes.Files.Changed)
	found := false
	for _, item := range changes.Files.Changed {
		if item.ID == file.ID && item.Name == "after.txt" {
			found = true
		}
	}
	assert.True(t, found, "a rename must surface as a changed file")
}

func TestSyncReportsTrashedFileAsDeleted(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "trash-sync@example.com", "password12345")

	resp := uploadFile(ts, token, "doomed.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	cursor := fetchSyncChanges(t, ts, token, "2020-01-01T00:00:00Z").ServerTime

	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token).StatusCode)

	changes := fetchSyncChanges(t, ts, token, cursor)
	require.Len(t, changes.Files.Deleted, 1)
	assert.Equal(t, file.ID, changes.Files.Deleted[0].ID)
	assert.False(t, changes.Files.Deleted[0].Permanent, "a trashed file is recoverable, not permanently gone")
}

func TestSyncReportsPermanentDeleteAfterEmptyTrash(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "tombstone-sync@example.com", "password12345")

	resp := uploadFile(ts, token, "purged.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	cursor := fetchSyncChanges(t, ts, token, "2020-01-01T00:00:00Z").ServerTime

	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token).StatusCode)
	require.Equal(t, http.StatusOK, doDelete(ts, "/trash", token).StatusCode)

	changes := fetchSyncChanges(t, ts, token, cursor)
	require.Len(t, changes.Files.Deleted, 1,
		"a client whose cursor predates the purge must still learn the file is gone")
	assert.Equal(t, file.ID, changes.Files.Deleted[0].ID)
	assert.True(t, changes.Files.Deleted[0].Permanent)
}

func TestSyncIncludesSpaceFilesForMembers(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "sync-space-owner@example.com", "password12345")
	memberID, member := registerUser(ts, "sync-space-member@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Sync Space")
	addSpaceMember(t, ts, owner, spaceID, memberID)

	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, owner, "shared.txt", "data", spaceID).StatusCode)
	require.Equal(t, http.StatusCreated, uploadFile(ts, owner, "personal.txt", "mine", nil).StatusCode)

	state := fetchSyncState(t, ts, member)
	names := map[string]*int64{}
	for _, item := range state.Files {
		names[item.Name] = item.SpaceID
	}

	spaceOfFile, ok := names["shared.txt"]
	require.True(t, ok, "a space member must receive files uploaded by other members")
	require.NotNil(t, spaceOfFile, "space files must carry space_id so clients can namespace them")
	assert.Equal(t, spaceID, *spaceOfFile)

	_, leaked := names["personal.txt"]
	assert.False(t, leaked, "another user's personal files must never sync")
}

func TestSyncExcludesNonMembers(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "sync-excl-owner@example.com", "password12345")
	_, outsider := registerUser(ts, "sync-excl-out@example.com", "password12345")

	spaceID := createSpace(t, ts, owner, "Closed Space")
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, owner, "classified.txt", "data", spaceID).StatusCode)

	state := fetchSyncState(t, ts, outsider)
	assert.Empty(t, state.Files, "a non-member must not receive space files")
}
