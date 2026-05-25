package sharing

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service, storageClient *storage.Client) {
	handler := newHandler(service, storageClient)

	router.Route("/shares", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Post("/", handler.create)
		r.Get("/by-me", handler.listSharedByMe)
		r.Delete("/{id}", handler.deleteShare)
	})

	router.Get("/shared/{token}", handler.getPublic)
	router.Get("/shared/{token}/download/{fileId}", handler.downloadSharedFile)
	router.Get("/shared/{token}/files", handler.listSharedFolder)
}
