package schemas

import "time"

type FileVersion struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	FileID    int64     `json:"file_id" gorm:"index;not null"`
	Version   int       `json:"version" gorm:"not null"`
	BucketKey string    `json:"-" gorm:"not null"`
	Hash      string    `json:"hash"`
	Size      int64     `json:"size"`
	CreatedBy int64     `json:"created_by" gorm:"not null"`
	CreatedAt time.Time `json:"created_at"`
	File      File      `json:"-" gorm:"foreignKey:FileID"`
	User      User      `json:"-" gorm:"foreignKey:CreatedBy"`
}

func (FileVersion) TableName() string { return "file_versions" }
