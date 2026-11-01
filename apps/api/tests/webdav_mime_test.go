package tests

import (
	"net/http"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebDAVUploadTakesMediaTypeFromExtension(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "dav-mime@example.com", "password12345")

	opaque := "\x00\x01\x02\x03 bytes with no magic number"

	resp := davRequest(ts, "PUT", "/webdav/track.MP3", token, opaque)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file schemas.File
	require.NoError(t, ts.db.Where("name = ?", "track.MP3").First(&file).Error)
	assert.Equal(t, "audio/mpeg", file.MimeType)

	resp = davRequest(ts, "PUT", "/webdav/blob.bin", token, opaque)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var blob schemas.File
	require.NoError(t, ts.db.Where("name = ?", "blob.bin").First(&blob).Error)
	assert.Equal(t, "application/octet-stream", blob.MimeType)

	require.NoError(t, ts.db.Model(&file).Update("mime_type", "application/octet-stream").Error)
	resp = davRequest(ts, "PUT", "/webdav/track.MP3", token, opaque+" v2")
	require.Less(t, resp.StatusCode, 300)
	require.NoError(t, ts.db.Where("id = ?", file.ID).First(&file).Error)
	assert.Equal(t, "audio/mpeg", file.MimeType)
}
