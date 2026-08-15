package webdav

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"

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

const maxPutBodyBytes = 2 << 30

// RegisterRoutes wires the WebDAV handler onto the router, authenticating via
// Basic credentials.
func RegisterRoutes(router chi.Router, db *gorm.DB, storageClient *storage.Client, authService *auth.Service, quotaService *quota.Service, logger *slog.Logger) {
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

			if req.Method == http.MethodPut {
				req.Body = http.MaxBytesReader(w, req.Body, maxPutBodyBytes)
			}

			handler := &webdav.Handler{
				Prefix:     "/webdav",
				FileSystem: NewNuageFS(db, storageClient, quotaService, uid),
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

// authenticator is the auth service, narrowed to the one thing WebDAV needs.
//
// A WebDAV client re-sends its credentials on every request, so this must not
// be a login: AuthenticateToken verifies the credential and issues nothing,
// which is what keeps a Finder window from writing a session row per PROPFIND.
type authenticator interface {
	AuthenticateToken(w http.ResponseWriter, r *http.Request, token string) (int64, error)
	IdentityForUser(ctx context.Context, userID int64) (id string, email string, isAdmin bool, err error)
}

// requireBasicAuth authenticates a WebDAV request from its Basic credentials,
// accepting an API token in the password field and ignoring the username.
//
// The username is ignored and always has been: what this endpoint wants
// in the password field is an API token, and porte verifies it as the
// bearer credential it is.
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

			id, err := authService.AuthenticateToken(w, r, password)
			if err != nil {
				w.Header().Set("DAV", "1, 2")
				w.Header().Set("WWW-Authenticate", `Basic realm="Nuage WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			userID, email, isAdmin, err := authService.IdentityForUser(r.Context(), id)
			if err != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="Nuage WebDAV"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := authcontext.WithIdentity(r.Context(), authcontext.Identity{
				UserID:  userID,
				Email:   email,
				IsAdmin: isAdmin,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
