package trash

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/trash", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Get("/", handler.list)
		r.Delete("/", handler.emptyTrash)
		r.Post("/{type}/{id}/restore", handler.restore)
		r.Delete("/{type}/{id}", handler.permanentDelete)
	})
}
