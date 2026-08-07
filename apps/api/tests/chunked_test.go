package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/presign"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/files"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorageClient(t *testing.T) *storage.Client {
	t.Helper()
	endpoint := os.Getenv("TEST_MINIO_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	client, err := storage.NewClient(storage.MinIOConfig{
		Endpoint:  endpoint,
		AccessKey: "nuage-minio",
		SecretKey: "nuage-internal-storage",
		Bucket:    "nuage-test",
		UseSSL:    false,
	})
	require.NoError(t, err)
	return client
}

func initChunkedUpload(ts *testServer, token string, body map[string]any) *http.Response {
	return doJSON(ts, "POST", "/files/upload/init", body, token)
}

func putChunk(ts *testServer, token, sessionID string, part int, content []byte) *http.Response {
	req := httptest.NewRequest("PUT", fmt.Sprintf("/files/upload/%s/part/%d", sessionID, part), bytes.NewReader(content))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w.Result()
}

func completeChunked(ts *testServer, token, sessionID string) *http.Response {
	return doJSON(ts, "POST", fmt.Sprintf("/files/upload/%s/complete", sessionID), nil, token)
}

func initSession(t *testing.T, ts *testServer, token string, totalSize int64) string {
	t.Helper()
	resp := initChunkedUpload(ts, token, map[string]any{
		"file_name":  "chunked.bin",
		"total_size": totalSize,
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var init struct {
		SessionID string `json:"session_id"`
	}
	parseJSON(resp, &init)
	require.NotEmpty(t, init.SessionID)
	return init.SessionID
}

func TestChunkedUploadComplete(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked@example.com", "password12345")

	part1 := []byte("first chunk of data ")
	part2 := []byte("second chunk of data")
	full := append(append([]byte{}, part1...), part2...)
	wantHash := sha256.Sum256(full)

	sessionID := initSession(t, ts, token, int64(len(full)))

	resp := putChunk(ts, token, sessionID, 1, part1)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp = putChunk(ts, token, sessionID, 2, part2)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp = completeChunked(ts, token, sessionID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		File struct {
			ID   int64  `json:"id"`
			Hash string `json:"hash"`
			Size int64  `json:"size"`
		} `json:"file"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, int64(len(full)), result.File.Size)
	assert.Equal(t, hex.EncodeToString(wantHash[:]), result.File.Hash)

	dlResp := doGet(ts, fmt.Sprintf("/files/%d/download", result.File.ID), token)
	require.Equal(t, http.StatusOK, dlResp.StatusCode)
	body, err := io.ReadAll(dlResp.Body)
	require.NoError(t, err)
	dlResp.Body.Close()
	assert.Equal(t, full, body)
}

func TestChunkedUploadMissingChunk(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked-hole@example.com", "password12345")

	part1 := []byte("aaaa")
	part2 := []byte("bbbb")
	part3 := []byte("cccc")
	total := int64(len(part1) + len(part2) + len(part3))

	sessionID := initSession(t, ts, token, total)

	require.Equal(t, http.StatusCreated, putChunk(ts, token, sessionID, 1, part1).StatusCode)
	require.Equal(t, http.StatusCreated, putChunk(ts, token, sessionID, 3, part3).StatusCode)

	resp := completeChunked(ts, token, sessionID)
	require.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)

	require.Equal(t, http.StatusCreated, putChunk(ts, token, sessionID, 2, part2).StatusCode)

	resp = completeChunked(ts, token, sessionID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestChunkedUploadSizeMismatch(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked-short@example.com", "password12345")

	sessionID := initSession(t, ts, token, 100)
	require.Equal(t, http.StatusCreated, putChunk(ts, token, sessionID, 1, []byte("only ten b")).StatusCode)

	resp := completeChunked(ts, token, sessionID)
	require.Equal(t, http.StatusPreconditionFailed, resp.StatusCode)
}

func TestChunkedUploadChunkExceedsTotal(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked-over@example.com", "password12345")

	sessionID := initSession(t, ts, token, 10)
	resp := putChunk(ts, token, sessionID, 1, []byte("this is way more than ten bytes"))
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestChunkedInitQuotaCheck(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked-quota@example.com", "password12345")

	resp := initChunkedUpload(ts, token, map[string]any{
		"file_name":  "huge.bin",
		"total_size": int64(51) * 1024 * 1024 * 1024,
	})
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestChunkedUploadFolderOwnership(t *testing.T) {
	ts := setupTestServer(t)
	_, ownerToken := registerUser(ts, "folder-owner@example.com", "password12345")
	_, attackerToken := registerUser(ts, "folder-attacker@example.com", "password12345")

	resp := doJSON(ts, "POST", "/folders", map[string]string{"name": "private"}, ownerToken)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &folder)

	resp = initChunkedUpload(ts, attackerToken, map[string]any{
		"file_name":  "intruder.txt",
		"total_size": 10,
		"folder_id":  folder.ID,
	})
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	resp = initChunkedUpload(ts, ownerToken, map[string]any{
		"file_name":  "mine.txt",
		"total_size": 10,
		"folder_id":  folder.ID,
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestChunkedUploadSpaceMembership(t *testing.T) {
	ts := setupTestServer(t)
	userID, token := registerUser(ts, "space-chunk@example.com", "password12345")

	space := &schemas.Space{FacileID: fmt.Sprintf("space-chunk-%d", time.Now().UnixNano()), Name: "Team"}
	require.NoError(t, ts.db.Create(space).Error)

	body := map[string]any{
		"file_name":  "team.txt",
		"total_size": 10,
		"space_id":   space.ID,
	}

	resp := initChunkedUpload(ts, token, body)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)

	var uid int64
	_, err := fmt.Sscanf(userID, "%d", &uid)
	require.NoError(t, err)
	require.NoError(t, ts.db.Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: uid}).Error)

	resp = initChunkedUpload(ts, token, body)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestChunkedCompleteIdempotent(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked-idem@example.com", "password12345")

	content := []byte("idempotent content")
	sessionID := initSession(t, ts, token, int64(len(content)))
	require.Equal(t, http.StatusCreated, putChunk(ts, token, sessionID, 1, content).StatusCode)

	resp := completeChunked(ts, token, sessionID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp = completeChunked(ts, token, sessionID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	var count int64
	ts.db.Model(&schemas.File{}).Where("name = ?", "chunked.bin").Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestChunkedSessionSweeper(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "chunked-sweep@example.com", "password12345")

	content := []byte("abandoned chunk")
	sessionID := initSession(t, ts, token, int64(len(content))+100)
	require.Equal(t, http.StatusCreated, putChunk(ts, token, sessionID, 1, content).StatusCode)

	require.NoError(t, ts.db.Model(&schemas.UploadSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"expires_at": time.Now().Add(-time.Hour),
			"created_at": time.Now().Add(-25 * time.Hour),
		}).Error)

	storageClient := newTestStorageClient(t)
	notifier := nook.NewNotifier(ts.db)
	actLogger := activity.NewLogger(ts.db)
	quotaService := quota.NewService(ts.db)
	presignSecret := presign.DeriveSecret("test-secret", "nuage-presign-v1")
	fileService := files.NewService(ts.db, storageClient, notifier, actLogger, quotaService, presignSecret)

	swept, err := fileService.SweepExpiredSessions(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, swept, 1)

	var sessionCount int64
	ts.db.Model(&schemas.UploadSession{}).Where("id = ?", sessionID).Count(&sessionCount)
	assert.Equal(t, int64(0), sessionCount)

	var chunkCount int64
	ts.db.Model(&schemas.UploadChunk{}).Where("session_id = ?", sessionID).Count(&chunkCount)
	assert.Equal(t, int64(0), chunkCount)

	objects, err := storageClient.ListObjects(context.Background(), fmt.Sprintf("chunks/%s/", sessionID))
	require.NoError(t, err)
	assert.Empty(t, objects)
}
