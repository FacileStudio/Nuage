package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoPasswordRoutesExist(t *testing.T) {
	ts := setupTestServer(t)

	for _, path := range []string{"/auth/register", "/auth/login"} {
		resp := doJSON(ts, "POST", path, map[string]string{"email": "a@example.com", "password": "password12345"}, "")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
	}
}

func TestUnauthenticatedAccess(t *testing.T) {
	ts := setupTestServer(t)

	resp := doGet(ts, "/files", "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAuthConfig(t *testing.T) {
	ts := setupTestServer(t)

	resp := doGet(ts, "/auth/config", "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]bool
	parseJSON(resp, &result)
	assert.False(t, result["oidc_enabled"])
}

func TestProfileEditsNameAndColorOnly(t *testing.T) {
	ts := setupTestServer(t)
	_, token := registerUser(ts, "me@example.com", "")

	resp := doJSON(ts, "PATCH", "/users/me", map[string]string{"name": "Camille"}, token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var me struct {
		User struct{ Name, Email, Locale string } `json:"user"`
	}
	parseJSON(resp, &me)
	assert.Equal(t, "Camille", me.User.Name)
	assert.Equal(t, "me@example.com", me.User.Email)
	assert.Equal(t, "en", me.User.Locale)

	for _, body := range []map[string]string{
		{"email": "other@example.com"},
		{"password": "another-long-password"},
		{"current_password": "x"},
	} {
		resp := doJSON(ts, "PATCH", "/users/me", body, token)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "%v", body)
	}
}
