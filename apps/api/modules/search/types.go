package search

type SearchResult struct {
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Path      string `json:"path"`
	MimeType  string `json:"mime_type,omitempty"`
	Size      int64  `json:"size"`
	FolderID  *int64 `json:"folder_id,omitempty"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	UpdatedAt string `json:"updated_at"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}
