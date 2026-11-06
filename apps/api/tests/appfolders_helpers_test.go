package tests

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/porte"
	"github.com/stretchr/testify/require"
)

const appPassword = "long-password-123"

// appVerifier stands in for Registre's token exchange: each minted token is a
// delegated JWT for one user, with act.sub naming the client that exchanged it.
type appVerifier struct {
	mu     *sync.Mutex
	tokens map[string]porte.Identity
}

func newAppVerifier() appVerifier {
	return appVerifier{mu: &sync.Mutex{}, tokens: map[string]porte.Identity{}}
}

// VerifyJWT resolves a token minted by mint, and refuses any other.
func (v appVerifier) VerifyJWT(_ context.Context, raw string) (porte.Identity, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	identity, ok := v.tokens[raw]
	if !ok {
		return porte.Identity{}, errors.New("unknown token")
	}
	return identity, nil
}

// mint returns a JWT-shaped token for userID exchanged by the client actor,
// carrying nuage.read and nuage.write.
func (v appVerifier) mint(userID int64, actor string) string {
	v.mu.Lock()
	defer v.mu.Unlock()
	token := fmt.Sprintf("h.%s-%d-%d.s", actor, userID, len(v.tokens))
	v.tokens[token] = porte.Identity{UserID: userID, Subject: fmt.Sprintf("sub-%d", userID), Actor: actor,
		Scopes: []string{"nuage.read", "nuage.write"}}
	return token
}

// testApps registers plume and ardoise under client ids distinct from their keys.
var testApps = []appregistry.App{
	{Key: "plume", Name: "Plume", ClientID: "plume-client"},
	{Key: "ardoise", Name: "Ardoise", ClientID: "ardoise-client"},
}

func appEnv() env.Config {
	cfg := delegatedEnv()
	cfg.Apps = testApps
	return cfg
}

func setupAppServer(t *testing.T) (*testServer, appVerifier) {
	t.Helper()
	verifier := newAppVerifier()
	return setupTestServerWith(t, appEnv(), verifier), verifier
}

func appURL(path, key string) string {
	return path + "?key=" + url.QueryEscape(key)
}

func putObject(ts *testServer, token, key, content string) *http.Response {
	req := httptest.NewRequest(http.MethodPut, appURL("/app/object", key), strings.NewReader(content))
	req.Header.Set("Content-Type", "application/pdf")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w.Result()
}

func getObject(ts *testServer, token, key string) (int, string) {
	resp := doGet(ts, appURL("/app/object", key), token)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

type objectList struct {
	Objects []struct {
		Key  string `json:"key"`
		Size int64  `json:"size"`
	} `json:"objects"`
	Next string `json:"next"`
}

func listObjects(t *testing.T, ts *testServer, token, query string) objectList {
	t.Helper()
	resp := doGet(ts, "/app/objects"+query, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list objectList
	parseJSON(resp, &list)
	return list
}
