package settings

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/settings", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Get("/", handler.list)
		r.Put("/", handler.update)
		r.Post("/test-nook", handler.testNook)
	})
}
