package schemas

import "time"

const (
	// TrashRetentionKey names the setting holding how many days a trashed item
	// is kept before it is deleted for good. Zero keeps it until someone empties
	// the trash.
	TrashRetentionKey = "trash_retention_days"
	// DefaultTrashRetentionDays applies while the setting has never been saved.
	DefaultTrashRetentionDays = 30
	// MaxTrashRetentionDays bounds the setting so the cutoff stays a sane date.
	MaxTrashRetentionDays = 3650
)

// Setting is one key/value application setting.
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey"`
	Value     string    `json:"value" gorm:"type:text"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Setting) TableName() string { return "settings" }
