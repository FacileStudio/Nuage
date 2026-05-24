package schemas

import "time"

type Share struct {
	ID         int64      `json:"id" gorm:"primaryKey"`
	Token      string     `json:"token" gorm:"uniqueIndex;not null"`
	FileID     *int64     `json:"file_id" gorm:"index"`
	FolderID   *int64     `json:"folder_id" gorm:"index"`
	SharedBy   int64      `json:"shared_by" gorm:"not null"`
	SharedWith *int64     `json:"shared_with" gorm:"index"`
	Permission string     `json:"permission" gorm:"not null;default:'view'"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	File       *File      `json:"file,omitempty" gorm:"foreignKey:FileID"`
	Folder     *Folder    `json:"folder,omitempty" gorm:"foreignKey:FolderID"`
	Owner      User       `json:"owner,omitempty" gorm:"foreignKey:SharedBy"`
}

func (Share) TableName() string { return "shares" }
