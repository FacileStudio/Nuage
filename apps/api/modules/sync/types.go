package sync

type ChangesResponse struct {
	Files      ChangedItems `json:"files"`
	Folders    ChangedItems `json:"folders"`
	ServerTime string       `json:"server_time"`
}

type ChangedItems struct {
	Changed []ItemResponse `json:"changed"`
	Deleted []DeletedItem  `json:"deleted"`
}

type DeletedItem struct {
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	SpaceID   *int64 `json:"space_id"`
	DeletedAt string `json:"deleted_at"`
	Permanent bool   `json:"permanent"`
}

type ItemResponse struct {
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Hash      string `json:"hash,omitempty"`
	FolderID  *int64 `json:"folder_id"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	SpaceID   *int64 `json:"space_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type StateResponse struct {
	Files      []ItemResponse `json:"files"`
	Folders    []ItemResponse `json:"folders"`
	ServerTime string         `json:"server_time"`
}
