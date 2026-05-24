package schemas

import "time"

type File struct {
	ID         int64      `json:"id" gorm:"primaryKey"`
	FacileID   string     `json:"facile_id" gorm:"uniqueIndex;not null"`
	Name       string     `json:"name" gorm:"not null"`
	MimeType   string     `json:"mime_type"`
	Size       int64      `json:"size"`
	BucketKey  string     `json:"-" gorm:"not null"`
	FolderID   *int64     `json:"folder_id" gorm:"index"`
	OriginApp  string     `json:"origin_app"`
	LinkedTo   string     `json:"linked_to" gorm:"index"`
	UploadedBy int64      `json:"uploaded_by" gorm:"not null"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at" gorm:"index"`
	Folder     *Folder    `json:"folder,omitempty" gorm:"foreignKey:FolderID"`
	User       User       `json:"user,omitempty" gorm:"foreignKey:UploadedBy"`
}

func (File) TableName() string { return "files" }
