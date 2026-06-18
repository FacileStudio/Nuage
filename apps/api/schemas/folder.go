package schemas

import "time"

type Folder struct {
	ID        int64      `json:"id" gorm:"primaryKey"`
	FacileID  string     `json:"facile_id" gorm:"uniqueIndex;not null"`
	Name      string     `json:"name" gorm:"not null"`
	ParentID  *int64     `json:"parent_id" gorm:"index"`
	OwnerID   int64      `json:"owner_id" gorm:"not null"`
	SpaceID   *int64     `json:"space_id" gorm:"index"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" gorm:"index"`
	Parent    *Folder    `json:"parent,omitempty" gorm:"foreignKey:ParentID"`
	User      User       `json:"user,omitempty" gorm:"foreignKey:OwnerID"`
	Space     *Space     `json:"space,omitempty" gorm:"foreignKey:SpaceID"`
}

func (Folder) TableName() string { return "folders" }
