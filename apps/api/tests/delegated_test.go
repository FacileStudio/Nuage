package tests

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/porte"
)

const (
	delegatedRead  = "head.body.read"
	delegatedWrite = "head.body.write"
	delegatedNone  = "head.body.none"
)

type delegatedVerifier struct{ userID *int64 }

// VerifyJWT stands in for Registre's signed exchange tokens: each fixed token
// resolves to the registered user with one scope set and Bulle as the actor.
func (v delegatedVerifier) VerifyJWT(_ context.Context, raw string) (porte.Identity, error) {
	identity := porte.Identity{UserID: *v.userID, Subject: "registre-sub", Actor: "bulle"}
	switch raw {
	case delegatedRead:
		identity.Scopes = []string{"nuage.read"}
	case delegatedWrite:
		identity.Scopes = []string{"nuage.read", "nuage.write"}
	case delegatedNone:
	default:
		return porte.Identity{}, errors.New("unknown token")
	}
	return identity, nil
}

// delegatedEnv is the production wiring with OIDC_MACHINE_AUDIENCE set to
// Nuage's client id and OIDC_SCOPE_PREFIX=nuage.
func delegatedEnv() env.Config {
	return env.Config{OIDC: &env.OIDCConfig{
		Issuer:          "https://registre.example.test",
		ClientID:        "nuage-client",
		ClientSecret:    "secret",
		RedirectURL:     "https://nuage.example.test/api/auth/oidc/callback",
		SuccessURL:      "https://nuage.example.test",
		MachineAudience: "nuage-client",
		ScopePrefix:     "nuage",
	}}
}

// TestDelegatedTokenScopes checks that a delegated JWT reads with nuage.read,
// needs nuage.write to change anything, never acts as admin and cannot mint a
// lasting credential, while the user's own session token keeps working.
func TestDelegatedTokenScopes(t *testing.T) {
	userID := new(int64)
	ts := setupTestServerWith(t, delegatedEnv(), delegatedVerifier{userID: userID})

	rawID, session := registerUser(ts, "delegated@example.test", "long-password-123")
	parsed, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || session == "" {
		t.Fatalf("register: id %q, token %q", rawID, session)
	}
	*userID = parsed

	if resp := uploadFile(ts, session, "notes.txt", "hello", nil); resp.StatusCode != http.StatusCreated {
		t.Fatalf("session upload = %d, want 201", resp.StatusCode)
	}

	var listed struct {
		Files []struct {
			Name string `json:"name"`
		} `json:"files"`
	}
	resp := doGet(ts, "/files", delegatedRead)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /files with nuage.read = %d, want 200", resp.StatusCode)
	}
	parseJSON(resp, &listed)
	if len(listed.Files) != 1 || listed.Files[0].Name != "notes.txt" {
		t.Fatalf("GET /files listed %+v, want notes.txt", listed.Files)
	}

	folder := map[string]string{"name": "Bulle"}
	cases := []struct {
		name   string
		method string
		path   string
		body   any
		token  string
		want   int
	}{
		{"read scope cannot create a folder", http.MethodPost, "/folders", folder, delegatedRead, http.StatusForbidden},
		{"no scope cannot read", http.MethodGet, "/files", nil, delegatedNone, http.StatusForbidden},
		{"write scope creates a folder", http.MethodPost, "/folders", folder, delegatedWrite, http.StatusCreated},
		{"delegated token is not admin", http.MethodGet, "/quota/users", nil, delegatedWrite, http.StatusForbidden},
		{"session token stays admin", http.MethodGet, "/quota/users", nil, session, http.StatusOK},
		{"delegated token cannot mint a token", http.MethodPost, "/users/me/api-token",
			map[string]string{"name": "x"}, delegatedWrite, http.StatusForbidden},
		{"delegated token cannot edit the profile", http.MethodPatch, "/users/me",
			map[string]string{"password": "another-password-456"}, delegatedWrite, http.StatusForbidden},
		{"delegated token reads the profile", http.MethodGet, "/users/me", nil, delegatedRead, http.StatusOK},
		{"delegated token cannot offer billing", http.MethodPost, "/spaces/1/billing",
			map[string]int64{"user_id": 2}, delegatedWrite, http.StatusForbidden},
		{"delegated token cannot withdraw billing", http.MethodDelete, "/spaces/1/billing", nil, delegatedWrite, http.StatusForbidden},
		{"delegated token cannot accept billing", http.MethodPost, "/spaces/1/billing/accept", nil, delegatedWrite, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := doJSON(ts, tc.method, tc.path, tc.body, tc.token).StatusCode; got != tc.want {
				t.Fatalf("%s %s = %d, want %d", tc.method, tc.path, got, tc.want)
			}
		})
	}
}
