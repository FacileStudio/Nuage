package webdav

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"

	"github.com/go-chi/chi/v5"
	"golang.org/x/net/webdav"
	"gorm.io/gorm"
)

func init() {
	chi.RegisterMethod("PROPFIND")
	chi.RegisterMethod("PROPPATCH")
	chi.RegisterMethod("MKCOL")
	chi.RegisterMethod("COPY")
	chi.RegisterMethod("MOVE")
	chi.RegisterMethod("LOCK")
	chi.RegisterMethod("UNLOCK")
}

func RegisterRoutes(router chi.Router, db *gorm.DB, storageClient *storage.Client, authService *auth.Service, logger *slog.Logger) {
	lockSystem := webdav.NewMemLS()

	router.Route("/webdav", func(r chi.Router) {
		r.Use(requireBasicAuth(authService))
		r.HandleFunc("/*", func(w http.ResponseWriter, req *http.Request) {
			identity, ok := authcontext.IdentityFromContext(req.Context())
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			uid, _ := strconv.ParseInt(identity.UserID, 10, 64)

			handler := &webdav.Handler{
				Prefix:     "/webdav",
				FileSystem: NewNuageFS(db, storageClient, uid),
				LockSystem: lockSystem,
				Logger: func(r *http.Request, err error) {
					if err != nil {
						logger.Error("webdav", slog.String("method", r.Method),
							slog.String("path", r.URL.Path), slog.Any("error", err))
					}
				},
			}
			handler.ServeHTTP(w, req)
		})
	})
}

type authenticator interface {
	Authenticate(ctx context.Context, authorization string) (string, any, error)
}

func requireBasicAuth(authService authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("DAV", "1, 2")
				w.Header().Set("Allow", "OPTIONS, GET, HEAD, PUT, DELETE, PROPFIND, PROPPATCH, MKCOL, MOVE, COPY, LOCK, UNLOCK")
				w.Header().Set("MS-Author-Via", "DAV")
				w.Header().Set("WWW-Authenticate", `Basic realm="Nuage WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userID, rawData, err := authService.Authenticate(r.Context(), "Bearer "+password)
			if err != nil {
				w.Header().Set("DAV", "1, 2")
				w.Header().Set("WWW-Authenticate", `Basic realm="Nuage WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			data, ok := rawData.(interface{ GetEmail() string })
			if !ok || data == nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="Nuage WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := authcontext.WithIdentity(r.Context(), authcontext.Identity{
				UserID: userID,
				Email:  data.GetEmail(),
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
