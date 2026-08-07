package tests

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetQuota(t *testing.T) {
	ts := setupSettledServer(t)
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
	ts := setupSettledServer(t)
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
	ts := setupSettledServer(t)
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
	ts := setupSettledServer(t)
	_, adminToken := registerUser(ts, "admin-quota@example.com", "password12345")
	registerUser(ts, "user2@example.com", "password12345")

	resp := doGet(ts, "/quota/users", adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Users []struct {
			UserID int64 `json:"user_id"`
		} `json:"users"`
	}
	parseJSON(resp, &result)
	assert.Len(t, result.Users, 2)
}

func TestQuotaAdminEndpointsRequireAdmin(t *testing.T) {
	ts := setupSettledServer(t)
	_, adminToken := registerUser(ts, "quota-admin@example.com", "password12345")
	victimID, victimToken := registerUser(ts, "quota-victim@example.com", "password12345")

	resp := doGet(ts, "/quota/users", victimToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	resp = doJSON(ts, "PUT", "/quota/users/"+victimID, map[string]int64{"storage_limit": -1}, victimToken)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	resp = doGet(ts, "/quota/users", adminToken)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp = doJSON(ts, "PUT", "/quota/users/"+victimID, map[string]int64{"storage_limit": -5}, adminToken)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	resp = doJSON(ts, "PUT", "/quota/users/"+victimID, map[string]int64{"storage_limit": 1024}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var usage struct {
		StorageLimit int64 `json:"storage_limit"`
	}
	parseJSON(resp, &usage)
	assert.Equal(t, int64(1024), usage.StorageLimit)
}

func TestQuotaRecalculateCountsTrashedFiles(t *testing.T) {
	ts := setupSettledServer(t)
	_, token := registerUser(ts, "recalc-trash@example.com", "password12345")

	fileID := uploadTestFile(t, ts, token, "trashed.txt", "trashed file content", nil)

	var fileSize int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("id = ?", fileID).Pluck("size", &fileSize).Error)
	require.Greater(t, fileSize, int64(0))

	delResp := doDelete(ts, fmt.Sprintf("/files/%d", fileID), token)
	require.Equal(t, http.StatusOK, delResp.StatusCode)

	resp := doJSON(ts, "POST", "/quota/me/recalculate", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(resp, &usage)
	assert.Equal(t, fileSize, usage.StorageUsed)

	permResp := doDelete(ts, fmt.Sprintf("/trash/file/%d", fileID), token)
	require.Equal(t, http.StatusOK, permResp.StatusCode)

	resp = doJSON(ts, "POST", "/quota/me/recalculate", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	parseJSON(resp, &usage)
	assert.Equal(t, int64(0), usage.StorageUsed)
}

func TestQuotaRecalculateCountsVersions(t *testing.T) {
	ts := setupSettledServer(t)
	rawID, token := registerUser(ts, "recalc-versions@example.com", "password12345")
	userID, err := strconv.ParseInt(rawID, 10, 64)
	require.NoError(t, err)

	fileID := uploadTestFile(t, ts, token, "versioned.txt", "first version content", nil)

	var file schemas.File
	require.NoError(t, ts.db.First(&file, fileID).Error)
	version := schemas.FileVersion{
		FileID:    fileID,
		Version:   1,
		BucketKey: file.BucketKey + ".v1",
		Hash:      file.Hash,
		Size:      123,
		CreatedBy: userID,
	}
	require.NoError(t, ts.db.Create(&version).Error)

	resp := doJSON(ts, "POST", "/quota/me/recalculate", nil, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(resp, &usage)
	assert.Equal(t, file.Size+version.Size, usage.StorageUsed)
}

func TestQuotaUpdateUsageAtomic(t *testing.T) {
	ts := setupSettledServer(t)
	rawID, _ := registerUser(ts, "atomic@example.com", "password12345")
	userID, err := strconv.ParseInt(rawID, 10, 64)
	require.NoError(t, err)

	svc := quota.NewService(ts.db)
	_, err = svc.GetUsage(context.Background(), userID)
	require.NoError(t, err)

	const workers = 20
	const delta = int64(1000)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, svc.UpdateUsage(context.Background(), userID, delta))
		}()
	}
	wg.Wait()

	usage, err := svc.GetUsage(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, int64(workers)*delta, usage.StorageUsed)
}
