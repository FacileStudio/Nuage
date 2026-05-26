package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActivityLogOnUpload(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "activity@example.com", "password12345")

	uploadFile(ts, token, "tracked.txt", "data", nil)

	time.Sleep(100 * time.Millisecond)

	resp := doGet(ts, "/activity/me", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Activities []struct {
			EventType    string `json:"event_type"`
			ResourceType string `json:"resource_type"`
			ResourceName string `json:"resource_name"`
		} `json:"activities"`
		Total int64 `json:"total"`
	}
	parseJSON(resp, &result)
	assert.GreaterOrEqual(t, result.Total, int64(1))

	found := false
	for _, a := range result.Activities {
		if a.EventType == "file.uploaded" {
			found = true
			assert.Equal(t, "file", a.ResourceType)
			assert.Equal(t, "tracked.txt", a.ResourceName)
		}
	}
	assert.True(t, found, "expected file.uploaded activity")
}

func TestActivityPagination(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "paginate@example.com", "password12345")

	for i := 0; i < 5; i++ {
		uploadFile(ts, token, "file.txt", "data", nil)
	}
	time.Sleep(100 * time.Millisecond)

	resp := doGet(ts, "/activity/me?per_page=2&page=1", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Activities []struct{} `json:"activities"`
		Total      int64      `json:"total"`
		Page       int        `json:"page"`
		PerPage    int        `json:"per_page"`
	}
	parseJSON(resp, &result)
	assert.Len(t, result.Activities, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.PerPage)
	assert.GreaterOrEqual(t, result.Total, int64(5))
}

func TestActivityAllEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "all-activity@example.com", "password12345")

	uploadFile(ts, token, "global.txt", "data", nil)
	time.Sleep(100 * time.Millisecond)

	resp := doGet(ts, "/activity", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Total int64 `json:"total"`
	}
	parseJSON(resp, &result)
	assert.GreaterOrEqual(t, result.Total, int64(1))
}

func TestActivityFilters(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "filter@example.com", "password12345")

	uploadFile(ts, token, "filtered.txt", "data", nil)
	doJSON(ts, "POST", "/folders", map[string]string{"name": "folder"}, token)
	time.Sleep(100 * time.Millisecond)

	resp := doGet(ts, "/activity/me?resource_type=folder", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Activities []struct {
			ResourceType string `json:"resource_type"`
		} `json:"activities"`
	}
	parseJSON(resp, &result)
	for _, a := range result.Activities {
		assert.Equal(t, "folder", a.ResourceType)
	}
}
