package tests

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrashAndRestore(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "trash@example.com", "password123")

	resp := uploadFile(ts, token, "trashme.txt", "data", nil)
	var file struct{ ID int64 `json:"id"` }
	parseJSON(resp, &file)

	doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token)

	trashResp := doGet(ts, "/trash", token)
	require.Equal(t, http.StatusOK, trashResp.StatusCode)

	var trashList struct {
		Items []struct {
			Type string `json:"type"`
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	}
	parseJSON(trashResp, &trashList)
	assert.Len(t, trashList.Items, 1)
	assert.Equal(t, "file", trashList.Items[0].Type)

	restoreResp := doJSON(ts, "POST", fmt.Sprintf("/trash/file/%d/restore", file.ID), nil, token)
	assert.Equal(t, http.StatusOK, restoreResp.StatusCode)

	getResp := doGet(ts, fmt.Sprintf("/files/%d", file.ID), token)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
}

func TestPermanentDelete(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "permdelete@example.com", "password123")

	resp := uploadFile(ts, token, "goodbye.txt", "data", nil)
	var file struct{ ID int64 `json:"id"` }
	parseJSON(resp, &file)

	doDelete(ts, fmt.Sprintf("/files/%d", file.ID), token)

	delResp := doDelete(ts, fmt.Sprintf("/trash/file/%d", file.ID), token)
	assert.Equal(t, http.StatusOK, delResp.StatusCode)

	trashResp := doGet(ts, "/trash", token)
	var trashList struct {
		Items []struct{} `json:"items"`
	}
	parseJSON(trashResp, &trashList)
	assert.Empty(t, trashList.Items)
}
