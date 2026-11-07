package auth

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/tokenscope"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Service is what is left of Nuage's authentication after porte took the
// credential: the profile lookup the rest of the app reads and the token scope
// bookkeeping. People sign in through Registre only.
type Service struct {
	orm        *gorm.DB
	sessions   *session.Manager
	logger     *slog.Logger
	fullAccess map[string]bool
}

// NewService builds the auth Service over the given database connection and
// session manager.
func NewService(orm *gorm.DB, sessions *session.Manager, logger *slog.Logger) *Service {
	return &Service{orm: orm, sessions: sessions, logger: logger}
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

// Issue mints a named API token: a porte session with a label and no expiry,
// which is what the separate api_tokens table used to be.
func (service *Service) Issue(ctx context.Context, userID int64, label string) (string, porte.Session, error) {
	return service.sessions.Issue(ctx, userID, label)
}

// AuthenticateRequest resolves the caller of a route that is not mounted
// behind RequireAuth, which today is WebDAV alone. It accepts only a
// credential a person holds: a session or an unrestricted API token. A bearer
// JWT (porte SessionID 0) is another service acting for the user and is
// refused, because RequireAuth's confinement of delegated tokens does not run
// here; so is a restricted app token.
func (service *Service) AuthenticateRequest(w http.ResponseWriter, r *http.Request) (int64, error) {
	identity, err := service.sessions.Authenticate(w, r)
	if err != nil {
		return 0, err
	}
	if identity.SessionID == 0 {
		return 0, errors.Forbidden("a delegated token cannot be used here")
	}
	scope, err := service.ScopeForSession(r.Context(), identity.SessionID)
	if err != nil {
		return 0, err
	}
	if scope != nil {
		return 0, errors.Forbidden("this token cannot be used here")
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

// ScopeForSession returns the restriction attached to a session, or nil when
// the session is an ordinary login or an unrestricted personal token. The
// space ids are resolved now, so a space provisioned after the token was
// minted becomes reachable without touching the token.
func (service *Service) ScopeForSession(ctx context.Context, sessionID int64) (*tokenscope.Scope, error) {
	var row schemas.TokenScope
	err := service.orm.WithContext(ctx).Where("session_id = ?", sessionID).Limit(1).Find(&row).Error
	if err != nil {
		return nil, errors.Internal("failed to read the token scope", err)
	}
	if row.SessionID == 0 {
		return nil, nil
	}

	scope := &tokenscope.Scope{App: row.App, Provision: row.Provision, FacileIDs: row.SpaceFacileIDs}
	query := service.orm.WithContext(ctx).Model(&schemas.Space{})
	provisioned := row.Provision && row.App != ""
	switch {
	case len(row.SpaceFacileIDs) > 0 && provisioned:
		query = query.Where("facile_id IN ? OR provisioned_by = ?", row.SpaceFacileIDs, row.App)
	case len(row.SpaceFacileIDs) > 0:
		query = query.Where("facile_id IN ?", row.SpaceFacileIDs)
	case provisioned:
		query = query.Where("provisioned_by = ?", row.App)
	default:
		return scope, nil
	}
	if err := query.Pluck("id", &scope.SpaceIDs).Error; err != nil {
		return nil, errors.Internal("failed to resolve the token spaces", err)
	}
	return scope, nil
}

// SaveScope restricts an issued token to the given spaces and stamps its app.
// With provision the token also reaches the spaces its app label creates.
func (service *Service) SaveScope(ctx context.Context, sessionID int64, app string, provision bool, spaceFacileIDs []string) error {
	row := schemas.TokenScope{SessionID: sessionID, App: app, Provision: provision, SpaceFacileIDs: spaceFacileIDs}
	if err := service.orm.WithContext(ctx).Create(&row).Error; err != nil {
		return errors.Internal("failed to save the token scope", err)
	}
	return nil
}

// WithFullAccessActors lets delegated JWTs exchanged by these Registre client
// ids act with the user's full access (NUAGE_FULL_ACCESS_ACTORS), and returns
// the Service. Every other actor that is not a registered app is refused.
func (service *Service) WithFullAccessActors(clientIDs []string) *Service {
	service.fullAccess = make(map[string]bool, len(clientIDs))
	for _, id := range clientIDs {
		service.fullAccess[id] = true
	}
	return service
}

// ActorAccess resolves the client a delegated JWT was exchanged by: the
// active registered app it is, or whether NUAGE_FULL_ACCESS_ACTORS lists it.
// Neither means the actor may not act at all.
func (service *Service) ActorAccess(ctx context.Context, clientID string) (*schemas.RegisteredApp, bool, error) {
	app, err := appregistry.ForClient(ctx, service.orm, clientID)
	if err != nil {
		return nil, false, err
	}
	return app, service.fullAccess[clientID], nil
}
