package tests

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/require"
)

type chunkedFile struct {
	File struct {
		ID       int64  `json:"id"`
		LinkedTo string `json:"linked_to"`
	} `json:"file"`
}

func initPublish(ts *testServer, token string, spaceID int64, name string, size int, linkedTo string) *http.Response {
	return initChunkedUpload(ts, token, map[string]any{
		"file_name": name, "total_size": size, "space_id": spaceID, "linked_to": linkedTo,
	})
}

func stageChunked(t *testing.T, ts *testServer, token string, spaceID int64, name, content, linkedTo string) string {
	t.Helper()
	resp := initPublish(ts, token, spaceID, name, len(content), linkedTo)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var init struct {
		SessionID string `json:"session_id"`
	}
	parseJSON(resp, &init)
	require.NotEmpty(t, init.SessionID)
	require.Equal(t, http.StatusCreated, putChunk(ts, token, init.SessionID, 1, []byte(content)).StatusCode)
	return init.SessionID
}

func liveFiles(t *testing.T, ts *testServer, spaceID int64) int64 {
	t.Helper()
	var count int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("space_id = ? AND deleted_at IS NULL", spaceID).Count(&count).Error)
	return count
}

func TestChunkedPublishStoresOnceAndInitShortCircuits(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "cpub-once@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	facileID, spaceID := provisionSpace(t, ts, scoped)

	session := stageChunked(t, ts, scoped, spaceID, "big.pdf", "chunked bytes", "plume:document:7")
	first := completeChunked(ts, scoped, session)
	require.Equal(t, http.StatusCreated, first.StatusCode)
	var created chunkedFile
	parseJSON(first, &created)
	require.Equal(t, "plume:document:7", created.File.LinkedTo)

	var stored schemas.File
	require.NoError(t, ts.db.First(&stored, created.File.ID).Error)
	require.Equal(t, "plume", stored.OriginApp)
	require.Equal(t, "plume:document:7", stored.LinkedTo)

	var sessions int64
	require.NoError(t, ts.db.Model(&schemas.UploadSession{}).Count(&sessions).Error)

	retry := initPublish(ts, scoped, spaceID, "big.pdf", 13, "plume:document:7")
	require.Equal(t, http.StatusOK, retry.StatusCode)
	var short struct {
		SessionID    string `json:"session_id"`
		ExistingFile struct {
			ID int64 `json:"id"`
		} `json:"existing_file"`
	}
	parseJSON(retry, &short)
	require.Empty(t, short.SessionID)
	require.Equal(t, created.File.ID, short.ExistingFile.ID)

	var after int64
	require.NoError(t, ts.db.Model(&schemas.UploadSession{}).Count(&after).Error)
	require.Equal(t, sessions, after, "a short-circuited init creates no session")
	require.EqualValues(t, 1, liveFiles(t, ts, spaceID))
	require.Equal(t, 1, countSpaceObjects(t, facileID))

	bad := initPublish(ts, scoped, spaceID, "big.pdf", 13, "bad\nkey")
	require.Equal(t, http.StatusBadRequest, bad.StatusCode)
}

func TestChunkedCompleteReturnsTheKeyHolderWhenItLostTheRace(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "cpub-late@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	facileID, spaceID := provisionSpace(t, ts, scoped)

	winner := stageChunked(t, ts, scoped, spaceID, "a.pdf", "first bytes", "plume:document:late")
	loser := stageChunked(t, ts, scoped, spaceID, "a.pdf", "second bytes", "plume:document:late")

	first := completeChunked(ts, scoped, winner)
	require.Equal(t, http.StatusCreated, first.StatusCode)
	var created chunkedFile
	parseJSON(first, &created)

	second := completeChunked(ts, scoped, loser)
	require.Equal(t, http.StatusOK, second.StatusCode)
	var again chunkedFile
	parseJSON(second, &again)
	require.Equal(t, created.File.ID, again.File.ID)

	require.EqualValues(t, 1, liveFiles(t, ts, spaceID))
	require.Equal(t, 1, countSpaceObjects(t, facileID))

	var status string
	require.NoError(t, ts.db.Model(&schemas.UploadSession{}).Select("status").Where("id = ?", loser).Scan(&status).Error)
	require.Equal(t, "completed", status)
}

func TestConcurrentChunkedCompletesYieldOneFile(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "cpub-race@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	facileID, spaceID := provisionSpace(t, ts, scoped)

	const callers = 6
	sessions := make([]string, callers)
	for i := range sessions {
		sessions[i] = stageChunked(t, ts, scoped, spaceID, "race.pdf", fmt.Sprintf("bytes number %d", i), "plume:document:race")
	}

	statuses := make(chan int, callers)
	var wg sync.WaitGroup
	for _, id := range sessions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- completeChunked(ts, scoped, id).StatusCode
		}()
	}
	wg.Wait()
	close(statuses)

	created := 0
	for status := range statuses {
		switch status {
		case http.StatusCreated:
			created++
		case http.StatusOK:
		default:
			t.Fatalf("unexpected status %d", status)
		}
	}
	require.Equal(t, 1, created)
	require.EqualValues(t, 1, liveFiles(t, ts, spaceID))
	require.Equal(t, 1, countSpaceObjects(t, facileID))
}

func TestUnrestrictedChunkedUploadIgnoresLinkedTo(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "cpub-plain@example.com", "password12345")
	spaceID := createSpace(t, ts, user, "Plain")

	for i := range 2 {
		session := stageChunked(t, ts, user, spaceID, fmt.Sprintf("plain-%d.bin", i), fmt.Sprintf("plain bytes %d", i), "task:1")
		resp := completeChunked(ts, user, session)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var file chunkedFile
		parseJSON(resp, &file)
		require.Empty(t, file.File.LinkedTo)
	}

	var keyed int64
	require.NoError(t, ts.db.Model(&schemas.UploadSession{}).Where("linked_to <> ''").Count(&keyed).Error)
	require.Zero(t, keyed)
	require.EqualValues(t, 2, liveFiles(t, ts, spaceID))
}

func TestRestrictedTokenLooksUpPublishedFileByKeyInsideItsSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "cpub-lookup@example.com", "password12345")
	sharedID := createSpace(t, ts, user, "Shared")
	hiddenID := createSpace(t, ts, user, "Hidden")
	var shared, hidden schemas.Space
	require.NoError(t, ts.db.First(&shared, sharedID).Error)
	require.NoError(t, ts.db.First(&hidden, hiddenID).Error)

	plume := mintScopedToken(t, ts, user, "plume", shared.FacileID)
	ardoise := mintScopedToken(t, ts, user, "ardoise", shared.FacileID)
	other := mintScopedToken(t, ts, user, "opus", hidden.FacileID)

	require.Equal(t, http.StatusCreated, publish(ts, plume, sharedID, "p.pdf", "p", "plume:document:9").StatusCode)
	require.Equal(t, http.StatusCreated, publish(ts, other, hiddenID, "h.pdf", "h", "plume:document:9").StatusCode)

	type listing struct {
		Files []struct {
			ID        int64  `json:"id"`
			OriginApp string `json:"origin_app"`
			LinkedTo  string `json:"linked_to"`
		} `json:"files"`
	}
	lookup := func(token string, spaceID int64) *http.Response {
		return doGet(ts, fmt.Sprintf("/files?space_id=%d&linked_to=plume:document:9", spaceID), token)
	}

	own := lookup(plume, sharedID)
	require.Equal(t, http.StatusOK, own.StatusCode)
	var found listing
	parseJSON(own, &found)
	require.Len(t, found.Files, 1)
	require.Equal(t, "plume", found.Files[0].OriginApp)

	cross := lookup(ardoise, sharedID)
	require.Equal(t, http.StatusOK, cross.StatusCode)
	var seen listing
	parseJSON(cross, &seen)
	require.Len(t, seen.Files, 1, "origin_app does not hide a file from another app in a shared space")
	require.Equal(t, found.Files[0].ID, seen.Files[0].ID)

	require.Equal(t, http.StatusForbidden, lookup(plume, hiddenID).StatusCode, "a space outside the token")
	require.Equal(t, http.StatusForbidden, doGet(ts, "/files?linked_to=plume:document:9", plume).StatusCode, "no space means personal scope")

	missing := doGet(ts, fmt.Sprintf("/files?space_id=%d&linked_to=nope", sharedID), plume)
	require.Equal(t, http.StatusOK, missing.StatusCode)
	var none listing
	parseJSON(missing, &none)
	require.Empty(t, none.Files)
}

func TestFindOrCreateDoesNotConfirmATakenIDToAnOutsider(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "cpub-404@example.com", "password12345")
	_, outsider := registerUser(ts, "cpub-404b@example.com", "password12345")
	facileID := facile.NewID()

	require.Equal(t, http.StatusCreated, ensureSpace(ts, owner, facileID).StatusCode)
	resp := ensureSpace(ts, outsider, facileID)
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	require.Equal(t, http.StatusOK, ensureSpace(ts, owner, facileID).StatusCode)
}
