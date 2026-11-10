package mcp

// listInput is the input of nuage_list_folder.
type listInput struct {
	FolderID       int64  `json:"folder_id,omitempty" jsonschema:"the folder to open, from an earlier listing or search. Omit for a root"`
	SpaceID        int64  `json:"space_id,omitempty" jsonschema:"a space from the spaces list, to open that space's root. Omit, with folder_id omitted too, for the personal drive"`
	Limit          int    `json:"limit,omitempty" jsonschema:"items per page, 1 to 200. Default 100"`
	Cursor         string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"concise (default) or detailed, which adds hashes, owners and timestamps"`
}

// item is one file or folder in a listing or a search.
type item struct {
	Type      string `json:"type"`
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	SpaceID   *int64 `json:"space_id,omitempty"`
	FolderID  *int64 `json:"folder_id,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	Size      int64  `json:"size,omitempty"`
	UpdatedAt string `json:"updated_at"`
	itemDetail
}

// itemDetail is what the detailed format adds.
type itemDetail struct {
	Hash      string `json:"sha256,omitempty"`
	OwnerID   int64  `json:"owner_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// spaceSummary is one space the person belongs to.
type spaceSummary struct {
	SpaceID int64  `json:"space_id"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

// listOutput is the output of nuage_list_folder.
type listOutput struct {
	Location   string          `json:"location"`
	FolderID   *int64          `json:"folder_id,omitempty"`
	SpaceID    *int64          `json:"space_id,omitempty"`
	Items      []item          `json:"items"`
	Total      int             `json:"total"`
	NextCursor string          `json:"next_cursor,omitempty"`
	Spaces     *[]spaceSummary `json:"spaces,omitempty"`
}

// searchInput is the input of nuage_search.
type searchInput struct {
	Query          string `json:"query" jsonschema:"part of a file or folder name, case ignored, such as invoice or .pdf"`
	Type           string `json:"type,omitempty" jsonschema:"file or folder to keep only one kind. Omit for both"`
	SpaceID        int64  `json:"space_id,omitempty" jsonschema:"search one space only. Omit to search the personal drive and every space"`
	Limit          int    `json:"limit,omitempty" jsonschema:"results, 1 to 100. Default 50"`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"concise (default) or detailed"`
}

// searchOutput is the output of nuage_search.
type searchOutput struct {
	Items []item `json:"items"`
	More  bool   `json:"more"`
	Note  string `json:"note,omitempty"`
}

// readInput is the input of nuage_read_file.
type readInput struct {
	FileID   int64 `json:"file_id" jsonschema:"the file, from a listing or search"`
	Offset   int64 `json:"offset,omitempty" jsonschema:"byte offset to read from, next_offset of the previous read. Default 0"`
	MaxBytes int   `json:"max_bytes,omitempty" jsonschema:"bytes to read, 1 to 60000. Default 60000"`
}

// readOutput is the output of nuage_read_file.
type readOutput struct {
	FileID     int64  `json:"file_id"`
	Name       string `json:"name"`
	MimeType   string `json:"mime_type"`
	Size       int64  `json:"size"`
	Text       bool   `json:"text"`
	Content    string `json:"content,omitempty"`
	Offset     int64  `json:"offset"`
	NextOffset int64  `json:"next_offset,omitempty"`
	Truncated  bool   `json:"truncated"`
	Note       string `json:"note,omitempty"`
}

// saveInput is the input of nuage_save_file.
type saveInput struct {
	Name     string `json:"name,omitempty" jsonschema:"file name with its extension, such as notes.md. Required for a new file; ignored when file_id is set"`
	Content  string `json:"content" jsonschema:"the whole content: UTF-8 text, or base64 when encoding is base64. At most 5 MB once decoded"`
	Encoding string `json:"encoding,omitempty" jsonschema:"text (default) or base64 for binary content such as an image"`
	FolderID int64  `json:"folder_id,omitempty" jsonschema:"folder to save a new file in. Omit for a root"`
	SpaceID  int64  `json:"space_id,omitempty" jsonschema:"space to save a new file in, when folder_id is omitted. Omit for the personal drive"`
	FileID   int64  `json:"file_id,omitempty" jsonschema:"an existing file to overwrite; its previous content stays as a version"`
}

// writeOutput is the output of the tools that create or change one item.
type writeOutput struct {
	Status string `json:"status"`
	Item   item   `json:"item"`
	Note   string `json:"note,omitempty"`
}

// folderInput is the input of nuage_create_folder.
type folderInput struct {
	Name     string `json:"name" jsonschema:"the folder name"`
	ParentID int64  `json:"parent_id,omitempty" jsonschema:"folder to create it in. Omit for a root"`
	SpaceID  int64  `json:"space_id,omitempty" jsonschema:"space to create it in, when parent_id is omitted. Omit for the personal drive"`
}

// moveInput is the input of nuage_move.
type moveInput struct {
	Type          string `json:"type" jsonschema:"file or folder"`
	ID            int64  `json:"id" jsonschema:"the file or folder id"`
	NewName       string `json:"new_name,omitempty" jsonschema:"the new name. Omit to keep the name"`
	DestinationID *int64 `json:"destination_folder_id,omitempty" jsonschema:"folder to move into, in the same drive or space; 0 is the root. Omit to stay in place"`
}

// itemRef names one file or folder.
type itemRef struct {
	Type string `json:"type" jsonschema:"file or folder"`
	ID   int64  `json:"id" jsonschema:"the file or folder id"`
}

// shareInput is the input of nuage_create_share.
type shareInput struct {
	Type          string `json:"type" jsonschema:"file or folder"`
	ID            int64  `json:"id" jsonschema:"the file or folder id"`
	ExpiresInDays int    `json:"expires_in_days,omitempty" jsonschema:"days the link works, 1 to 365. Omit for a link without expiry"`
}

// shareRef names one link.
type shareRef struct {
	ShareID int64 `json:"share_id" jsonschema:"the link, from nuage_list_shares or nuage_create_share"`
}

// share is one public link.
type share struct {
	ShareID   int64  `json:"share_id"`
	URL       string `json:"url"`
	Type      string `json:"type"`
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	ExpiresAt string `json:"expires_at,omitempty"`
	CreatedAt string `json:"created_at"`
}

// sharesInput is the input of nuage_list_shares.
type sharesInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"links per page, 1 to 200. Default 100"`
	Cursor string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
}

// sharesOutput is the output of nuage_list_shares.
type sharesOutput struct {
	Shares     []share `json:"shares"`
	Total      int     `json:"total"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

// actionOutput is the output of the confirmed tools and of revoking.
type actionOutput struct {
	Status     string `json:"status"`
	Type       string `json:"type,omitempty"`
	ID         int64  `json:"id,omitempty"`
	ConfirmURL string `json:"confirm_url,omitempty"`
	Share      *share `json:"share,omitempty"`
	Message    string `json:"message"`
}
