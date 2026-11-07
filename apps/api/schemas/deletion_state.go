package schemas

import "time"

// DeletionStateID is the primary key of the only DirectoryDeletionState row.
const DeletionStateID int64 = 1

// DeletedUserEmail is the address of the placeholder account that stands for
// every erased person. Space files, folders and activity they authored are
// reassigned to it, so the space keeps its content with no author. The .invalid
// top-level domain cannot be registered, so no real person can hold it.
const DeletedUserEmail = "deleted-user@nuage.invalid"

// DirectoryDeletionState holds the GDPR deletion receiver's cursor and handled
// subjects as one opaque blob, rewritten after each poll so a restart neither
// replays nor skips a notice.
type DirectoryDeletionState struct {
	ID        int64     `gorm:"primaryKey"`
	State     []byte    `gorm:"type:bytea;not null"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName names the table after the porte receiver it serves.
func (DirectoryDeletionState) TableName() string { return "directory_deletion_state" }
