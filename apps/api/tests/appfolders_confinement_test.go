package tests

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebDAVRefusesDelegatedTokens is the PR #5 review PoC: WebDAV never runs
// RequireAuth, so a JWT in the Basic password used to reach the whole drive.
func TestWebDAVRefusesDelegatedTokens(t *testing.T) {
	ts, verifier := setupAppServer(t)
	alice, session := registered(t, ts, "dav-jwt@example.com")
	require.Equal(t, http.StatusCreated, uploadFile(ts, session, "secret.txt", "alice personal secret", nil).StatusCode)

	for _, actor := range []string{"plume-client", "bulle", "stranger-client"} {
		token := verifier.mint(alice, actor)
		resp := davRequest(ts, "GET", "/webdav/secret.txt", token, "")
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, "%s read over WebDAV: %q", actor, body)
		assert.NotContains(t, string(body), "secret")
		assert.Equal(t, http.StatusUnauthorized, davRequest(ts, "PUT", "/webdav/by-"+actor+".txt", token, "x").StatusCode, actor)
		assert.Equal(t, http.StatusUnauthorized, davRequest(ts, "PROPFIND", "/webdav/", token, "").StatusCode, actor)
	}
	assert.Equal(t, http.StatusOK, davRequest(ts, "GET", "/webdav/secret.txt", session, "").StatusCode, "a personal token still works")
}

// TestUnlistedActorsAreDeniedEverywhere checks deny by default: an actor that
// is neither an active registered app nor in NUAGE_FULL_ACCESS_ACTORS reaches
// nothing, including an app removed from NUAGE_APPS (review PoC 2).
func TestUnlistedActorsAreDeniedEverywhere(t *testing.T) {
	ts, verifier := setupAppServer(t)
	alice, session := registered(t, ts, "unlisted@example.com")
	require.Equal(t, http.StatusCreated, uploadFile(ts, session, "secret.txt", "alice personal secret", nil).StatusCode)
	plume := verifier.mint(alice, "plume-client")
	stranger := verifier.mint(alice, "stranger-client")
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "a", "x").StatusCode)

	require.NoError(t, appregistry.Sync(context.Background(), ts.db, testApps[1:]))
	for name, token := range map[string]string{"deregistered app": plume, "unregistered actor": stranger} {
		for _, path := range []string{"/files", "/spaces", "/app/objects", "/app-data", "/users/me"} {
			assert.Equal(t, http.StatusForbidden, doGet(ts, path, token).StatusCode, "%s on %s", name, path)
		}
		assert.Equal(t, http.StatusForbidden, putObject(ts, token, "b", "x").StatusCode, name)
	}
	assert.Equal(t, http.StatusOK, doGet(ts, "/files", verifier.mint(alice, "bulle")).StatusCode, "an allowlisted actor keeps full access")
}

func TestFullAccessActorsMustNotBeApps(t *testing.T) {
	if _, err := appregistry.ParseActors("bulle, plume-client", testApps); err == nil {
		t.Fatal("a registered app was also granted full access")
	}
	actors, err := appregistry.ParseActors(" bulle ;", testApps)
	require.NoError(t, err)
	assert.Equal(t, []string{"bulle"}, actors)
}
