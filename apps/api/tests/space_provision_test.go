package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/require"
)

type provisioned struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Role     string `json:"role"`
}

func ensureSpace(ts *testServer, token, facileID string) *http.Response {
	return doJSON(ts, "PUT", "/spaces/by-facile-id/"+facileID, map[string]string{"name": "Contracts"}, token)
}

func mintScopedToken(t *testing.T, ts *testServer, token, app string, facileIDs ...string) string {
	t.Helper()
	resp := doJSON(ts, "POST", "/users/me/api-token", map[string]any{
		"name": app, "app": app, "space_facile_ids": facileIDs,
	}, token)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var created struct {
		Token string `json:"token"`
	}
	parseJSON(resp, &created)
	require.NotEmpty(t, created.Token)
	return created.Token
}

func TestEnsureSpaceIsIdempotent(t *testing.T) {
	ts := setupTestServer(t)
	store := newTestStorageClient(t)
	_, token := registerUser(ts, "ensure-owner@example.com", "password12345")
	facileID := facile.NewID()

	first := ensureSpace(ts, token, facileID)
	require.Equal(t, http.StatusCreated, first.StatusCode)
	var created provisioned
	parseJSON(first, &created)
	require.Equal(t, facileID, created.FacileID)
	require.Equal(t, "owner", created.Role)

	_, err := store.For(storage.SpaceBucket(facileID)).ListObjects(context.Background(), "")
	require.NoError(t, err, "provisioning creates the bucket")

	second := ensureSpace(ts, token, facileID)
	require.Equal(t, http.StatusOK, second.StatusCode)
	var again provisioned
	parseJSON(second, &again)
	require.Equal(t, created.ID, again.ID)

	var count int64
	require.NoError(t, ts.db.Model(&schemas.Space{}).Where("facile_id = ?", facileID).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestEnsureSpaceRejectsMalformedFacileID(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "ensure-bad@example.com", "password12345")

	for _, id := range []string{"fac_short", "FAC_" + "0123456789abcdef01234567", "fac_0123456789abcdef0123456g", "x"} {
		require.Equal(t, http.StatusBadRequest, ensureSpace(ts, token, id).StatusCode, id)
	}
}

func TestEnsureSpaceConcurrentCreateYieldsOneSpace(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "ensure-race@example.com", "password12345")
	facileID := facile.NewID()

	const callers = 8
	statuses := make(chan int, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- ensureSpace(ts, token, facileID).StatusCode
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

	var count, owners int64
	require.NoError(t, ts.db.Model(&schemas.Space{}).Where("facile_id = ?", facileID).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, ts.db.Model(&schemas.SpaceMember{}).Joins("JOIN spaces ON spaces.id = space_members.space_id").
		Where("spaces.facile_id = ? AND space_members.role = 'owner'", facileID).Count(&owners).Error)
	require.EqualValues(t, 1, owners)
}

func TestEnsureSpaceRefusesANonMember(t *testing.T) {
	ts := setupTestServer(t)
	_, owner := registerUser(ts, "ensure-a@example.com", "password12345")
	_, outsider := registerUser(ts, "ensure-b@example.com", "password12345")
	facileID := facile.NewID()

	require.Equal(t, http.StatusCreated, ensureSpace(ts, owner, facileID).StatusCode)
	require.Equal(t, http.StatusForbidden, ensureSpace(ts, outsider, facileID).StatusCode)

	var members int64
	require.NoError(t, ts.db.Model(&schemas.SpaceMember{}).Count(&members).Error)
	require.EqualValues(t, 1, members, "a refused caller is never added")
}

func TestRestrictedTokenIsConfinedToItsSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "scoped-user@example.com", "password12345")
	allowedID, otherID := facile.NewID(), facile.NewID()

	otherSpace := createSpace(t, ts, user, "Other")
	require.Equal(t, http.StatusCreated, uploadFile(ts, user, "personal.txt", "private", nil).StatusCode)
	var personal schemas.File
	require.NoError(t, ts.db.Where("name = ?", "personal.txt").First(&personal).Error)

	scoped := mintScopedToken(t, ts, user, "ardoise", allowedID)

	require.Equal(t, http.StatusForbidden, ensureSpace(ts, scoped, otherID).StatusCode, "cannot provision an id it was not minted for")
	resp := ensureSpace(ts, scoped, allowedID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var space provisioned
	parseJSON(resp, &space)

	require.Equal(t, http.StatusForbidden, uploadFileToSpace(ts, scoped, "x.txt", "x", otherSpace).StatusCode, "another space of the same user")
	require.Equal(t, http.StatusForbidden, uploadFile(ts, scoped, "x.txt", "x", nil).StatusCode, "personal scope")
	require.Equal(t, http.StatusForbidden, doGet(ts, "/files", scoped).StatusCode, "personal listing")
	require.Equal(t, http.StatusForbidden, doGet(ts, fmt.Sprintf("/files?space_id=%d", otherSpace), scoped).StatusCode)
	require.Equal(t, http.StatusNotFound, doGet(ts, fmt.Sprintf("/files/%d", personal.ID), scoped).StatusCode, "personal file by id")
	require.Equal(t, http.StatusNotFound, doGet(ts, fmt.Sprintf("/files/%d/download", personal.ID), scoped).StatusCode)
	require.Equal(t, http.StatusNotFound, doDelete(ts, fmt.Sprintf("/files/%d", personal.ID), scoped).StatusCode)
	require.Equal(t, http.StatusForbidden, doGet(ts, "/users/me/api-token", scoped).StatusCode, "token management")
	require.Equal(t, http.StatusForbidden, doGet(ts, "/spaces", scoped).StatusCode)
	require.Equal(t, http.StatusForbidden, doGet(ts, "/folders", scoped).StatusCode)
	require.Equal(t, http.StatusForbidden, doGet(ts, "/files/"+fmt.Sprint(personal.ID)+"/versions", scoped).StatusCode)

	init := doJSON(ts, "POST", "/files/upload/init", map[string]any{"file_name": "p.bin", "total_size": 4}, scoped)
	require.Equal(t, http.StatusForbidden, init.StatusCode, "chunked upload into personal scope")
}

func TestRestrictedTokenUploadLandsInTheSpaceBucketWithOrigin(t *testing.T) {
	ts := setupTestServer(t)
	store := newTestStorageClient(t)
	_, user := registerUser(ts, "scoped-upload@example.com", "password12345")
	facileID := facile.NewID()
	scoped := mintScopedToken(t, ts, user, "ardoise", facileID)

	var space provisioned
	parseJSON(ensureSpace(ts, scoped, facileID), &space)
	require.NotZero(t, space.ID)

	resp := uploadFileWithFields(ts, scoped, "contract.pdf", "pdf bytes", map[string]string{
		"space_id":   fmt.Sprint(space.ID),
		"origin_app": "spoofed",
	})
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var file schemas.File
	require.NoError(t, ts.db.Where("name = ?", "contract.pdf").First(&file).Error)
	require.Equal(t, "ardoise", file.OriginApp, "the token's app label wins over the form value")
	require.Equal(t, storage.SpaceBucket(facileID), file.Bucket)
	_, err := store.For(file.Bucket).StatObject(context.Background(), file.BucketKey)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/files?space_id=%d", space.ID), scoped).StatusCode)
	require.Equal(t, http.StatusOK, doGet(ts, fmt.Sprintf("/files/%d/download", file.ID), scoped).StatusCode)

	init := doJSON(ts, "POST", "/files/upload/init", map[string]any{
		"file_name": "big.bin", "total_size": 4, "space_id": space.ID, "origin_app": "spoofed",
	}, scoped)
	require.Equal(t, http.StatusCreated, init.StatusCode)
	var session schemas.UploadSession
	require.NoError(t, ts.db.Where("file_name = ?", "big.bin").First(&session).Error)
	require.Equal(t, "ardoise", session.OriginApp)
}

func TestRestrictedTokenRequiresAppAndSpaces(t *testing.T) {
	ts := setupTestServer(t)
	_, user := registerUser(ts, "scoped-invalid@example.com", "password12345")

	resp := doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "app": "ardoise"}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp = doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "app": "a", "space_facile_ids": []string{"nope"}}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp = doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "app": "Not Valid!", "space_facile_ids": []string{"fac_" + strings.Repeat("a", 24)}}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("fac_%024x", i)
	}
	resp = doJSON(ts, "POST", "/users/me/api-token", map[string]any{"name": "x", "app": "ardoise", "space_facile_ids": tooMany}, user)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSpaceWithFilesCannotBeDeleted(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "delete-blocked@example.com", "password12345")
	spaceID := createSpace(t, ts, token, "Busy")
	require.Equal(t, http.StatusCreated, uploadFileToSpace(ts, token, "a.txt", "a", spaceID).StatusCode)

	require.Equal(t, http.StatusConflict, doDelete(ts, fmt.Sprintf("/spaces/%d", spaceID), token).StatusCode)
}
