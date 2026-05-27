package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresignFile(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign@example.com", "password12345")

	content := "presigned content here"
	resp := uploadFile(ts, token, "secret.txt", content, nil)
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

	parsed, err := url.Parse(result.URL)
	require.NoError(t, err)

	dlReq := httptest.NewRequest("GET", parsed.RequestURI(), nil)
	dlW := httptest.NewRecorder()
	ts.router.ServeHTTP(dlW, dlReq)
	dlResp := dlW.Result()

	require.Equal(t, http.StatusOK, dlResp.StatusCode)
	body, _ := io.ReadAll(dlResp.Body)
	dlResp.Body.Close()
	assert.Equal(t, content, string(body))
	assert.Equal(t, "application/octet-stream", dlResp.Header.Get("Content-Type"))
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

	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), map[string]int64{"expires_in": 300}, token)
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

func TestPresignedDownloadInvalidToken(t *testing.T) {
	ts := setupTestServer(t)

	req := httptest.NewRequest("GET", "/presigned/garbage-token", nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	resp := w.Result()

	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPresignedDownloadTamperedToken(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "presign-tamper@example.com", "password12345")

	resp := uploadFile(ts, token, "tamper.txt", "tamper content", nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)

	presignResp := doJSON(ts, "POST", fmt.Sprintf("/files/%d/presign", file.ID), nil, token)
	require.Equal(t, http.StatusOK, presignResp.StatusCode)

	var result struct {
		URL string `json:"url"`
	}
	json.NewDecoder(presignResp.Body).Decode(&result)
	presignResp.Body.Close()

	parsed, err := url.Parse(result.URL)
	require.NoError(t, err)

	tampered := parsed.RequestURI() + "tampered"
	req := httptest.NewRequest("GET", tampered, nil)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	dlResp := w.Result()

	require.Equal(t, http.StatusUnauthorized, dlResp.StatusCode)
}
