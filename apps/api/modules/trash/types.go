package trash

type TrashItem struct {
	Type      string `json:"type"`
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type,omitempty"`
	Size      int64  `json:"size,omitempty"`
	DeletedAt string `json:"deleted_at"`
}

type TrashListResponse struct {
	Items []TrashItem `json:"items"`
}
