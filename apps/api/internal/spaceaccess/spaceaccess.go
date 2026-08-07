package spaceaccess

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Require reports whether the user is a member of the space, returning a
// Forbidden error otherwise. Every endpoint that accepts a caller-supplied
// space_id must call this before using it as a query or write predicate.
func Require(ctx context.Context, orm *gorm.DB, spaceID int64, userID int64) error {
	var member schemas.SpaceMember
	err := orm.WithContext(ctx).
		Where("space_id = ? AND user_id = ?", spaceID, userID).
		First(&member).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.Forbidden("you are not a member of this space")
		}
		return errors.Internal("failed to check space membership", err)
	}
	return nil
}

// MemberIDs lists the spaces the user belongs to.
func MemberIDs(ctx context.Context, orm *gorm.DB, userID int64) ([]int64, error) {
	var ids []int64
	if err := orm.WithContext(ctx).
		Model(&schemas.SpaceMember{}).
		Where("user_id = ?", userID).
		Pluck("space_id", &ids).Error; err != nil {
		return nil, errors.Internal("failed to list space memberships", err)
	}
	return ids, nil
}
