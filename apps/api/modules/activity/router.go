package activity

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/activity", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Get("/", handler.listAll)
		r.Get("/me", handler.listMine)
		r.Get("/files/{id}", handler.forFile)
	})
}
