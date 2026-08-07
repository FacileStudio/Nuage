package schemas

import "time"

// Tombstone records the permanent removal of a file or folder so that sync
// clients whose cursor predates the removal still learn about it. Rows are
// pruned once they are older than TombstoneRetention.
type Tombstone struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	ResourceType string    `json:"resource_type" gorm:"not null;index"`
	ResourceID   int64     `json:"resource_id" gorm:"not null;index"`
	FacileID     string    `json:"facile_id" gorm:"not null;index"`
	Name         string    `json:"name"`
	UserID       int64     `json:"user_id" gorm:"not null;index"`
	SpaceID      *int64    `json:"space_id" gorm:"index"`
	DeletedAt    time.Time `json:"deleted_at" gorm:"not null;index"`
}

func (Tombstone) TableName() string { return "tombstones" }

// TombstoneRetention bounds how long a permanent deletion stays observable to
// sync clients. A client offline for longer must resynchronise from /sync/state.
const TombstoneRetention = 90 * 24 * time.Hour
