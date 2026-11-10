// Package ideas serves Bulle's Ideas API to Nuage's own client.
//
// The routes proxy Bulle with a token exchanged for the signed-in user. The
// package decides the two things Bulle cannot know: which Registre space is
// active, taken from a space_id the user is a member of and never sent to
// Bulle as the client wrote it, and whether that space or person has ai.access.
package ideas

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/bulle"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// App is Nuage's name in Bulle's idea enum.
const App = "nuage"

type scopeKey struct{}

type scope struct {
	space    string
	aiAccess bool
}

// Gate resolves the space and the plan of every Ideas request.
type Gate struct {
	db    *gorm.DB
	plans *entitlements.Plans
	quota *quota.Service
}

// New builds the gate over the database, the plan entitlements and the quota
// service that picks a person's plan subject.
func New(db *gorm.DB, plans *entitlements.Plans, quotaService *quota.Service) *Gate {
	return &Gate{db: db, plans: plans, quota: quotaService}
}

// Mount serves /ideas on router, which must already require authentication.
// opts supplies the Bulle URL, the token source and the upgrade link; Mount
// sets the app, the space and the AI check.
func (g *Gate) Mount(router chi.Router, opts bulle.Options) error {
	opts.App = App
	opts.Space = func(r *http.Request) string { return scopeOf(r).space }
	opts.AIAccess = func(r *http.Request) bool { return scopeOf(r).aiAccess }
	router.Use(g.resolve)
	return bulle.MountIdeas(router, opts)
}

func scopeOf(r *http.Request) scope {
	resolved, _ := r.Context().Value(scopeKey{}).(scope)
	return resolved
}

func (g *Gate) resolve(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolved, err := g.scope(r)
		if err != nil {
			httpjson.WriteError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, resolved)))
	})
}

func (g *Gate) scope(r *http.Request) (scope, error) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		return scope{}, errors.Unauthorized("missing auth")
	}
	userID, err := strconv.ParseInt(identity.UserID, 10, 64)
	if err != nil {
		return scope{}, errors.Unauthorized("missing auth")
	}
	if raw := r.URL.Query().Get("space_id"); raw != "" {
		return g.spaceScope(r.Context(), raw, userID)
	}
	person, err := g.quota.PersonScope(r.Context(), userID)
	if err != nil {
		return scope{}, errors.Internal("failed to read the plan", err)
	}
	return scope{aiAccess: g.plans.AIAccess(person)}, nil
}

func (g *Gate) spaceScope(ctx context.Context, raw string, userID int64) (scope, error) {
	spaceID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || spaceID <= 0 {
		return scope{}, errors.Invalid("invalid space_id")
	}
	if err := spaceaccess.Require(ctx, g.db, spaceID, userID); err != nil {
		return scope{}, err
	}
	var registreID *string
	if err := g.db.WithContext(ctx).Table("spaces").Select("registre_space_id").Where("id = ?", spaceID).Limit(1).Scan(&registreID).Error; err != nil {
		return scope{}, errors.Internal("failed to load the space", err)
	}
	if registreID == nil || *registreID == "" {
		return scope{}, errors.Failed("this space is not linked to Registre yet, so Bulle cannot suggest ideas for it")
	}
	return scope{space: *registreID, aiAccess: g.plans.AIAccess(forfait.Scope{SpaceID: *registreID})}, nil
}
