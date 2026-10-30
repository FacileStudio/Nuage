package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uploadedFileID(t *testing.T, resp *http.Response) int64 {
	t.Helper()
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)
	require.NotZero(t, file.ID)
	return file.ID
}

func TestIdenticalReuploadIsTheSameFile(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "identical@example.com", "password12345")

	first := uploadFile(ts, token, "same.txt", "same bytes", nil)
	require.Equal(t, http.StatusCreated, first.StatusCode)
	firstID := uploadedFileID(t, first)

	again := uploadFile(ts, token, "same.txt", "same bytes", nil)
	require.Equal(t, http.StatusOK, again.StatusCode)
	assert.Equal(t, firstID, uploadedFileID(t, again))

	var rows int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("deleted_at IS NULL").Count(&rows).Error)
	assert.Equal(t, int64(1), rows, "no copy was stored")
}

func TestDifferentContentUnderTheSameNameStillGetsACopyName(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "different@example.com", "password12345")

	require.Equal(t, http.StatusCreated, uploadFile(ts, token, "doc.txt", "version one", nil).StatusCode)
	second := uploadFile(ts, token, "doc.txt", "version two", nil)
	require.Equal(t, http.StatusCreated, second.StatusCode)

	var names []string
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("deleted_at IS NULL").Order("id").Pluck("name", &names).Error)
	assert.Equal(t, []string{"doc.txt", "doc (1).txt"}, names)
}

func TestConcurrentIdenticalUploadsAreOneFile(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "racing@example.com", "password12345")

	const uploads = 6
	ids := make([]int64, uploads)
	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := uploadFile(ts, token, "race.csv", "identical payload", nil)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var file struct {
				ID int64 `json:"id"`
			}
			_ = json.Unmarshal(body, &file)
			ids[i] = file.ID
			if file.ID == 0 {
				t.Logf("upload %d: status %d body %s", i, resp.StatusCode, string(body))
			}
		}()
	}
	wg.Wait()

	var rows int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("deleted_at IS NULL").Count(&rows).Error)
	assert.Equal(t, int64(1), rows, "the racing uploads collapsed into one file")
	for _, id := range ids {
		assert.Equal(t, ids[0], id)
	}
}
