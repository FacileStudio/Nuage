package files

import (
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/files", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Post("/", func(w http.ResponseWriter, req *http.Request) {
			req.Body = http.MaxBytesReader(w, req.Body, 100<<20)
			handler.upload(w, req)
		})
		r.Get("/", handler.list)
		r.Get("/{id}", handler.get)
		r.Get("/{id}/download", handler.download)
		r.Delete("/{id}", handler.deleteFile)
		r.Put("/{id}", handler.updateFile)
		r.Post("/{id}/link", handler.linkFile)
	})

	router.Route("/folders", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Post("/", handler.createFolder)
		r.Get("/", handler.listFolders)
		r.Get("/{id}", handler.getFolder)
		r.Put("/{id}", handler.updateFolder)
		r.Delete("/{id}", handler.deleteFolder)
	})
}
