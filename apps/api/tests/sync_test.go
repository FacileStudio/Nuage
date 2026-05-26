package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncState(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "sync@example.com", "password12345")

	uploadFile(ts, token, "synced.txt", "data", nil)

	resp := doGet(ts, "/sync/state", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var state struct {
		Files      []struct{ Name string } `json:"files"`
		Folders    []struct{}              `json:"folders"`
		ServerTime string                  `json:"server_time"`
	}
	parseJSON(resp, &state)
	assert.Len(t, state.Files, 1)
	assert.NotEmpty(t, state.ServerTime)
}

func TestSyncChanges(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "changes@example.com", "password12345")

	resp := doGet(ts, "/sync/changes?since=2020-01-01T00:00:00Z", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var changes struct {
		Files struct {
			Changed []struct{} `json:"changed"`
			Deleted []struct{} `json:"deleted"`
		} `json:"files"`
		ServerTime string `json:"server_time"`
	}
	parseJSON(resp, &changes)
	assert.NotEmpty(t, changes.ServerTime)
}
