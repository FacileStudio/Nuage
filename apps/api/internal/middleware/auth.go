package middleware

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
)

type Authenticator interface {
	Authenticate(context context.Context, authorization string) (string, any, error)
}

func RequireAuth(authService Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			authorization := request.Header.Get("Authorization")
			if authorization == "" {
				if token := request.URL.Query().Get("token"); token != "" {
					authorization = "Bearer " + token
				}
			}

			userID, rawData, err := authService.Authenticate(request.Context(), authorization)
			if err != nil {
				httpjson.WriteError(w, err)
				return
			}
			data, ok := rawData.(interface{ GetEmail() string })
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}
			if data == nil {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}

			identity := authcontext.Identity{
				UserID: userID,
				Email:  data.GetEmail(),
			}
			if admin, ok := rawData.(interface{ GetIsAdmin() bool }); ok {
				identity.IsAdmin = admin.GetIsAdmin()
			}

			authContext := authcontext.WithIdentity(request.Context(), identity)
			next.ServeHTTP(w, request.WithContext(authContext))
		})
	}
}

// RequireAdmin rejects authenticated callers that are not administrators. It
// must be mounted after RequireAuth.
func RequireAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			identity, ok := authcontext.IdentityFromContext(request.Context())
			if !ok {
				httpjson.WriteError(w, errors.Unauthorized("missing auth"))
				return
			}
			if !identity.IsAdmin {
				httpjson.WriteError(w, errors.Forbidden("admin access required"))
				return
			}
			next.ServeHTTP(w, request)
		})
	}
}
