package schemas

import "time"

type UploadSession struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	FileName  string    `json:"file_name" gorm:"not null"`
	MimeType  string    `json:"mime_type"`
	FolderID  *int64    `json:"folder_id"`
	OriginApp string    `json:"origin_app"`
	UserID    int64     `json:"user_id" gorm:"index;not null"`
	TotalSize int64     `json:"total_size"`
	Status    string    `json:"status" gorm:"not null;default:'pending'"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (UploadSession) TableName() string { return "upload_sessions" }

type UploadChunk struct {
	ID         int64     `json:"id" gorm:"primaryKey"`
	SessionID  string    `json:"session_id" gorm:"uniqueIndex:idx_chunk_session_part;not null"`
	PartNumber int       `json:"part_number" gorm:"uniqueIndex:idx_chunk_session_part;not null"`
	BucketKey  string    `json:"-" gorm:"not null"`
	Size       int64     `json:"size"`
	Hash       string    `json:"hash"`
	CreatedAt  time.Time `json:"created_at"`
}

func (UploadChunk) TableName() string { return "upload_chunks" }
