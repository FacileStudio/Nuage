package spaces

import (
	"context"
	stderrors "errors"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	portespaces "github.com/FacileStudio/porte/spaces"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Service manages spaces and their members.
type Service struct {
	orm   *gorm.DB
	guard portespaces.Guard
}

// NewService builds a spaces Service over the given database connection.
func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm, guard: spaceaccess.NewGuard(orm)}
}

func (s *Service) createSpace(ctx context.Context, userID int64, req CreateSpaceRequest) (*schemas.Space, error) {
	if req.Name == "" {
		return nil, errors.Invalid("name is required")
	}

	space := &schemas.Space{
		FacileID:    facile.NewID(),
		Name:        req.Name,
		Description: req.Description,
	}

	err := s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(space).Error; err != nil {
			return errors.Internal("failed to create space", err)
		}

		member := &schemas.SpaceMember{
			SpaceID: space.ID,
			UserID:  userID,
			Role:    "owner",
		}
		if err := tx.Create(member).Error; err != nil {
			return errors.Internal("failed to add owner to space", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return space, nil
}

func (s *Service) listSpaces(ctx context.Context, userID int64) ([]SpaceResponse, error) {
	var members []schemas.SpaceMember
	if err := s.orm.WithContext(ctx).
		Preload("Space").
		Where("user_id = ?", userID).
		Find(&members).Error; err != nil {
		return nil, errors.Internal("failed to list spaces", err)
	}

	result := make([]SpaceResponse, 0, len(members))
	for _, m := range members {
		result = append(result, mapSpaceWithRole(m.Space, m.Role))
	}
	return result, nil
}

func (s *Service) getSpace(ctx context.Context, userID int64, spaceID int64) (*SpaceResponse, error) {
	role, err := s.getMemberRole(ctx, spaceID, userID)
	if err != nil {
		return nil, err
	}

	var space schemas.Space
	if err := s.orm.WithContext(ctx).Where("id = ?", spaceID).First(&space).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("space not found")
		}
		return nil, errors.Internal("failed to get space", err)
	}

	resp := mapSpaceWithRole(space, role)
	return &resp, nil
}

func (s *Service) updateSpace(ctx context.Context, userID int64, spaceID int64, req UpdateSpaceRequest) (*schemas.Space, error) {
	if err := s.requireRole(ctx, spaceID, userID, portespaces.RoleAdmin); err != nil {
		return nil, err
	}

	var space schemas.Space
	if err := s.orm.WithContext(ctx).Where("id = ?", spaceID).First(&space).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("space not found")
		}
		return nil, errors.Internal("failed to get space", err)
	}

	if req.Name != nil {
		if *req.Name == "" {
			return nil, errors.Invalid("name cannot be empty")
		}
		space.Name = *req.Name
	}
	if req.Description != nil {
		space.Description = *req.Description
	}

	if err := s.orm.WithContext(ctx).Save(&space).Error; err != nil {
		return nil, errors.Internal("failed to update space", err)
	}

	return &space, nil
}

func (s *Service) deleteSpace(ctx context.Context, userID int64, spaceID int64) error {
	if err := s.requireRole(ctx, spaceID, userID, portespaces.RoleOwner); err != nil {
		return err
	}

	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("space_id = ?", spaceID).Delete(&schemas.SpaceMember{}).Error; err != nil {
			return errors.Internal("failed to delete space members", err)
		}
		if err := tx.Where("id = ?", spaceID).Delete(&schemas.Space{}).Error; err != nil {
			return errors.Internal("failed to delete space", err)
		}
		return nil
	})
}

func (s *Service) listMembers(ctx context.Context, userID int64, spaceID int64) ([]MemberResponse, error) {
	if _, err := s.getMemberRole(ctx, spaceID, userID); err != nil {
		return nil, err
	}

	var members []schemas.SpaceMember
	if err := s.orm.WithContext(ctx).
		Preload("User").
		Where("space_id = ?", spaceID).
		Order("joined_at asc").
		Find(&members).Error; err != nil {
		return nil, errors.Internal("failed to list members", err)
	}

	result := make([]MemberResponse, 0, len(members))
	for _, m := range members {
		result = append(result, mapMember(m))
	}
	return result, nil
}

func (s *Service) addMember(ctx context.Context, userID int64, spaceID int64, req AddMemberRequest) (*MemberResponse, error) {
	actor, err := s.scope(ctx, spaceID, userID, portespaces.RoleAdmin)
	if err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = "member"
	}
	target, err := parseRole(role)
	if err != nil {
		return nil, err
	}
	if !s.guard.AssignableBy(actor, target) {
		return nil, errors.Forbidden("cannot grant a role above your own")
	}

	var existing schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, req.UserID).First(&existing).Error; err == nil {
		return nil, errors.Conflict("user is already a member of this space")
	}

	var targetUser schemas.User
	if err := s.orm.WithContext(ctx).Where("id = ?", req.UserID).First(&targetUser).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("user not found")
		}
		return nil, errors.Internal("failed to find user", err)
	}

	member := &schemas.SpaceMember{
		SpaceID: spaceID,
		UserID:  req.UserID,
		Role:    role,
	}

	if err := s.orm.WithContext(ctx).Create(member).Error; err != nil {
		return nil, errors.Internal("failed to add member", err)
	}

	if err := s.orm.WithContext(ctx).Preload("User").First(member, member.ID).Error; err != nil {
		return nil, errors.Internal("failed to load member", err)
	}

	resp := mapMember(*member)
	return &resp, nil
}

func (s *Service) updateMember(ctx context.Context, userID int64, spaceID int64, memberID int64, req UpdateMemberRequest) (*MemberResponse, error) {
	actor, err := s.scope(ctx, spaceID, userID, portespaces.RoleAdmin)
	if err != nil {
		return nil, err
	}

	target, err := parseRole(req.Role)
	if err != nil {
		return nil, err
	}

	var member schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("id = ? AND space_id = ?", memberID, spaceID).First(&member).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("member not found")
		}
		return nil, errors.Internal("failed to find member", err)
	}

	if member.Role == "owner" {
		return nil, errors.Forbidden("cannot change the owner's role")
	}

	if !s.guard.AssignableBy(actor, target) {
		return nil, errors.Forbidden("admins cannot promote to owner")
	}

	member.Role = string(target)
	if err := s.orm.WithContext(ctx).Save(&member).Error; err != nil {
		return nil, errors.Internal("failed to update member role", err)
	}

	if err := s.orm.WithContext(ctx).Preload("User").First(&member, member.ID).Error; err != nil {
		return nil, errors.Internal("failed to load member", err)
	}

	resp := mapMember(member)
	return &resp, nil
}

func (s *Service) removeMember(ctx context.Context, userID int64, spaceID int64, memberID int64) error {
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMembers(ctx, tx, spaceID); err != nil {
			return err
		}

		locked := spaceaccess.NewGuard(tx)

		actor, err := scopeOn(ctx, locked, spaceID, userID, portespaces.RoleAdmin)
		if err != nil {
			return err
		}

		var member schemas.SpaceMember
		if err := tx.Where("id = ? AND space_id = ?", memberID, spaceID).First(&member).Error; err != nil {
			if stderrors.Is(err, gorm.ErrRecordNotFound) {
				return errors.NotFound("member not found")
			}
			return errors.Internal("failed to find member", err)
		}

		if actor.Role == portespaces.RoleAdmin && portespaces.Role(member.Role) != portespaces.RoleMember {
			return errors.Forbidden("admins cannot remove owners or other admins")
		}

		leave := locked.CanLeave(ctx, strconv.FormatInt(member.UserID, 10), strconv.FormatInt(spaceID, 10))
		if leave != nil {
			if stderrors.Is(leave, portespaces.ErrSoleOwner) {
				return errors.Conflict("cannot remove the last owner; promote another owner first")
			}
			return spaceaccess.Translate(leave)
		}

		if err := tx.Delete(&member).Error; err != nil {
			return errors.Internal("failed to remove member", err)
		}
		return nil
	})
}

func (s *Service) leaveSpace(ctx context.Context, userID int64, spaceID int64) error {
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMembers(ctx, tx, spaceID); err != nil {
			return err
		}

		leave := spaceaccess.NewGuard(tx).CanLeave(ctx, strconv.FormatInt(userID, 10), strconv.FormatInt(spaceID, 10))
		if leave != nil {
			if stderrors.Is(leave, portespaces.ErrSoleOwner) {
				return errors.Conflict("the sole owner cannot leave; transfer ownership or delete the space")
			}
			return spaceaccess.Translate(leave)
		}

		err := tx.Where("space_id = ? AND user_id = ?", spaceID, userID).Delete(&schemas.SpaceMember{}).Error
		if err != nil {
			return errors.Internal("failed to leave space", err)
		}
		return nil
	})
}

func (s *Service) scope(ctx context.Context, spaceID int64, userID int64, min portespaces.Role) (portespaces.Scope, error) {
	return scopeOn(ctx, s.guard, spaceID, userID, min)
}

// scopeOn resolves the caller's role through the given guard. Callers that act
// on the outcome inside a transaction pass a guard built on the transaction
// handle, so the role is read under the same lock the decision relies on.
func scopeOn(ctx context.Context, guard portespaces.Guard, spaceID int64, userID int64, min portespaces.Role) (portespaces.Scope, error) {
	scope, err := guard.Require(ctx, strconv.FormatInt(userID, 10), strconv.FormatInt(spaceID, 10), min)
	return scope, spaceaccess.Translate(err)
}

func (s *Service) getMemberRole(ctx context.Context, spaceID int64, userID int64) (string, error) {
	scope, err := s.guard.Resolve(ctx, strconv.FormatInt(userID, 10), strconv.FormatInt(spaceID, 10))
	if err != nil {
		return "", spaceaccess.Translate(err)
	}
	return string(scope.Role), nil
}

func (s *Service) requireRole(ctx context.Context, spaceID int64, userID int64, min portespaces.Role) error {
	_, err := s.scope(ctx, spaceID, userID, min)
	return err
}

func parseRole(role string) (portespaces.Role, error) {
	parsed := portespaces.Role(role)
	if !portespaces.Default().Valid(parsed) {
		return "", errors.Invalid("invalid role, must be one of: owner, admin, member")
	}
	return parsed, nil
}

func lockMembers(ctx context.Context, tx *gorm.DB, spaceID int64) error {
	var rows []schemas.SpaceMember
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("space_id = ?", spaceID).
		Find(&rows).Error
	if err != nil {
		return errors.Internal("failed to lock space members", err)
	}
	return nil
}

func mapSpaceWithRole(space schemas.Space, role string) SpaceResponse {
	return SpaceResponse{
		ID:          space.ID,
		FacileID:    space.FacileID,
		Name:        space.Name,
		Description: space.Description,
		Role:        role,
		CreatedAt:   space.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   space.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func mapMember(m schemas.SpaceMember) MemberResponse {
	resp := MemberResponse{
		ID:       m.ID,
		UserID:   m.UserID,
		Role:     m.Role,
		JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
	}
	if m.User.ID != 0 {
		resp.User = &MemberUser{
			ID:        m.User.ID,
			Email:     m.User.Email,
			Name:      m.User.Name,
			AvatarURL: m.User.Avatar(),
			Color:     m.User.Color,
		}
	}
	return resp
}
