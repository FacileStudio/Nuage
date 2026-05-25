package quota

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/quota", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Get("/me", handler.getMyUsage)
		r.Post("/me/recalculate", handler.recalculate)
		r.Get("/users", handler.listAllUsage)
		r.Put("/users/{userId}", handler.setUserQuota)
	})
}
