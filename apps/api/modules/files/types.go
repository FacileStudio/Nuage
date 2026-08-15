package files

// FileResponse is one file as returned by the API.
type FileResponse struct {
	ID         int64  `json:"id"`
	FacileID   string `json:"facile_id"`
	Name       string `json:"name"`
	MimeType   string `json:"mime_type"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
	FolderID   *int64 `json:"folder_id"`
	OriginApp  string `json:"origin_app"`
	LinkedTo   string `json:"linked_to"`
	UploadedBy int64  `json:"uploaded_by"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// FileListResponse is a list of files.
type FileListResponse struct {
	Files []FileResponse `json:"files"`
}

// UpdateFileRequest is the body used to rename or move a file.
type UpdateFileRequest struct {
	Name     *string `json:"name"`
	FolderID *int64  `json:"folder_id"`
}

// LinkFileRequest is the body used to attach an external link to a file.
type LinkFileRequest struct {
	LinkedTo string `json:"linked_to"`
}

// FolderResponse is one folder as returned by the API.
type FolderResponse struct {
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ParentID  *int64 `json:"parent_id"`
	OwnerID   int64  `json:"owner_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// FolderDetailResponse is a folder with its direct files and subfolders.
type FolderDetailResponse struct {
	Folder  FolderResponse   `json:"folder"`
	Files   []FileResponse   `json:"files"`
	Folders []FolderResponse `json:"folders"`
}

// FolderListResponse is a list of folders.
type FolderListResponse struct {
	Folders []FolderResponse `json:"folders"`
}

// CreateFolderRequest is the body used to create a folder.
type CreateFolderRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
	SpaceID  *int64 `json:"space_id"`
}

// UpdateFolderRequest is the body used to rename or move a folder.
type UpdateFolderRequest struct {
	Name     *string `json:"name"`
	ParentID *int64  `json:"parent_id"`
}

// PresignRequest is the body used to request a presigned download link.
type PresignRequest struct {
	ExpiresIn *int64 `json:"expires_in"`
}

// PresignResponse is a presigned download link and its expiry.
type PresignResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}
