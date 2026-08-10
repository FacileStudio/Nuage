package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service is what is left of Nuage's authentication after porte took the
// credential: the profile lookup the rest of the app reads, and a thin wrapper
// over porte/local so the register and login routes keep their response shape.
type Service struct {
	orm        *gorm.DB
	sessions   *session.Manager
	passwords  *local.Kit
	logger     *slog.Logger
	controller *Controller
}

func NewService(orm *gorm.DB, sessions *session.Manager, passwords *local.Kit, logger *slog.Logger) *Service {
	service := &Service{orm: orm, sessions: sessions, passwords: passwords, logger: logger}
	service.controller = newController(service)
	return service
}

// RequireAuth is porte's session middleware, re-exported so the module routers
// keep passing this one service to middleware.RequireAuth.
func (service *Service) RequireAuth(next http.Handler) http.Handler {
	return service.sessions.RequireAuth(next)
}

// IdentityForUser turns the user id porte authenticated into the identity the
// rest of Nuage reads. It is no longer where authentication happens.
//
// porte deliberately carries neither the email nor any role: what a role may
// do is the app's business, and the profile lives in the app's table. So the
// address is looked up here, which costs the one query the old join cost.
func (service *Service) IdentityForUser(ctx context.Context, userID int64) (string, string, bool, error) {
	var out struct {
		ID      int64
		Email   string
		IsAdmin bool
	}
	err := service.orm.WithContext(ctx).
		Model(&schemas.User{}).
		Select("id", "email", "is_admin").
		Where("id = ?", userID).
		Scan(&out).Error
	if err != nil {
		return "", "", false, errors.Internal("failed to load the account", err)
	}
	if out.ID == 0 {
		// The session outlived the user. porte's foreign key cascades a
		// delete, so this is a race, and it is still not authenticated.
		return "", "", false, errors.Unauthorized("invalid auth token")
	}
	return strconv.FormatInt(out.ID, 10), out.Email, out.IsAdmin, nil
}

// RevokeBrowserSessions ends every login a user holds and spares their named
// API tokens.
//
// Changing a password signs the other browsers out — that is what this app did
// and it is the point of the rule. It is not porte's RevokeUser, which takes
// everything: before porte the tokens lived in their own table and were
// untouched by a DELETE on sessions, so taking them now would silently break
// whatever script holds one on the day somebody rotates their password.
func (service *Service) RevokeBrowserSessions(ctx context.Context, userID int64) error {
	held, err := service.sessions.List(ctx, userID)
	if err != nil {
		return errors.Internal("failed to read the sessions", err)
	}
	for _, candidate := range held {
		if candidate.Label != "" {
			continue
		}
		if err := service.sessions.Revoke(ctx, userID, candidate.ID); err != nil {
			return errors.Internal("failed to revoke a session", err)
		}
	}
	return nil
}

// Register creates an account through porte/local and signs it in. The cookie
// is set on the way out and the token comes back in the body, so one call
// serves the browser and anything holding the old {user_id, token} shape.
func (service *Service) Register(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (string, string, error) {
	userID, token, err := service.passwords.Register(ctx, w, r, email, "", password)
	if err != nil {
		return "", "", err
	}
	return strconv.FormatInt(userID, 10), token, nil
}

func (service *Service) Login(ctx context.Context, w http.ResponseWriter, r *http.Request, email, password string) (string, string, error) {
	userID, token, err := service.passwords.Login(ctx, w, r, email, password)
	if err != nil {
		return "", "", err
	}
	return strconv.FormatInt(userID, 10), token, nil
}

// SetPassword is what PATCH /users/me calls when the body carries one.
func (service *Service) SetPassword(ctx context.Context, userID int64, email, password string) error {
	return service.passwords.SetPassword(ctx, userID, email, password)
}

// Issue mints a named API token: a porte session with a label and no expiry,
// which is what the separate api_tokens table used to be.
func (service *Service) Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error) {
	return service.sessions.Issue(ctx, userID, label)
}

// AuthenticateRequest resolves the caller of a route that is not mounted
// behind RequireAuth — the inline-image endpoint, which a browser reaches with
// an <img src> and therefore with a cookie and no header.
func (service *Service) AuthenticateRequest(w http.ResponseWriter, r *http.Request) (int64, error) {
	identity, err := service.sessions.Authenticate(w, r)
	if err != nil {
		return 0, err
	}
	return identity.UserID, nil
}

// AuthenticateToken resolves a credential this app received somewhere other
// than an Authorization header, and hands it to porte as the bearer token it
// is. WebDAV is the caller: it carries the token in the password field of HTTP
// Basic, because that is the only slot a Finder or a mount client offers.
func (service *Service) AuthenticateToken(w http.ResponseWriter, r *http.Request, token string) (int64, error) {
	bearer := r.Clone(r.Context())
	bearer.Header.Set("Authorization", "Bearer "+token)
	return service.AuthenticateRequest(w, bearer)
}

// Sessions exposes the manager for the modules that list or revoke tokens.
func (service *Service) Sessions() *session.Manager { return service.sessions }
