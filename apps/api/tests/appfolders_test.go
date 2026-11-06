package tests

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppCannotReachAnotherAppsOrUsersFolder(t *testing.T) {
	ts, verifier := setupAppServer(t)
	alice, _ := registered(t, ts, "alice-app@example.com")
	bob, _ := registered(t, ts, "bob-app@example.com")
	plumeAlice := verifier.mint(alice, "plume-client")
	plumeBob := verifier.mint(bob, "plume-client")
	ardoiseAlice := verifier.mint(alice, "ardoise-client")

	resp := putObject(ts, plumeAlice, "contracts/a.pdf", "alice's contract")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, http.StatusOK, putObject(ts, plumeAlice, "contracts/a.pdf", "alice's contract v2").StatusCode)

	status, body := getObject(ts, plumeAlice, "contracts/a.pdf")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "alice's contract v2", body)

	status, _ = getObject(ts, plumeBob, "contracts/a.pdf")
	assert.Equal(t, http.StatusNotFound, status, "another user's folder")
	status, _ = getObject(ts, ardoiseAlice, "contracts/a.pdf")
	assert.Equal(t, http.StatusNotFound, status, "another app's folder")
	assert.Empty(t, listObjects(t, ts, plumeBob, "").Objects)
	assert.Empty(t, listObjects(t, ts, ardoiseAlice, "").Objects)
	assert.Equal(t, http.StatusNotFound, doDelete(ts, appURL("/app/object", "contracts/a.pdf"), ardoiseAlice).StatusCode)
	assert.Len(t, listObjects(t, ts, plumeAlice, "?prefix=contracts/").Objects, 1)
}

func TestRegisteredAppIsConfinedToItsFolder(t *testing.T) {
	ts, verifier := setupAppServer(t)
	alice, _ := registered(t, ts, "confined-app@example.com")
	plumeAlice := verifier.mint(alice, "plume-client")

	cases := []struct {
		name, method, path, token string
	}{
		{"app token cannot list personal files", http.MethodGet, "/files", plumeAlice},
		{"app token cannot list spaces", http.MethodGet, "/spaces", plumeAlice},
		{"app token cannot read app data", http.MethodGet, "/app-data", plumeAlice},
		{"app token cannot delete app data", http.MethodDelete, "/app-data/ardoise", plumeAlice},
		{"unregistered actor cannot use app folders", http.MethodGet, "/app/objects", verifier.mint(alice, "bulle")},
		{"the key label is not the client id", http.MethodGet, "/app/objects", verifier.mint(alice, "plume")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, http.StatusForbidden, doJSON(ts, tc.method, tc.path, nil, tc.token).StatusCode)
		})
	}

	require.NoError(t, appregistry.Sync(context.Background(), ts.db, testApps[1:]))
	assert.Equal(t, http.StatusForbidden, doGet(ts, "/app/objects", plumeAlice).StatusCode, "an app removed from NUAGE_APPS stops acting")
}

func TestAppObjectKeysAndPagesAreValidated(t *testing.T) {
	ts, verifier := setupAppServer(t)
	alice, _ := registered(t, ts, "keys-app@example.com")
	plume := verifier.mint(alice, "plume-client")

	for _, key := range []string{"", "/lead", "trail/", "a//b", "a/../b", "./a", "back\\slash", "ctl\x01", strings.Repeat("k", 513)} {
		assert.Equal(t, http.StatusBadRequest, putObject(ts, plume, key, "x").StatusCode, "key %q", key)
	}
	for i := range 3 {
		require.Equal(t, http.StatusCreated, putObject(ts, plume, fmt.Sprintf("k%d", i), "x").StatusCode)
	}
	first := listObjects(t, ts, plume, "?limit=2")
	require.Len(t, first.Objects, 2)
	require.Equal(t, "k1", first.Next)
	second := listObjects(t, ts, plume, "?limit=2&after="+first.Next)
	require.Len(t, second.Objects, 1)
	assert.Equal(t, "k2", second.Objects[0].Key)
	assert.Empty(t, second.Next)

	req := httptest.NewRequest(http.MethodPut, appURL("/app/object", "big"), bytes.NewReader(nil))
	req.ContentLength = 100<<20 + 1
	req.Header.Set("Authorization", "Bearer "+plume)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestSelfLabelledTokenCannotActAsAnApp(t *testing.T) {
	ts, _ := setupAppServer(t)
	_, session := registerUser(ts, "label-app@example.com", appPassword)
	labelled := mintProvisionToken(t, ts, session, "plume")

	for _, token := range []string{labelled, session} {
		assert.Equal(t, http.StatusForbidden, putObject(ts, token, "x", "x").StatusCode)
		assert.Equal(t, http.StatusForbidden, doGet(ts, "/app/objects", token).StatusCode)
		status, _ := getObject(ts, token, "x")
		assert.Equal(t, http.StatusForbidden, status)
	}
}

func TestAppFolderSharesThePersonalPlan(t *testing.T) {
	ts, verifier := setupAppServer(t)
	userID, session := registered(t, ts, "plan-app@example.com")
	require.NoError(t, ts.db.Exec(
		"INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)", userID, testIssuer, "sub-1").Error)
	e := planEnv{ts: ts, userID: userID, token: session}
	plume := verifier.mint(userID, "plume-client")
	ardoise := verifier.mint(userID, "ardoise-client")

	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-20)
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "a", "0123456789").StatusCode)
	require.Equal(t, http.StatusCreated, putObject(ts, ardoise, "b", "0123456789").StatusCode)

	assert.Equal(t, http.StatusPaymentRequired, putObject(ts, plume, "c", "x").StatusCode)
	assert.Equal(t, http.StatusPaymentRequired, putObject(ts, ardoise, "b", "01234567890").StatusCode, "growth past the plan")
	assert.Equal(t, http.StatusOK, putObject(ts, plume, "a", "abcdefghij").StatusCode, "same-size replacement adds nothing")
	assert.Equal(t, http.StatusPaymentRequired, uploadFile(ts, session, "mine.txt", "x", nil).StatusCode)

	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(doGet(ts, "/quota/me", session), &usage)
	assert.Equal(t, gibibyte, usage.StorageUsed, "the meter counts app folders")

	require.Equal(t, http.StatusNoContent, doDelete(ts, appURL("/app/object", "a"), plume).StatusCode)
	assert.Equal(t, http.StatusCreated, uploadFile(ts, session, "mine.txt", "x", nil).StatusCode)
}

func TestUserSeesAppDataUsage(t *testing.T) {
	ts, verifier := setupAppServer(t)
	userID, session := registered(t, ts, "data-app@example.com")
	plume := verifier.mint(userID, "plume-client")
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "one", "12345").StatusCode)
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "two", "123").StatusCode)

	var data struct {
		Apps []struct {
			App       string `json:"app"`
			Name      string `json:"name"`
			Objects   int64  `json:"objects"`
			UsedBytes int64  `json:"used_bytes"`
		} `json:"apps"`
	}
	parseJSON(doGet(ts, "/app-data", session), &data)
	require.Len(t, data.Apps, 1)
	assert.Equal(t, "plume", data.Apps[0].App)
	assert.Equal(t, "Plume", data.Apps[0].Name)
	assert.Equal(t, int64(2), data.Apps[0].Objects)
	assert.Equal(t, int64(8), data.Apps[0].UsedBytes)

	var files struct {
		Files []any `json:"files"`
	}
	parseJSON(doGet(ts, "/files", session), &files)
	assert.Empty(t, files.Files, "app objects never show in the file tree")
	var spaces struct {
		Spaces []any `json:"spaces"`
	}
	parseJSON(doGet(ts, "/spaces", session), &spaces)
	assert.Empty(t, spaces.Spaces, "an app folder is not a space")
}

func TestUserDeletesAppData(t *testing.T) {
	ts, verifier := setupAppServer(t)
	userID, session := registered(t, ts, "delete-app@example.com")
	plume := verifier.mint(userID, "plume-client")
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "one", "12345").StatusCode)
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "two", "123").StatusCode)

	var stored []string
	require.NoError(t, ts.db.Table("app_objects").Pluck("bucket_key", &stored).Error)
	require.Len(t, stored, 2)

	labelled := mintProvisionToken(t, ts, session, "plume")
	assert.Equal(t, http.StatusForbidden, doDelete(ts, "/app-data/plume", labelled).StatusCode)
	assert.Equal(t, http.StatusForbidden, doDelete(ts, "/app-data/plume", verifier.mint(userID, "bulle")).StatusCode)
	assert.Equal(t, http.StatusNotFound, doDelete(ts, "/app-data/ardoise", session).StatusCode)
	assert.Equal(t, http.StatusNoContent, doDelete(ts, "/app-data/plume", session).StatusCode)

	var data struct {
		Apps []any `json:"apps"`
	}
	parseJSON(doGet(ts, "/app-data", session), &data)
	assert.Empty(t, data.Apps)
	status, _ := getObject(ts, plume, "one")
	assert.Equal(t, http.StatusNotFound, status)
	for _, key := range stored {
		_, err := ts.store.StatObject(context.Background(), key)
		assert.Error(t, err, "stored bytes are removed")
	}
	assert.Equal(t, http.StatusCreated, putObject(ts, plume, "one", "again").StatusCode, "the app starts a fresh folder")
}
