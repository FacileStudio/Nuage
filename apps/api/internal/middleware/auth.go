package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/tokenscope"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

// Authenticator is the auth service: porte's session middleware, plus the
// lookup that turns the user id porte resolved into the identity the rest of
// Nuage reads.
//
// It stays one parameter so every module router keeps calling
// middleware.RequireAuth(authService) unchanged — the seam this migration was
// designed around.
type Authenticator interface {
	RequireAuth(http.Handler) http.Handler
	IdentityForUser(ctx context.Context, userID int64) (id string, email string, isAdmin bool, err error)
	ScopeForSession(ctx context.Context, sessionID int64) (*tokenscope.Scope, error)
}

// RequireAuth runs porte's middleware and then fills in what porte
// deliberately does not carry.
//
// porte verifies the credential — cookie or bearer, one hashed session row,
// one expiry, one idle window — and hands on a user id. It holds no email,
// because a library that decided what an identity means to an app would be
// routed around by the second app that adopted it. So the profile is looked up
// here and lands in the same context every controller already reads, keeping
// authcontext.Identity.UserID a decimal string and leaving the ParseInt call
// sites downstream untouched.
//
// A bearer JWT has no session row, so its SessionID is zero. Such a token is
// another suite service acting for the user, and it never carries the admin
// role, whatever the account holds. Nor may it mint an API token, change
// the password or address the account signs in with, or move who pays for a
// space: a five-minute delegation must not turn itself into a credential that
// outlives it, nor consent on the person's behalf.
func RequireAuth(auth Authenticator) func(http.Handler) http.Handler {
	hydrate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			authenticated, ok := porte.From(request.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}
			scope, err := auth.ScopeForSession(request.Context(), authenticated.SessionID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			if err := permitted(authenticated, scope, request.Method, request.URL.Path); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			userID, email, isAdmin, err := auth.IdentityForUser(request.Context(), authenticated.UserID)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			authContext := authcontext.WithIdentity(request.Context(), authcontext.Identity{
				UserID:  userID,
				Email:   email,
				IsAdmin: isAdmin && scope == nil && authenticated.SessionID != 0,
			})
			if scope != nil {
				authContext = tokenscope.With(authContext, scope)
			}
			next.ServeHTTP(w, request.WithContext(authContext))
		})
	}
	return func(next http.Handler) http.Handler {
		return auth.RequireAuth(hydrate(next))
	}
}

// permitted refuses a restricted API token outside its allowlist and a
// delegated JWT on the routes that manage credentials.
func permitted(authenticated porte.Identity, scope *tokenscope.Scope, method, path string) error {
	if authenticated.SessionID == 0 && managesCredentials(method, path) {
		return errors.Forbidden("a delegated token cannot manage credentials")
	}
	if scope != nil && !tokenscope.Permits(method, path) {
		return errors.Forbidden("this token cannot call this endpoint")
	}
	return nil
}

// managesCredentials reports whether the route mints, lists or revokes API
// tokens, edits the profile that holds the sign-in password and address, or
// offers, withdraws or accepts who pays for a space. Billing is consent, and
// consent comes only from the person's own session.
func managesCredentials(method, path string) bool {
	path = strings.TrimSuffix(strings.TrimPrefix(path, "/api"), "/")
	if path == "/users/me" {
		return method != http.MethodGet
	}
	return strings.HasPrefix(path, "/users/me/api-token") || isBilling(path)
}

// isBilling reports whether path is a space's billing route.
func isBilling(path string) bool {
	rest, ok := strings.CutPrefix(path, "/spaces/")
	return ok && (strings.HasSuffix(rest, "/billing") || strings.HasSuffix(rest, "/billing/accept"))
}

// RequireAdmin rejects authenticated callers that are not administrators. It
// must be mounted after RequireAuth.
//
// porte carries no notion of an admin and should not: the identity provider
// assigns roles, the app decides what a role may do, and is_admin is a column
// on this app's own users table. RequireAuth reads it there and puts it in the
// context this checks.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			identity, ok := authcontext.IdentityFromContext(request.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}
			if !identity.IsAdmin {
				httpjson.WriteError(w, errors.Forbidden("admin access required"))
				return
			}
			next.ServeHTTP(w, request)
		})
	}
}
