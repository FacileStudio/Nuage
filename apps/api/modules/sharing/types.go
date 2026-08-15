package sharing

// CreateShareRequest is the body used to create a public share.
type CreateShareRequest struct {
	FileID     *int64  `json:"file_id"`
	FolderID   *int64  `json:"folder_id"`
	Permission string  `json:"permission"`
	ExpiresAt  *string `json:"expires_at"`
	SpaceID    *int64  `json:"space_id"`
}

// ShareResponse is a share owned by the caller, with its resource if present.
type ShareResponse struct {
	ID         int64         `json:"id"`
	Token      string        `json:"token"`
	FileID     *int64        `json:"file_id"`
	FolderID   *int64        `json:"folder_id"`
	SharedBy   int64         `json:"shared_by"`
	Permission string        `json:"permission"`
	ExpiresAt  *string       `json:"expires_at"`
	CreatedAt  string        `json:"created_at"`
	File       *PublicFile   `json:"file,omitempty"`
	Folder     *PublicFolder `json:"folder,omitempty"`
}

// ShareListResponse is a list of the caller's shares.
type ShareListResponse struct {
	Shares []ShareResponse `json:"shares"`
}

// PublicShareResponse is a share as seen by an anonymous visitor.
type PublicShareResponse struct {
	Token      string        `json:"token"`
	Permission string        `json:"permission"`
	File       *PublicFile   `json:"file,omitempty"`
	Folder     *PublicFolder `json:"folder,omitempty"`
}

// PublicFile is a shared file as seen by an anonymous visitor.
type PublicFile struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// PublicFolder is a shared folder as seen by an anonymous visitor.
type PublicFolder struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
}

// SharedFolderContentsResponse is the contents of a shared folder.
type SharedFolderContentsResponse struct {
	Permission string         `json:"permission"`
	Files      []PublicFile   `json:"files"`
	Folders    []PublicFolder `json:"folders"`
}
