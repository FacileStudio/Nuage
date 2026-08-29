package spaceaccess

import (
	"context"
	stderrors "errors"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/spaces"

	"gorm.io/gorm"
)

// Store reads the space_members table as a porte spaces.Store. Nuage's space
// and user ids are int64 and the package deliberately owns no models, so the
// conversion to and from strings happens here and nowhere else.
type Store struct {
	orm *gorm.DB
}

// NewStore builds a Store over the given database handle. Pass a transaction
// to read the rows a guard counts under the same lock that deletes them.
func NewStore(orm *gorm.DB) *Store {
	return &Store{orm: orm}
}

// NewGuard builds the space guard Nuage runs on: this Store and the suite's
// owner/admin/member ladder.
func NewGuard(orm *gorm.DB) spaces.Guard {
	return spaces.Guard{Store: NewStore(orm)}
}

// Membership returns the user's row in one space, or spaces.ErrNotMember when
// there is none. The ids come back from the row, never from the arguments, so
// the guard's cross-check stays armed.
func (s *Store) Membership(ctx context.Context, spaceID, userID string) (spaces.Membership, error) {
	space, user, ok := pair(spaceID, userID)
	if !ok {
		return spaces.Membership{}, spaces.ErrNotMember
	}

	var member schemas.SpaceMember
	err := s.orm.WithContext(ctx).
		Where("space_id = ? AND user_id = ?", space, user).
		First(&member).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return spaces.Membership{}, spaces.ErrNotMember
		}
		return spaces.Membership{}, err
	}

	return membership(member), nil
}

// Memberships returns every space the user belongs to.
func (s *Store) Memberships(ctx context.Context, userID string) ([]spaces.Membership, error) {
	user, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return nil, nil
	}

	var members []schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("user_id = ?", user).Find(&members).Error; err != nil {
		return nil, err
	}

	out := make([]spaces.Membership, 0, len(members))
	for _, member := range members {
		out = append(out, membership(member))
	}
	return out, nil
}

// CountRole returns how many members of the space hold exactly that role.
func (s *Store) CountRole(ctx context.Context, spaceID string, role spaces.Role) (int, error) {
	space, err := strconv.ParseInt(spaceID, 10, 64)
	if err != nil {
		return 0, nil
	}

	var count int64
	err = s.orm.WithContext(ctx).
		Model(&schemas.SpaceMember{}).
		Where("space_id = ? AND role = ?", space, string(role)).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func membership(member schemas.SpaceMember) spaces.Membership {
	return spaces.Membership{
		SpaceID: strconv.FormatInt(member.SpaceID, 10),
		UserID:  strconv.FormatInt(member.UserID, 10),
		Role:    spaces.Role(member.Role),
	}
}

func pair(spaceID, userID string) (int64, int64, bool) {
	space, err := strconv.ParseInt(spaceID, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	user, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return space, user, true
}
