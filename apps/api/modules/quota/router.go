package quota

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the quota endpoints onto the router.
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/quota", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Get("/me", handler.getMyUsage)
		r.Post("/me/recalculate", handler.recalculate)

		r.Group(func(admin chi.Router) {
			admin.Use(middleware.RequireAdmin())
			admin.Get("/users", handler.listAllUsage)
			admin.Put("/users/{userId}", handler.setUserQuota)
		})
	})
}
