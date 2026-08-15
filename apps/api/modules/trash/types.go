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
}

// TrashListResponse is a list of trashed items.
type TrashListResponse struct {
	Items []TrashItem `json:"items"`
}
