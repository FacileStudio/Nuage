package spaces

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, authService *auth.Service) {
	handler := newHandler(service)

	router.Route("/spaces", func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Post("/", handler.create)
		r.Get("/", handler.list)
		r.Get("/{id}", handler.get)
		r.Put("/{id}", handler.update)
		r.Delete("/{id}", handler.deleteSpace)

		r.Get("/{id}/members", handler.listMembers)
		r.Post("/{id}/members", handler.addMember)
		r.Put("/{id}/members/{memberId}", handler.updateMember)
		r.Delete("/{id}/members/{memberId}", handler.removeMember)
	})
}
