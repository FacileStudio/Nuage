package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	ts := setupTestServer(t)

	resp := doJSON(ts, "POST", "/auth/register", map[string]string{
		"email": "test@example.com", "password": "password12345",
	}, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result struct {
		UserID string `json:"user_id"`
		Token  string `json:"token"`
	}
	parseJSON(resp, &result)
	assert.NotEmpty(t, result.UserID)
	assert.NotEmpty(t, result.Token)
}

func TestRegisterDuplicate(t *testing.T) {
	ts := setupTestServer(t)

	registerUser(ts, "dupe@example.com", "password12345")

	resp := doJSON(ts, "POST", "/auth/register", map[string]string{
		"email": "dupe@example.com", "password": "password12345",
	}, "")
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestLogin(t *testing.T) {
	ts := setupTestServer(t)
	registerUser(ts, "login@example.com", "password12345")

	resp := doJSON(ts, "POST", "/auth/login", map[string]string{
		"email": "login@example.com", "password": "password12345",
	}, "")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result struct {
		Token string `json:"token"`
	}
	parseJSON(resp, &result)
	assert.NotEmpty(t, result.Token)
}

func TestLoginWrongPassword(t *testing.T) {
	ts := setupTestServer(t)
	registerUser(ts, "wrong@example.com", "password12345")

	resp := doJSON(ts, "POST", "/auth/login", map[string]string{
		"email": "wrong@example.com", "password": "wrongpassword",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
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
	assert.False(t, result["sso_only"])
}
