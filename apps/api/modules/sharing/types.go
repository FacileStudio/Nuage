package sharing

type CreateShareRequest struct {
	FileID     *int64  `json:"file_id"`
	FolderID   *int64  `json:"folder_id"`
	SharedWith *int64  `json:"shared_with"`
	Permission string  `json:"permission"`
	ExpiresAt  *string `json:"expires_at"`
}

type ShareResponse struct {
	ID         int64   `json:"id"`
	Token      string  `json:"token"`
	FileID     *int64  `json:"file_id"`
	FolderID   *int64  `json:"folder_id"`
	SharedBy   int64   `json:"shared_by"`
	SharedWith *int64  `json:"shared_with"`
	Permission string  `json:"permission"`
	ExpiresAt  *string `json:"expires_at"`
	CreatedAt  string  `json:"created_at"`
}

type ShareListResponse struct {
	Shares []ShareResponse `json:"shares"`
}

type PublicShareResponse struct {
	Token      string       `json:"token"`
	Permission string       `json:"permission"`
	File       *PublicFile  `json:"file,omitempty"`
	Folder     *PublicFolder `json:"folder,omitempty"`
}

type PublicFile struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type PublicFolder struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Name     string `json:"name"`
}
