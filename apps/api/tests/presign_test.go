package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresignFile(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign@example.com", "password12345")

	resp := uploadFile(ts, token, "secret.txt", "presigned content", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), nil, token)
	require.Equal(t, http.StatusOK, presignResp.StatusCode)

	var result struct {
		URL       string `json:"url"`
		ExpiresAt string `json:"expires_at"`
	}
	parseJSON(presignResp, &result)
	assert.NotEmpty(t, result.URL)
	assert.NotEmpty(t, result.ExpiresAt)
}

func TestPresignFileCustomExpiry(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign-expiry@example.com", "password12345")

	resp := uploadFile(ts, token, "timed.txt", "timed content", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	expiresIn := int64(300)
	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), map[string]int64{"expires_in": expiresIn}, token)
	require.Equal(t, http.StatusOK, presignResp.StatusCode)

	var result struct {
		URL       string `json:"url"`
		ExpiresAt string `json:"expires_at"`
	}
	parseJSON(presignResp, &result)
	assert.NotEmpty(t, result.URL)
	assert.NotEmpty(t, result.ExpiresAt)
}

func TestPresignFileTooShort(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign-short@example.com", "password12345")

	resp := uploadFile(ts, token, "short.txt", "content", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), map[string]int64{"expires_in": 10}, token)
	require.Equal(t, http.StatusBadRequest, presignResp.StatusCode)
}

func TestPresignFileTooLong(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign-long@example.com", "password12345")

	resp := uploadFile(ts, token, "long.txt", "content", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), map[string]int64{"expires_in": 999999}, token)
	require.Equal(t, http.StatusBadRequest, presignResp.StatusCode)
}

func TestPresignFileNotFound(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign-missing@example.com", "password12345")

	presignResp := doJSON(ts, "POST", "/files/99999/presign", nil, token)
	require.Equal(t, http.StatusNotFound, presignResp.StatusCode)
}

func TestPresignFileUnauthorized(t *testing.T) {
	ts := setupTestServer(t)

	presignResp := doJSON(ts, "POST", "/files/1/presign", nil, "")
	require.Equal(t, http.StatusUnauthorized, presignResp.StatusCode)
}

func TestPresignFileOtherUser(t *testing.T) {
	ts := setupTestServer(t)
	_, token1 := registerUser(ts, "presign-owner@example.com", "password12345")
	_, token2 := registerUser(ts, "presign-other@example.com", "password12345")

	resp := uploadFile(ts, token1, "private.txt", "private content", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), nil, token2)
	require.Equal(t, http.StatusNotFound, presignResp.StatusCode)
}
