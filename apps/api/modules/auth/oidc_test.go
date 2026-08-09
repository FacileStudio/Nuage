package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmailClaimTrustedRefusesOnlyAnExplicitDenial(t *testing.T) {
	cases := []struct {
		name    string
		claim   any
		trusted bool
	}{
		{"absent claim is trusted, or every pre-subject account is stranded", nil, true},
		{"verified", true, true},
		{"unverified", false, false},
		{"verified as a string", "true", true},
		{"unverified as a string", "false", false},
		{"unverified as a string, mixed case", "False", false},
		{"unexpected type", 1, false},
	}
	for _, c := range cases {
		if got := emailClaimTrusted(c.claim); got != c.trusted {
			t.Errorf("%s: emailClaimTrusted(%#v) = %v, want %v", c.name, c.claim, got, c.trusted)
		}
	}
}

func TestCallbackRejectsAMismatchedState(t *testing.T) {
	h := &oidcHandler{successURL: "https://nuage.example.com"}
	cases := []struct {
		name      string
		query     string
		wantError string
	}{
		{"mismatched state is refused before any token exchange", "sneaky", "login session expired, please try again"},
		{"matching state gets past the check", "expected-state", "identity provider rejected the login"},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state="+c.query+"&code=abc", nil)
		r.AddCookie(&http.Cookie{Name: oidcStateCookie, Value: "expected-state"})
		w := httptest.NewRecorder()
		h.callback(w, r)
		dest, err := w.Result().Location()
		if err != nil {
			t.Fatalf("%s: no redirect: %v", c.name, err)
		}
		if got := dest.Query().Get("error"); got != c.wantError {
			t.Errorf("%s: error = %q, want %q", c.name, got, c.wantError)
		}
	}
}
