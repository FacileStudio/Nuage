package tests

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listedEmails(t *testing.T, ts *testServer, token string) []string {
	t.Helper()
	resp := doGet(ts, "/users", token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var list struct {
		Users []struct {
			Email string `json:"email"`
		} `json:"users"`
	}
	parseJSON(resp, &list)
	emails := make([]string, 0, len(list.Users))
	for _, user := range list.Users {
		emails = append(emails, user.Email)
	}
	return emails
}

// Any Registre account can sign in, so a session that is not an
// administrator's must not read the address of someone it shares no space with.
func TestANonAdminSessionCannotListEveryUser(t *testing.T) {
	ts := setupTestServer(t)
	adminID, adminToken := registerUser(ts, "admin-visibility@example.com", "")
	_, callerToken := registerUser(ts, "caller-visibility@example.com", "")
	peerID, _ := registerUser(ts, "peer-visibility@example.com", "")
	strangerID, _ := registerUser(ts, "stranger-visibility@example.com", "")
	addSpaceMember(t, ts, callerToken, createSpace(t, ts, callerToken, "Shared"), peerID)

	assert.ElementsMatch(t, []string{"caller-visibility@example.com", "peer-visibility@example.com"}, listedEmails(t, ts, callerToken))
	for _, hidden := range []string{strangerID, adminID} {
		assert.Equal(t, http.StatusNotFound, doGet(ts, "/users/"+hidden, callerToken).StatusCode, "GET /users/%s as a non-admin", hidden)
	}
	assert.Equal(t, http.StatusOK, doGet(ts, "/users/"+peerID, callerToken).StatusCode)

	assert.Equal(t, http.StatusUnauthorized, doGet(ts, "/users", "").StatusCode)
	assert.Len(t, listedEmails(t, ts, adminToken), 4)
}

// A user_id is guessable and the member list shows the address of whoever it
// names, so a stranger is added only by exact email.
func TestAddingAStrangerByIDIsRefused(t *testing.T) {
	ts := setupTestServer(t)
	registerUser(ts, "admin-add@example.com", "")
	_, ownerToken := registerUser(ts, "owner-add@example.com", "")
	strangerID, _ := registerUser(ts, "stranger-add@example.com", "")
	spaceID := createSpace(t, ts, ownerToken, "Team")
	path := fmt.Sprintf("/spaces/%d/members", spaceID)

	id, err := strconv.ParseInt(strangerID, 10, 64)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, doJSON(ts, "POST", path, map[string]any{"user_id": id, "role": "member"}, ownerToken).StatusCode)
	assert.Equal(t, http.StatusCreated, doJSON(ts, "POST", path, map[string]any{"email": "Stranger-Add@example.com", "role": "member"}, ownerToken).StatusCode)
	assert.Equal(t, http.StatusNotFound, doJSON(ts, "POST", path, map[string]any{"email": "nobody@example.com", "role": "member"}, ownerToken).StatusCode)
}
