package auth

import (
	"log/slog"
	"net/http"

	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte owns: /auth/config, /auth/logout,
// /auth/sync-profile and the whole OIDC flow. Nuage has no password route of
// its own; people sign in through Registre. A nil kit leaves the OIDC flow
// unmounted.
func RegisterRoutes(router chi.Router, sessions *session.Manager, oidcKit *oidc.Kit) error {
	return kit.Mount(router, kit.Config{Sessions: sessions, OIDC: oidcKit})
}

// RegisterInvitations mounts GET /me/invitations behind requireAuth: the
// signed-in person's pending Registre space invitations for the user menu.
// Without a Registre client (no OIDC issuer) it mounts nothing.
func RegisterInvitations(
	router chi.Router, requireAuth func(http.Handler) http.Handler, client directory.Client,
	identities kit.IdentityLister, logger *slog.Logger,
) error {
	if client.Issuer == "" {
		return nil
	}
	return kit.MountInvitations(router.With(requireAuth), kit.InvitationsConfig{
		Directory: client, Identities: identities, Logger: logger,
	})
}
