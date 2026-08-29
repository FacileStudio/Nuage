package spaceaccess

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/porte/spaces"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Require reports whether the user is a member of the space, returning a
// Forbidden error otherwise. Every endpoint that accepts a caller-supplied
// space_id must call this before using it as a query or write predicate.
//
// It takes a space id and not a pointer: a request with no space_id is
// personal scope, which is the caller's own `spaceID == nil` branch and never
// reaches the guard.
func Require(ctx context.Context, orm *gorm.DB, spaceID int64, userID int64) error {
	_, err := NewGuard(orm).Resolve(ctx, strconv.FormatInt(userID, 10), strconv.FormatInt(spaceID, 10))
	return Translate(err)
}

// MemberIDs lists the spaces the user belongs to.
func MemberIDs(ctx context.Context, orm *gorm.DB, userID int64) ([]int64, error) {
	members, err := NewGuard(orm).Spaces(ctx, strconv.FormatInt(userID, 10))
	if err != nil {
		return nil, Translate(err)
	}

	ids := make([]int64, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseInt(member.SpaceID, 10, 64)
		if err != nil {
			return nil, errors.Internal("failed to list space memberships", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Translate maps the guard's refusals onto Nuage's error envelope. A caller
// that needs its own wording for one of them tests that sentinel first and
// falls through to here for the rest.
func Translate(err error) error {
	switch {
	case err == nil:
		return nil
	case stderrors.Is(err, spaces.ErrNotMember):
		return errors.Forbidden("you are not a member of this space")
	case stderrors.Is(err, spaces.ErrForbidden):
		return errors.Forbidden("insufficient permissions")
	case stderrors.Is(err, spaces.ErrSoleOwner):
		return errors.Conflict("the space would be left without an owner")
	case stderrors.Is(err, spaces.ErrUnknownRole):
		return errors.Internal("unknown space role", err)
	default:
		return errors.Internal("failed to check space membership", err)
	}
}
