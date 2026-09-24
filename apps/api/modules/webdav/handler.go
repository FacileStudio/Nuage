package webdav

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
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

// davServer serves WebDAV requests for one authenticated caller across the
// personal tree, the spaces index and the per-space mounts.
type davServer struct {
	db      *gorm.DB
	storage *storage.Client
	quota   *quota.Service
	locks   *lockRegistry
	logger  *slog.Logger
}

// RegisterRoutes wires the WebDAV handler onto the router, authenticating via
// Basic credentials.
func RegisterRoutes(router chi.Router, db *gorm.DB, storageClient *storage.Client, authService *auth.Service, quotaService *quota.Service, logger *slog.Logger) {
	server := &davServer{db: db, storage: storageClient, quota: quotaService, locks: newLockRegistry(), logger: logger}

	router.Route("/webdav", func(r chi.Router) {
		r.Use(requireBasicAuth(authService))
		r.HandleFunc("/*", server.serve)
	})
}

func (s *davServer) serve(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, _ := strconv.ParseInt(identity.UserID, 10, 64)

	mount, ok := parseMountPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if needsTrailingSlashRedirect(r.URL.Path, mount) {
		http.Redirect(w, r, r.URL.Path+"/", http.StatusMovedPermanently)
		return
	}

	fs, key, err := s.mount(r.Context(), uid, mount)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	if r.Method == http.MethodPut {
		r.Body = http.MaxBytesReader(w, r.Body, maxPutBodyBytes)
	}
	if (r.Method == "MOVE" || r.Method == "COPY") && destinationEscapesMount(mount, r.Header.Get("Destination")) {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	s.handler(mount, fs, key).ServeHTTP(w, r)
}

func (s *davServer) mount(ctx context.Context, userID int64, m mountPath) (webdav.FileSystem, string, error) {
	switch m.kind {
	case mountIndex:
		return newSpacesFS(s.db, userID), lockKey(userID, m), nil
	case mountSpace:
		sc, err := resolveSpaceScope(ctx, s.db, userID, m.spaceID)
		if err != nil {
			return nil, "", err
		}
		return NewNuageFS(s.db, s.storage, s.quota, sc), lockKey(userID, m), nil
	default:
		return NewNuageFS(s.db, s.storage, s.quota, scope{userID: userID}), lockKey(userID, m), nil
	}
}

func (s *davServer) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	s.logger.Error("webdav", slog.String("method", r.Method),
		slog.String("path", r.URL.Path), slog.Any("error", err))
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

func (s *davServer) handler(m mountPath, fs webdav.FileSystem, key string) *webdav.Handler {
	return &webdav.Handler{
		Prefix:     m.prefix,
		FileSystem: fs,
		LockSystem: s.locks.forKey(key),
		Logger: func(r *http.Request, err error) {
			if err != nil {
				s.logger.Error("webdav", slog.String("method", r.Method),
					slog.String("path", r.URL.Path), slog.Any("error", err))
			}
		},
	}
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
