package appfolders

import (
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes wires the app folder routes. /app is for a registered app
// acting for a user through a delegated Registre JWT, and RequireAuth refuses
// every other caller there. /app-data is the user's own view: usage per app,
// and deleting one app's data, which only the user's own session may do.
func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	h := &handler{service: service}

	router.Route("/app", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/objects", h.list)
		r.Get("/object", h.get)
		r.Put("/object", func(w http.ResponseWriter, req *http.Request) {
			req.Body = http.MaxBytesReader(w, req.Body, MaxObjectSize)
			h.put(w, req)
		})
		r.Delete("/object", h.remove)
	})

	router.Route("/app-data", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Get("/", h.usage)
		r.Delete("/{app}", h.deleteAppData)
	})
}
