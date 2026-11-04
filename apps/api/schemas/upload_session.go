package schemas

import (
	"fmt"
	"time"
)

// UploadSession is a chunked-upload session in progress.
type UploadSession struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	FileName  string    `json:"file_name" gorm:"not null"`
	MimeType  string    `json:"mime_type"`
	FolderID  *int64    `json:"folder_id"`
	OriginApp string    `json:"origin_app"`
	UserID    int64     `json:"user_id" gorm:"index;not null"`
	TotalSize int64     `json:"total_size"`
	SpaceID   *int64    `json:"space_id"`
	LinkedTo  string    `json:"linked_to"`
	Status    string    `json:"status" gorm:"not null;default:'pending'"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (UploadSession) TableName() string { return "upload_sessions" }

// UploadChunk is one received chunk of an upload session.
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

// SessionIdleWindow is how long a pending chunked upload keeps reserving its
// declared size, and counting toward its user's open uploads, after its last
// chunk arrived (or after it started, before any chunk). A session idle for
// longer can still be resumed; its completion is checked again on its own.
const SessionIdleWindow = time.Hour

// OpenSessionClause is the SQL condition, on the upload_sessions row aliased
// alias, for a session that is pending, unexpired and active within
// SessionIdleWindow. It takes the idle cutoff as its one argument.
func OpenSessionClause(alias string) string {
	return fmt.Sprintf(`%[1]s.status = 'pending' AND %[1]s.expires_at > now()
		AND COALESCE((SELECT MAX(c.created_at) FROM upload_chunks c WHERE c.session_id = %[1]s.id), %[1]s.created_at) > ?`, alias)
}
