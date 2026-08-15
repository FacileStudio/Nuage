package sync

// ChangesResponse is the incremental sync payload since a cursor.
type ChangesResponse struct {
	Files      ChangedItems `json:"files"`
	Folders    ChangedItems `json:"folders"`
	ServerTime string       `json:"server_time"`
}

// ChangedItems pairs the changed and deleted items of one resource type.
type ChangedItems struct {
	Changed []ItemResponse `json:"changed"`
	Deleted []DeletedItem  `json:"deleted"`
}

// DeletedItem records an item that was removed since a cursor.
type DeletedItem struct {
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	SpaceID   *int64 `json:"space_id"`
	DeletedAt string `json:"deleted_at"`
	Permanent bool   `json:"permanent"`
}

// ItemResponse is one synchronised file or folder.
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

// StateResponse is the full snapshot for an initial sync.
type StateResponse struct {
	Files      []ItemResponse `json:"files"`
	Folders    []ItemResponse `json:"folders"`
	ServerTime string         `json:"server_time"`
}
