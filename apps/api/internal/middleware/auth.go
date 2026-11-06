package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/tokenscope"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
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
	ActorAccess(ctx context.Context, clientID string) (app *schemas.RegisteredApp, fullAccess bool, err error)
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
//
// A delegated JWT whose actor (act.sub, set by Registre to the client that
// performed the exchange) is a registered app is that app acting for the user.
// It reaches the app folder routes and nothing else, and nothing else reaches
// those routes.
func RequireAuth(auth Authenticator) func(http.Handler) http.Handler {
	hydrate := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			authContext, err := hydrated(request, auth)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, request.WithContext(authContext))
		})
	}
	return func(next http.Handler) http.Handler {
		return auth.RequireAuth(hydrate(next))
	}
}

// hydrated checks what the credential porte verified may do on this request
// and returns the context carrying the identity, the token scope and the
// acting app.
func hydrated(request *http.Request, auth Authenticator) (context.Context, error) {
	ctx := request.Context()
	authenticated, ok := porte.From(ctx)
	if !ok {
		return nil, errors.Unauthorized("missing auth")
	}
	scope, err := auth.ScopeForSession(ctx, authenticated.SessionID)
	if err != nil {
		return nil, err
	}
	app, err := actingApp(ctx, auth, authenticated)
	if err != nil {
		return nil, err
	}
	if err := permitted(authenticated, scope, app, request.Method, request.URL.Path); err != nil {
		return nil, err
	}
	identity, err := identityOf(ctx, auth, authenticated, scope)
	if err != nil {
		return nil, err
	}
	ctx = authcontext.WithIdentity(ctx, identity)
	if scope != nil {
		ctx = tokenscope.With(ctx, scope)
	}
	if app != nil {
		ctx = appregistry.With(ctx, app)
	}
	return ctx, nil
}

// identityOf is the profile behind the credential. Only the person's own
// unrestricted session carries the admin role.
func identityOf(ctx context.Context, auth Authenticator, authenticated porte.Identity, scope *tokenscope.Scope) (authcontext.Identity, error) {
	userID, email, isAdmin, err := auth.IdentityForUser(ctx, authenticated.UserID)
	if err != nil {
		return authcontext.Identity{}, err
	}
	return authcontext.Identity{
		UserID:  userID,
		Email:   email,
		IsAdmin: isAdmin && scope == nil && authenticated.SessionID != 0,
	}, nil
}

// permitted refuses a restricted API token outside its allowlist, a
// delegated JWT on the routes that manage credentials, a registered app
// outside its app folder and anyone else inside it.
func permitted(authenticated porte.Identity, scope *tokenscope.Scope, app *schemas.RegisteredApp, method, path string) error {
	if authenticated.SessionID == 0 && managesCredentials(method, path) {
		return errors.Forbidden("a delegated token cannot manage credentials")
	}
	if inApp := isAppRoute(path); inApp != (app != nil) {
		if inApp {
			return errors.Forbidden("only a registered app acting for a user can use app folders")
		}
		return errors.Forbidden("a registered app can only reach its app folder")
	}
	if scope != nil && !tokenscope.Permits(method, path) {
		return errors.Forbidden("this token cannot call this endpoint")
	}
	return nil
}

// managesCredentials reports whether the route mints, lists or revokes API
// tokens, edits the profile that holds the sign-in password and address,
// offers, withdraws or accepts who pays for a space, or deletes an app's data.
// Billing and deleting app data are consent, and consent comes only from the
// person's own session.
func managesCredentials(method, path string) bool {
	path = strings.TrimSuffix(strings.TrimPrefix(path, "/api"), "/")
	switch {
	case path == "/users/me":
		return method != http.MethodGet
	case strings.HasPrefix(path, "/app-data"):
		return method != http.MethodGet
	}
	return strings.HasPrefix(path, "/users/me/api-token") || isBilling(path)
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
