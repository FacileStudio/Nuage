package spaces

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

var validRoles = map[string]bool{
	"owner":  true,
	"admin":  true,
	"member": true,
}

type Service struct {
	orm *gorm.DB
}

func NewService(orm *gorm.DB) *Service {
	return &Service{orm: orm}
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
	if err := s.requireRole(ctx, spaceID, userID, "owner", "admin"); err != nil {
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
	if err := s.requireRole(ctx, spaceID, userID, "owner"); err != nil {
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
	if err := s.requireRole(ctx, spaceID, userID, "owner", "admin"); err != nil {
		return nil, err
	}

	role := req.Role
	if role == "" {
		role = "member"
	}
	if !validRoles[role] {
		return nil, errors.Invalid("invalid role, must be one of: owner, admin, member")
	}
	if role == "owner" {
		return nil, errors.Invalid("cannot add another owner")
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
	if err := s.requireRole(ctx, spaceID, userID, "owner", "admin"); err != nil {
		return nil, err
	}

	if !validRoles[req.Role] {
		return nil, errors.Invalid("invalid role, must be one of: owner, admin, member")
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

	callerRole, _ := s.getMemberRole(ctx, spaceID, userID)
	if callerRole == "admin" && req.Role == "owner" {
		return nil, errors.Forbidden("admins cannot promote to owner")
	}

	member.Role = req.Role
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
	if err := s.requireRole(ctx, spaceID, userID, "owner", "admin"); err != nil {
		return err
	}

	var member schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("id = ? AND space_id = ?", memberID, spaceID).First(&member).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.NotFound("member not found")
		}
		return errors.Internal("failed to find member", err)
	}

	if member.Role == "owner" {
		return errors.Forbidden("cannot remove the owner")
	}

	callerRole, _ := s.getMemberRole(ctx, spaceID, userID)
	if callerRole == "admin" && member.Role == "admin" {
		return errors.Forbidden("admins cannot remove other admins")
	}

	if err := s.orm.WithContext(ctx).Delete(&member).Error; err != nil {
		return errors.Internal("failed to remove member", err)
	}

	return nil
}

func (s *Service) getMemberRole(ctx context.Context, spaceID int64, userID int64) (string, error) {
	var member schemas.SpaceMember
	if err := s.orm.WithContext(ctx).Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.Forbidden("you are not a member of this space")
		}
		return "", errors.Internal("failed to check membership", err)
	}
	return member.Role, nil
}

func (s *Service) requireRole(ctx context.Context, spaceID int64, userID int64, roles ...string) error {
	role, err := s.getMemberRole(ctx, spaceID, userID)
	if err != nil {
		return err
	}

	for _, r := range roles {
		if role == r {
			return nil
		}
	}

	return errors.Forbidden("insufficient permissions")
}

func (s *Service) ResolveSpaceAccess(ctx context.Context, spaceID int64, userID int64) error {
	_, err := s.getMemberRole(ctx, spaceID, userID)
	return err
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
			AvatarURL: m.User.AvatarURL,
			Color:     m.User.Color,
		}
	}
	return resp
}
