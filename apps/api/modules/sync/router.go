package sync

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := &Handler{service: service}

	router.Route("/sync", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/changes", handler.changes)
		r.Get("/state", handler.state)
	})
}
