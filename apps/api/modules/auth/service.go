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

// NewService builds the auth Service over the given database connection,
// session manager and password kit.
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
//
// The session outlived the user when out.ID is zero. porte's foreign key
// cascades a delete, so this is a race, and it is still not authenticated.
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
		return "", "", false, errors.Unauthorized("invalid auth token")
	}
	return strconv.FormatInt(out.ID, 10), out.Email, out.IsAdmin, nil
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

// SetPassword gives a first password to an account that has none. porte
// answers ErrPasswordSet once there is one, because a session alone is not
// evidence enough to replace a password — that is ChangePassword.
func (service *Service) SetPassword(ctx context.Context, userID int64, password string) error {
	return service.passwords.SetPassword(ctx, userID, password)
}

// ChangePassword replaces a password after confirming the current one, and
// returns the caller's new token beside the number of other logins it ended.
//
// It takes the writer and the request because porte rotates this caller's
// session itself: the old token is dead before the call returns and the
// replacement is already in the cookie, so the screen that made the change
// keeps working. Named API tokens survive — before porte they lived in their
// own table and a password change never touched them, and taking them now
// would break somebody's script on the day they rotate.
func (service *Service) ChangePassword(ctx context.Context, w http.ResponseWriter, r *http.Request, userID int64, current, next string) (string, int64, error) {
	return service.passwords.ChangePassword(ctx, w, r, userID, current, next)
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

// VerifyPassword checks a password without issuing anything, for the settings
// screen confirming the current one before setting the next.
func (service *Service) VerifyPassword(ctx context.Context, email, password string) (int64, error) {
	return service.passwords.Verify(ctx, email, password)
}

// Sessions exposes the manager for the modules that list or revoke tokens.
func (service *Service) Sessions() *session.Manager { return service.sessions }
