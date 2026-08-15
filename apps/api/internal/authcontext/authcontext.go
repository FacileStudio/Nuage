package authcontext

import "context"

// Identity is the authenticated caller carried on a request context.
type Identity struct {
	UserID  string
	Email   string
	IsAdmin bool
}

type contextKey struct{}

// WithIdentity attaches an Identity to the given context.
func WithIdentity(parentContext context.Context, identity Identity) context.Context {
	return context.WithValue(parentContext, contextKey{}, identity)
}

// IdentityFromContext reads the Identity attached to the context, reporting
// whether one was present.
func IdentityFromContext(parentContext context.Context) (Identity, bool) {
	identity, ok := parentContext.Value(contextKey{}).(Identity)
	return identity, ok
}
