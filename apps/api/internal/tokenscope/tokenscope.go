package tokenscope

import (
	"context"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

var facileID = regexp.MustCompile(`^fac_[0-9a-f]{24}$`)

var appLabel = regexp.MustCompile(`^[a-z0-9-]{1,32}$`)

// MaxScopedSpaces caps how many spaces one restricted token may name, because
// every request made with it resolves that list against the database.
const MaxScopedSpaces = 50

// ValidApp reports whether label is usable as a token's app label, which ends
// up stamped on every file the token uploads as its origin_app.
func ValidApp(label string) bool {
	return appLabel.MatchString(label)
}

// ValidFacileID reports whether id has the shape facile.NewID produces, which
// is also what makes it a legal part of a bucket name.
func ValidFacileID(id string) bool {
	return facileID.MatchString(id)
}

// Scope is what a restricted API token may reach. FacileIDs is what the token
// was minted for; SpaceIDs is the subset of those that exist as spaces now.
type Scope struct {
	App       string
	FacileIDs []string
	SpaceIDs  []int64
}

type contextKey struct{}

// With attaches the scope to the context.
func With(ctx context.Context, scope *Scope) context.Context {
	return context.WithValue(ctx, contextKey{}, scope)
}

// From returns the scope of the calling token, or nil for an unrestricted one.
func From(ctx context.Context) *Scope {
	scope, _ := ctx.Value(contextKey{}).(*Scope)
	return scope
}

// AllowsSpace reports whether the restricted token may reach the space.
func (s *Scope) AllowsSpace(spaceID int64) bool {
	return slices.Contains(s.SpaceIDs, spaceID)
}

// AllowsFacileID reports whether the token was minted for this space.
func (s *Scope) AllowsFacileID(id string) bool {
	return slices.Contains(s.FacileIDs, id)
}

// RequireSpace refuses personal scope and any space outside the token's set
// when the caller holds a restricted token. An unrestricted caller passes.
func RequireSpace(ctx context.Context, spaceID *int64) error {
	scope := From(ctx)
	if scope == nil {
		return nil
	}
	if spaceID == nil {
		return errors.Forbidden("this token cannot access personal files")
	}
	if !scope.AllowsSpace(*spaceID) {
		return errors.Forbidden("this token cannot access this space")
	}
	return nil
}

// Origin returns the origin_app to record: the token's app label when it has
// one, otherwise what the caller asked for.
func Origin(ctx context.Context, requested string) string {
	if scope := From(ctx); scope != nil && scope.App != "" {
		return scope.App
	}
	return requested
}

// Permits reports whether a restricted token may call the route at all. It is
// an allowlist: file upload, read and delete, chunked uploads, and the
// find-or-create of the spaces the token was minted for. Every other route,
// including token management and personal-scope features, is refused.
func Permits(method, path string) bool {
	path = strings.TrimPrefix(path, "/api")
	switch {
	case strings.HasPrefix(path, "/spaces/by-facile-id/"):
		return method == http.MethodPut
	case path == "/files" || path == "/files/":
		return method == http.MethodGet || method == http.MethodPost
	case strings.HasPrefix(path, "/files/upload/"):
		return true
	}
	rest, ok := strings.CutPrefix(path, "/files/")
	if !ok {
		return false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	switch len(parts) {
	case 1:
		return method == http.MethodGet || method == http.MethodDelete
	case 2:
		return (parts[1] == "download" && method == http.MethodGet) || (parts[1] == "presign" && method == http.MethodPost)
	}
	return false
}
