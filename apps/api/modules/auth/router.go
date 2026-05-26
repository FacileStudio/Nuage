package auth

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
	mw "github.com/FacileStudio/Nuage/apps/api/internal/middleware"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, service *Service, appEnv env.Config) {
	oidcEnabled := appEnv.OIDC != nil

	router.Route("/auth", func(router chi.Router) {
		router.Get("/config", func(w http.ResponseWriter, r *http.Request) {
			httpjson.WriteJSON(w, http.StatusOK, map[string]bool{
				"sso_only":     appEnv.SSOOnly,
				"oidc_enabled": oidcEnabled,
			})
		})

		router.With(mw.RequireAuth(service)).Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")
			if err := service.deleteSession(r.Context(), authorization); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"logged_out": true})
		})

		if !appEnv.SSOOnly {
			router.Post("/register", func(w http.ResponseWriter, request *http.Request) {
				var req RegisterRequest
				if err := httpjson.DecodeJSON(w, request, &req); err != nil {
					httpjson.WriteError(w, err)
					return
				}
				resp, err := service.controller.register(request.Context(), &req)
				if err != nil {
					httpjson.WriteError(w, err)
					return
				}
				httpjson.WriteJSON(w, http.StatusCreated, resp)
			})

			router.Post("/login", func(w http.ResponseWriter, request *http.Request) {
				var req LoginRequest
				if err := httpjson.DecodeJSON(w, request, &req); err != nil {
					httpjson.WriteError(w, err)
					return
				}
				resp, err := service.controller.login(request.Context(), &req)
				if err != nil {
					httpjson.WriteError(w, err)
					return
				}
				httpjson.WriteJSON(w, http.StatusOK, resp)
			})
		}

		if oidcEnabled {
			oidc, err := newOIDCHandler(context.Background(), appEnv.OIDC, service)
			if err != nil {
				slog.Error("failed to initialize OIDC provider", slog.Any("error", err))
			} else {
				router.Get("/oidc", oidc.login)
				router.Get("/oidc/callback", oidc.callback)
			}
		}
	})
}
