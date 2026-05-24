package sharing

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/shares", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Post("/", handler.create)
		r.Get("/", handler.listSharedWithMe)
		r.Get("/by-me", handler.listSharedByMe)
		r.Delete("/{id}", handler.deleteShare)
	})

	router.Get("/shared/{token}", handler.getPublic)
}
