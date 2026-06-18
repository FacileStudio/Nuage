package schemas

import "time"

type Space struct {
	ID          int64     `json:"id" gorm:"primaryKey"`
	FacileID    string    `json:"facile_id" gorm:"uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (Space) TableName() string { return "spaces" }

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
