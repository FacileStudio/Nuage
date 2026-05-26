package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchRequiresQuery(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-req@example.com", "password12345")

	resp := doGet(ts, "/search", token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSearchEndpointFiles(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-files@example.com", "password12345")

	uploadFile(ts, token, "quarterly-report.pdf", "data", nil)
	uploadFile(ts, token, "invoice-2026.pdf", "data", nil)
	uploadFile(ts, token, "photo.jpg", "img", nil)

	resp := doGet(ts, "/search?q=quarterly", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Results []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			Path string `json:"path"`
		} `json:"results"`
		Total int `json:"total"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "quarterly-report.pdf", result.Results[0].Name)
	assert.Equal(t, "file", result.Results[0].Type)
	assert.Equal(t, "/quarterly-report.pdf", result.Results[0].Path)
}

func TestSearchFolders(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-folders@example.com", "password12345")

	doJSON(ts, "POST", "/folders", map[string]string{"name": "Documents"}, token)
	doJSON(ts, "POST", "/folders", map[string]string{"name": "Photos"}, token)

	resp := doGet(ts, "/search?q=doc&type=folder", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Results []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"results"`
		Total int `json:"total"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "Documents", result.Results[0].Name)
	assert.Equal(t, "folder", result.Results[0].Type)
}

func TestSearchTypeFilter(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-type@example.com", "password12345")

	uploadFile(ts, token, "notes.txt", "data", nil)
	doJSON(ts, "POST", "/folders", map[string]string{"name": "notes"}, token)

	resp := doGet(ts, "/search?q=notes&type=file", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Results []struct {
			Type string `json:"type"`
		} `json:"results"`
		Total int `json:"total"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "file", result.Results[0].Type)
}

func TestSearchInFolder(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-infolder@example.com", "password12345")

	folderResp := doJSON(ts, "POST", "/folders", map[string]string{"name": "Work"}, token)
	var folder struct {
		ID int64 `json:"id"`
	}
	parseJSON(folderResp, &folder)

	uploadFile(ts, token, "report.pdf", "data", &folder.ID)
	uploadFile(ts, token, "report.pdf", "other", nil)

	resp := doGet(ts, fmt.Sprintf("/search?q=report&folder_id=%d", folder.ID), token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Results []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"results"`
		Total int `json:"total"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "/Work/report.pdf", result.Results[0].Path)
}

func TestSearchLimit(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-limit@example.com", "password12345")

	for i := 0; i < 5; i++ {
		uploadFile(ts, token, fmt.Sprintf("doc-%d.txt", i), "data", nil)
	}

	resp := doGet(ts, "/search?q=doc&limit=2", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Results []struct{} `json:"results"`
		Total   int        `json:"total"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, 2, result.Total)
}

func TestSearchCaseInsensitive(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "search-case@example.com", "password12345")

	uploadFile(ts, token, "MyDocument.PDF", "data", nil)

	resp := doGet(ts, "/search?q=mydocument", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
		Total int `json:"total"`
	}
	parseJSON(resp, &result)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, "MyDocument.PDF", result.Results[0].Name)
}
