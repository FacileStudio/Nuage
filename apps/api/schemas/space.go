package schemas

import "time"

// Space is a shared team workspace.
//
// RegistreSpaceID links the row to the space Registre owns. A linked space is
// named and staffed by Registre's snapshot alone and never edited here; its
// folders and files stay Nuage's. It is nullable so the spaces Registre has not
// answered for, and the ones another app provisioned, stay unlinked rather than
// colliding on an empty string.
type Space struct {
	ID              int64     `json:"id" gorm:"primaryKey"`
	FacileID        string    `json:"facile_id" gorm:"uniqueIndex;not null"`
	Name            string    `json:"name" gorm:"not null"`
	Description     string    `json:"description"`
	ProvisionedBy   string    `json:"-" gorm:"not null;default:''"`
	RegistreSpaceID *string   `json:"registre_space_id" gorm:"uniqueIndex"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Space) TableName() string { return "spaces" }

// SpaceMember links a user to a space with a role.
type SpaceMember struct {
	ID       int64     `json:"id" gorm:"primaryKey"`
	SpaceID  int64     `json:"space_id" gorm:"not null;uniqueIndex:idx_space_user"`
	UserID   int64     `json:"user_id" gorm:"not null;uniqueIndex:idx_space_user"`
	Role     string    `json:"role" gorm:"not null;default:'member'"`
	JoinedAt time.Time `json:"joined_at" gorm:"autoCreateTime"`
	Space    Space     `json:"space,omitempty" gorm:"foreignKey:SpaceID;constraint:OnDelete:CASCADE"`
	User     User      `json:"user,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (SpaceMember) TableName() string { return "space_members" }
