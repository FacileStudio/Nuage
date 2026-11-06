package middleware

import (
	"context"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
)

// actingApp is the registered app a delegated JWT was exchanged by, or nil.
// Only a JWT has an actor: a cookie, an API token or a restricted token never
// acts as an app, whatever label it was minted with.
func actingApp(ctx context.Context, auth Authenticator, authenticated porte.Identity) (*schemas.RegisteredApp, error) {
	if authenticated.SessionID != 0 || authenticated.Actor == "" {
		return nil, nil
	}
	app, err := auth.RegisteredApp(ctx, authenticated.Actor)
	if err != nil {
		return nil, errors.Internal("failed to resolve the acting app", err)
	}
	return app, nil
}

// isAppRoute reports whether path is one of the routes a registered app uses
// to reach the app folder of the user it acts for.
func isAppRoute(path string) bool {
	path = strings.TrimPrefix(path, "/api")
	return path == "/app" || strings.HasPrefix(path, "/app/")
}

// isBilling reports whether path is a space's billing route.
func isBilling(path string) bool {
	rest, ok := strings.CutPrefix(path, "/spaces/")
	return ok && (strings.HasSuffix(rest, "/billing") || strings.HasSuffix(rest, "/billing/accept"))
}
