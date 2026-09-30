package tests

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateDeduplicatesAfterTakingTheNameLock holds a folder's name lock, lets a
// create for a taken name run while the lock is held, then inserts that name and
// commits. The create can only pick a free name if it deduplicates after it
// acquires the lock. Deduplicating before the lock — the old order — picks the
// name that was free at that moment and lands a second row under it.
func TestCreateDeduplicatesAfterTakingTheNameLock(t *testing.T) {
	ts := setupTestServer(t)
	userID, token := registerUser(ts, "dedup@example.com", "password12345")
	owner, err := strconv.ParseInt(userID, 10, 64)
	require.NoError(t, err)

	tx := ts.db.Begin()
	require.NoError(t, tx.Error)
	key := fmt.Sprintf("nuage:name:file:user-%d:root", owner)
	require.NoError(t, tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error)

	done := make(chan *http.Response, 1)
	go func() {
		done <- uploadFile(ts, token, "race.txt", "second", nil)
	}()

	time.Sleep(300 * time.Millisecond)

	require.NoError(t, tx.Create(&schemas.File{
		FacileID:   "race-winner",
		Name:       "race.txt",
		MimeType:   "text/plain",
		BucketKey:  "race/winner",
		UploadedBy: owner,
	}).Error)
	require.NoError(t, tx.Commit().Error)

	resp := <-done
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file struct {
		Name string `json:"name"`
	}
	parseJSON(resp, &file)
	assert.Equal(t, "race (1).txt", file.Name, "the create only sees the taken name after the lock")

	var sameName int64
	require.NoError(t, ts.db.Model(&schemas.File{}).
		Where("name = ? AND uploaded_by = ? AND deleted_at IS NULL", "race.txt", owner).
		Count(&sameName).Error)
	assert.Equal(t, int64(1), sameName, "one name must never hold two rows")
}

// TestCreateFolderDeduplicatesAfterTakingTheNameLock is the folder half: the
// same lock must cover a folder's name check and insert.
func TestCreateFolderDeduplicatesAfterTakingTheNameLock(t *testing.T) {
	ts := setupTestServer(t)
	userID, token := registerUser(ts, "folder-dedup@example.com", "password12345")
	owner, err := strconv.ParseInt(userID, 10, 64)
	require.NoError(t, err)

	tx := ts.db.Begin()
	require.NoError(t, tx.Error)
	key := fmt.Sprintf("nuage:name:folder:user-%d:root", owner)
	require.NoError(t, tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", key).Error)

	done := make(chan *http.Response, 1)
	go func() {
		done <- doJSON(ts, "POST", "/folders", map[string]string{"name": "docs"}, token)
	}()

	time.Sleep(300 * time.Millisecond)

	require.NoError(t, tx.Create(&schemas.Folder{
		FacileID: "folder-winner",
		Name:     "docs",
		OwnerID:  owner,
	}).Error)
	require.NoError(t, tx.Commit().Error)

	resp := <-done
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var folder struct {
		Name string `json:"name"`
	}
	parseJSON(resp, &folder)
	assert.Equal(t, "docs (1)", folder.Name, "the create only sees the taken name after the lock")
}
