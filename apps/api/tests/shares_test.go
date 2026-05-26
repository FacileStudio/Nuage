package tests

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateShareLink(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "sharer@example.com", "password12345")

	fileResp := uploadFile(ts, token, "shared.txt", "shared content", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(fileResp, &file)

	resp := doJSON(ts, "POST", "/shares", map[string]any{
		"file_id": file.ID, "permission": "view",
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var share struct {
		Token      string `json:"token"`
		Permission string `json:"permission"`
	}
	parseJSON(resp, &share)
	assert.NotEmpty(t, share.Token)
	assert.Equal(t, "view", share.Permission)
}

func TestPublicShareAccess(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "pub-share@example.com", "password12345")

	fileResp := uploadFile(ts, token, "public.txt", "public data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(fileResp, &file)

	shareResp := doJSON(ts, "POST", "/shares", map[string]any{
		"file_id": file.ID, "permission": "view",
	}, token)
	var share struct {
		Token string `json:"token"`
	}
	parseJSON(shareResp, &share)

	resp := doGet(ts, "/shared/"+share.Token, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var pub struct {
		Permission string `json:"permission"`
		File       *struct {
			Name string `json:"name"`
		} `json:"file"`
	}
	parseJSON(resp, &pub)
	assert.Equal(t, "view", pub.Permission)
	assert.Equal(t, "public.txt", pub.File.Name)
}

func TestSharedFileDownload(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dl-share@example.com", "password12345")

	content := "downloadable content"
	fileResp := uploadFile(ts, token, "download.txt", content, nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(fileResp, &file)

	shareResp := doJSON(ts, "POST", "/shares", map[string]any{
		"file_id": file.ID, "permission": "view",
	}, token)
	var share struct {
		Token string `json:"token"`
	}
	parseJSON(shareResp, &share)

	resp := doGet(ts, fmt.Sprintf("/shared/%s/download/%d", share.Token, file.ID), "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, content, string(body))
}

func TestSharePermissionEnforcement(t *testing.T) {
	ts := setupTestServer(t)
	_, ownerToken := registerUser(ts, "owner@example.com", "password12345")
	_, viewerToken := registerUser(ts, "viewer@example.com", "password12345")

	fileResp := uploadFile(ts, ownerToken, "protected.txt", "secret", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(fileResp, &file)

	_ = viewerToken

	shareResp := doJSON(ts, "POST", "/shares", map[string]any{
		"file_id": file.ID, "permission": "view",
	}, ownerToken)
	var share struct {
		Token string `json:"token"`
	}
	parseJSON(shareResp, &share)

	dlResp := doGet(ts, fmt.Sprintf("/shared/%s/download/%d", share.Token, file.ID), "")
	assert.Equal(t, http.StatusOK, dlResp.StatusCode)
	dlResp.Body.Close()
}

func TestListSharedByMe(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "by-me@example.com", "password12345")

	fileResp := uploadFile(ts, token, "mine.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(fileResp, &file)

	doJSON(ts, "POST", "/shares", map[string]any{
		"file_id": file.ID, "permission": "view",
	}, token)

	resp := doGet(ts, "/shares/by-me", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Shares []struct{ ID int64 } `json:"shares"`
	}
	parseJSON(resp, &result)
	assert.Len(t, result.Shares, 1)
}

func TestRevokeShare(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "revoke@example.com", "password12345")

	fileResp := uploadFile(ts, token, "revokable.txt", "data", nil)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(fileResp, &file)

	shareResp := doJSON(ts, "POST", "/shares", map[string]any{
		"file_id": file.ID, "permission": "view",
	}, token)
	var share struct {
		ID    int64  `json:"id"`
		Token string `json:"token"`
	}
	parseJSON(shareResp, &share)

	delResp := doDelete(ts, fmt.Sprintf("/shares/%d", share.ID), token)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)

	pubResp := doGet(ts, "/shared/"+share.Token, "")
	assert.Equal(t, http.StatusNotFound, pubResp.StatusCode)
}
