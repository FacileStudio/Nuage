package schemas

import (
	"fmt"
	"time"

	"gorm.io/gorm"
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
// declared size after it was last active: its start, or its last chunk. A lapsed session still counts toward
// its user's open uploads until it expires, and its next chunk must re-admit
// the reservation under the plan lock before it is stored.
const SessionIdleWindow = time.Hour

// ReservingSessionClause is the SQL condition, on the upload_sessions row
// aliased alias, for a session that is pending, unexpired and active within
// SessionIdleWindow. It takes the idle cutoff as its one argument.
func ReservingSessionClause(alias string) string {
	return fmt.Sprintf(`%[1]s.status = 'pending' AND %[1]s.expires_at > now() AND %[2]s > ?`, alias, lastActiveSQL(alias))
}

// lastActiveSQL is the SQL expression for when the session aliased alias was
// last active.
func lastActiveSQL(alias string) string {
	return fmt.Sprintf(`COALESCE((SELECT MAX(c.created_at) FROM upload_chunks c WHERE c.session_id = %[1]s.id), %[1]s.created_at)`, alias)
}

// SessionLapsed reports whether the session stopped reserving its size.
func SessionLapsed(db *gorm.DB, sessionID string) (bool, error) {
	var lapsed bool
	err := db.Raw(fmt.Sprintf("SELECT %s <= ? FROM upload_sessions s WHERE s.id = ?", lastActiveSQL("s")),
		time.Now().Add(-SessionIdleWindow), sessionID).Scan(&lapsed).Error
	return lapsed, err
}
