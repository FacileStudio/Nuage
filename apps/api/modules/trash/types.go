package trash

// TrashItem is one trashed file or folder.
type TrashItem struct {
	Type      string `json:"type"`
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type,omitempty"`
	Size      int64  `json:"size,omitempty"`
	DeletedAt string `json:"deleted_at"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// TrashListResponse is a list of trashed items and the number of days each is
// kept before it is deleted for good (zero when nothing expires).
type TrashListResponse struct {
	Items         []TrashItem `json:"items"`
	RetentionDays int         `json:"retention_days"`
}
