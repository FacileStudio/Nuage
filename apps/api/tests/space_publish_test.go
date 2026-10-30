package tests

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/require"
)

func mintProvisionToken(t *testing.T, ts *testServer, token, app string, facileIDs ...string) string {
	t.Helper()
	resp := doJSON(ts, "POST", "/users/me/api-token", map[string]any{
		"name": app, "app": app, "provision": true, "space_facile_ids": facileIDs,
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct {
		Token string `json:"token"`
	}
	parseJSON(resp, &created)
	require.NotEmpty(t, created.Token)
	return created.Token
}

func provisionSpace(t *testing.T, ts *testServer, token string) (string, int64) {
	t.Helper()
	facileID := facile.NewID()
	resp := ensureSpace(ts, token, facileID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var space provisioned
	parseJSON(resp, &space)
	return facileID, space.ID
}

func publish(ts *testServer, token string, spaceID int64, name, content, linkedTo string) *http.Response {
	return uploadFileWithFields(ts, token, name, content, map[string]string{
		"space_id":  fmt.Sprint(spaceID),
		"linked_to": linkedTo,
	})
}

func TestProvisionTokenCreatesAndReachesItsSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "prov-owner@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")

	facileID, spaceID := provisionSpace(t, ts, scoped)

	var space schemas.Space
	require.NoError(t, ts.db.Where("facile_id = ?", facileID).First(&space).Error)
	require.Equal(t, "plume", space.ProvisionedBy)

	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, scoped, "a.pdf", "a", spaceID).StatusCode)
	require.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/files?space_id=%d", spaceID), scoped).StatusCode)
	require.Equal(t, http.StatusOK, ensureSpace(ts, scoped, facileID).StatusCode)
	require.Equal(t, http.StatusForbidden, uploadFile(ts, scoped, "p.txt", "p", nil).StatusCode)
}

func TestProvisionTokenCannotReachForeignSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "prov-foreign@example.com", "password12345")

	userSpaceID := createSpace(t, ts, user, "Mine")
	var userSpace schemas.Space
	require.NoError(t, ts.db.First(&userSpace, userSpaceID).Error)

	other := mintProvisionToken(t, ts, user, "ardoise")
	_, otherSpaceID := provisionSpace(t, ts, other)
	var otherSpace schemas.Space
	require.NoError(t, ts.db.First(&otherSpace, otherSpaceID).Error)

	plume := mintProvisionToken(t, ts, user, "plume")

	require.Equal(t, http.StatusForbidden, ensureSpace(ts, plume, userSpace.FacileID).StatusCode, "user-created space")
	require.Equal(t, http.StatusForbidden, ensureSpace(ts, plume, otherSpace.FacileID).StatusCode, "another app's space")
	require.Equal(t, http.StatusForbidden, uploadFileToSpace(ts, plume, "x.txt", "x", userSpaceID).StatusCode)
	require.Equal(t, http.StatusForbidden, uploadFileToSpace(ts, plume, "x.txt", "x", otherSpaceID).StatusCode)
	require.Equal(t, http.StatusForbidden, doGet(ts, fmt.Sprintf("/files?space_id=%d", otherSpaceID), plume).StatusCode)
}

func TestProvisionTokenKeepsExplicitList(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "prov-list@example.com", "password12345")
	userSpaceID := createSpace(t, ts, user, "Listed")
	var userSpace schemas.Space
	require.NoError(t, ts.db.First(&userSpace, userSpaceID).Error)

	scoped := mintProvisionToken(t, ts, user, "plume", userSpace.FacileID)
	require.Equal(t, http.StatusOK, ensureSpace(ts, scoped, userSpace.FacileID).StatusCode)
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, scoped, "l.txt", "l", userSpaceID).StatusCode)
}

func TestNonProvisionTokenCannotCreateUnlistedSpace(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "prov-none@example.com", "password12345")
	scoped := mintScopedToken(t, ts, user, "plume", facile.NewID())
	require.Equal(t, http.StatusForbidden, ensureSpace(ts, scoped, facile.NewID()).StatusCode)
}

func TestTokenValidationForProvision(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "prov-invalid@example.com", "password12345")

	resp := doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "provision": true}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "provision without app")
	resp = doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "space_facile_ids": []string{facile.NewID()}}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "ids without app")
	resp = doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "app": "plume"}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "app alone")
	resp = doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "app": "plume", "provision": true}, user)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func countSpaceObjects(t *testing.T, facileID string) int {
	t.Helper()
	objects, err := newTestStorageClient(t).For(storage.SpaceBucket(facileID)).ListObjects(context.Background(), "")
	require.NoError(t, err)
	return len(objects)
}

func TestPublishWithSameLinkedToIsIdempotent(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "pub-idem@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	facileID, spaceID := provisionSpace(t, ts, scoped)

	first := publish(ts, scoped, spaceID, "signed.pdf", "pdf v1", "plume:document:42")
	require.Equal(t, http.StatusCreated, first.StatusCode)
	var created struct {
		ID       any    `json:"id"`
		LinkedTo string `json:"linked_to"`
	}
	parseJSON(first, &created)
	require.Equal(t, "plume:document:42", created.LinkedTo)

	second := publish(ts, scoped, spaceID, "signed.pdf", "pdf v1 again", "plume:document:42")
	require.Equal(t, http.StatusOK, second.StatusCode)
	var again struct {
		ID any `json:"id"`
	}
	parseJSON(second, &again)
	require.Equal(t, created.ID, again.ID)

	var files int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("space_id = ? AND deleted_at IS NULL", spaceID).Count(&files).Error)
	require.EqualValues(t, 1, files)
	require.Equal(t, 1, countSpaceObjects(t, facileID))

	third := publish(ts, scoped, spaceID, "other.pdf", "pdf v2", "plume:document:43")
	require.Equal(t, http.StatusCreated, third.StatusCode)
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("space_id = ? AND deleted_at IS NULL", spaceID).Count(&files).Error)
	require.EqualValues(t, 2, files)
	require.Equal(t, 2, countSpaceObjects(t, facileID))
}

func TestPublishAfterDeleteStoresANewFile(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "pub-deleted@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	_, spaceID := provisionSpace(t, ts, scoped)

	require.Equal(t, http.StatusCreated, publish(ts, scoped, spaceID, "a.pdf", "a", "plume:document:1").StatusCode)
	var file schemas.File
	require.NoError(t, ts.db.Where("space_id = ?", spaceID).First(&file).Error)
	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/files/%d", file.ID), scoped).StatusCode)
	require.Equal(t, http.StatusCreated, publish(ts, scoped, spaceID, "a.pdf", "a", "plume:document:1").StatusCode)
}

func TestConcurrentDuplicatePublishYieldsOneFile(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "pub-race@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	facileID, spaceID := provisionSpace(t, ts, scoped)

	const callers = 8
	statuses := make(chan int, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- publish(ts, scoped, spaceID, "race.pdf", "same bytes", "plume:document:race").StatusCode
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

	var files int64
	require.NoError(t, ts.db.Model(&schemas.File{}).Where("space_id = ?", spaceID).Count(&files).Error)
	require.EqualValues(t, 1, files)
	require.Equal(t, 1, countSpaceObjects(t, facileID), "losers delete their uploaded object")
}

func TestPublishRejectsInvalidLinkedToAndIgnoresItForUnrestricted(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "pub-invalid@example.com", "password12345")
	scoped := mintProvisionToken(t, ts, user, "plume")
	_, spaceID := provisionSpace(t, ts, scoped)

	long := make([]byte, 201)
	for i := range long {
		long[i] = 'a'
	}
	require.Equal(t, http.StatusBadRequest, publish(ts, scoped, spaceID, "a.pdf", "a", string(long)).StatusCode)
	require.Equal(t, http.StatusBadRequest, publish(ts, scoped, spaceID, "a.pdf", "a", "bad\x01key").StatusCode)

	own := createSpace(t, ts, user, "Plain")
	require.Equal(t, http.StatusCreated, publish(ts, user, own, "a.txt", "a", "k").StatusCode)
	require.Equal(t, http.StatusCreated, publish(ts, user, own, "a.txt", "b", "k").StatusCode)
}
