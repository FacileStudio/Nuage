package tests

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pngBytes(t *testing.T) string {
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for x := 0; x < 800; x++ {
		for y := 0; y < 600; y++ {
			img.Set(x, y, color.RGBA{uint8(x / 4), uint8(y / 3), 120, 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.String()
}

func TestSharedThumbnailServesImagesInsideTheShareOnly(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "share-thumb@example.com", "password12345")

	folderID := createFolder(t, ts, token, map[string]any{"name": "album"})
	inside := uploadForShare(t, ts, token, "inside.png", pngBytes(t), &folderID)
	text := uploadForShare(t, ts, token, "notes.txt", "plain words", &folderID)
	outside := uploadForShare(t, ts, token, "outside.png", pngBytes(t), nil)
	shared := createShare(t, ts, token, map[string]any{"folder_id": folderID, "permission": "view"})

	resp := doGet(ts, fmt.Sprintf("/shared/%s/thumbnail/%d", shared, inside), "")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/jpeg", resp.Header.Get("Content-Type"))
	assert.NotEmpty(t, resp.Header.Get("ETag"))

	small := doGet(ts, fmt.Sprintf("/shared/%s/thumbnail/%d?w=320", shared, inside), "")
	require.Equal(t, http.StatusOK, small.StatusCode)
	assert.Equal(t, http.StatusBadRequest, doGet(ts, fmt.Sprintf("/shared/%s/thumbnail/%d?w=999", shared, inside), "").StatusCode)
	assert.Equal(t, http.StatusNotFound, doGet(ts, fmt.Sprintf("/shared/%s/thumbnail/%d", shared, text), "").StatusCode)
	assert.Equal(t, http.StatusForbidden, doGet(ts, fmt.Sprintf("/shared/%s/thumbnail/%d", shared, outside), "").StatusCode)

	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/folders/%d", folderID), token).StatusCode)
	assert.Equal(t, http.StatusNotFound, doGet(ts, fmt.Sprintf("/shared/%s/thumbnail/%d", shared, inside), "").StatusCode)
}
