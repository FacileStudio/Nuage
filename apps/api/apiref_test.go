package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FacileStudio/porte/session"

	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	activitymod "github.com/FacileStudio/Nuage/apps/api/modules/activity"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/docs"
	"github.com/FacileStudio/Nuage/apps/api/modules/files"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/modules/search"
	"github.com/FacileStudio/Nuage/apps/api/modules/settings"
	"github.com/FacileStudio/Nuage/apps/api/modules/sharing"
	"github.com/FacileStudio/Nuage/apps/api/modules/spaces"
	"github.com/FacileStudio/Nuage/apps/api/modules/sync"
	"github.com/FacileStudio/Nuage/apps/api/modules/trash"
	"github.com/FacileStudio/Nuage/apps/api/modules/users"
	nuagewebdav "github.com/FacileStudio/Nuage/apps/api/modules/webdav"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/go-chi/chi/v5"
)

func buildTestRouter() chi.Router {
	router := chi.NewRouter()
	docs.Mount(router)

	appEnv := env.Config{}
	sessions, _ := session.New(appEnv.Porte(), session.Deps{Logger: slog.Default()})
	authService := auth.NewService(nil, sessions, nil, slog.Default())

	router.Route(apiPrefix, func(r chi.Router) {
		r.Get(avatarRoutePrefix+"*", func(w http.ResponseWriter, request *http.Request) {})

		auth.RegisterRoutes(r, authService, env.Config{})
		users.RegisterRoutes(r, nil, authService)
		files.RegisterRoutes(r, nil, authService)
		trash.RegisterRoutes(r, nil, authService)
		sharing.RegisterRoutes(r, nil, authService, nil)
		settings.RegisterRoutes(r, nil, authService)
		sync.RegisterRoutes(r, nil, authService)
		quota.RegisterRoutes(r, nil, authService)
		search.RegisterRoutes(r, nil, authService)
		spaces.RegisterRoutes(r, nil, authService)
		activitymod.RegisterRoutes(r, nil, authService)
	})

	nuagewebdav.RegisterRoutes(router, nil, nil, nil, nil, nil)
	return router
}

func TestEveryRouteIsDocumented(t *testing.T) {
	router := buildTestRouter()
	if missing := apiref.Undocumented(router, docs.Reference(), "/webdav", "/avatars"); len(missing) > 0 {
		t.Errorf("routes missing from the API registry: %v", missing)
	}
}

func TestRegistryIsComplete(t *testing.T) {
	if issues := apiref.Incomplete(
		docs.Reference(),
		"/files/{id}/download",
		"/files/{id}/reupload",
		"/files",
		"/files/upload/{sessionId}/part/{partNumber}",
		"/files/upload/{sessionId}/complete",
		"/files/{id}/versions/{versionId}/restore",
		"/shared/{token}/download/{fileId}",
		"/presigned/{token}",
		"/quota/me/recalculate",
		"/settings/test-nook",
		"/trash/{type}/{id}/restore",
		"/users/me/avatar",
		"/spaces/{id}/leave",
	); len(issues) > 0 {
		t.Errorf("incomplete documentation routes: %v", issues)
	}
}

func TestReferenceIsServedAtDocs(t *testing.T) {
	router := buildTestRouter()

	page := httptest.NewRecorder()
	router.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("GET /docs = %d, want 200", page.Code)
	}

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/docs/openapi.json", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("GET /docs/openapi.json = %d, want 200", spec.Code)
	}
	var document struct {
		OpenAPI string         `json:"openapi"`
		Paths   map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(spec.Body.Bytes(), &document); err != nil {
		t.Fatalf("spec is not JSON: %v", err)
	}
	if document.OpenAPI == "" || len(document.Paths) == 0 {
		t.Fatalf("spec is empty: %+v", document)
	}
}
