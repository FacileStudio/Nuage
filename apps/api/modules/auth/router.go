package auth

import (
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the routes porte owns: /auth/config, /auth/logout,
// /auth/sync-profile and the whole OIDC flow. Nuage has no password route of
// its own; people sign in through Registre. A nil kit leaves the OIDC flow
// unmounted.
func RegisterRoutes(router chi.Router, sessions *session.Manager, kit *oidc.Kit) {
	sessions.Mount(router)
	if kit != nil {
		kit.Mount(router)
	}
}
