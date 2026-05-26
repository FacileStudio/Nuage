package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQuota(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "quota@example.com", "password12345")

	resp := doGet(ts, "/quota/me", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var usage struct {
		StorageUsed  int64   `json:"storage_used"`
		StorageLimit int64   `json:"storage_limit"`
		Percentage   float64 `json:"percentage"`
	}
	parseJSON(resp, &usage)
	assert.Equal(t, int64(0), usage.StorageUsed)
	assert.Greater(t, usage.StorageLimit, int64(0))
}

func TestQuotaUpdatesOnUpload(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "quota-upload@example.com", "password12345")

	uploadFile(ts, token, "big.txt", "some file content here", nil)

	resp := doGet(ts, "/quota/me", token)
	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(resp, &usage)
	assert.Greater(t, usage.StorageUsed, int64(0))
}

func TestRecalculateQuota(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "recalc@example.com", "password12345")

	uploadFile(ts, token, "file1.txt", "content1", nil)
	uploadFile(ts, token, "file2.txt", "content2", nil)

	resp := doJSON(ts, "POST", "/quota/me/recalculate", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(resp, &usage)
	assert.Greater(t, usage.StorageUsed, int64(0))
}

func TestListAllUsage(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "admin-quota@example.com", "password12345")
	registerUser(ts, "user2@example.com", "password12345")

	resp := doGet(ts, "/quota/users", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Users []struct {
			UserID int64 `json:"user_id"`
		} `json:"users"`
	}
	parseJSON(resp, &result)
	assert.Len(t, result.Users, 2)
}
