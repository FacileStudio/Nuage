package files

type FileResponse struct {
	ID         int64   `json:"id"`
	FacileID   string  `json:"facile_id"`
	Name       string  `json:"name"`
	MimeType   string  `json:"mime_type"`
	Size       int64   `json:"size"`
	Hash       string  `json:"hash"`
	FolderID   *int64  `json:"folder_id"`
	OriginApp  string  `json:"origin_app"`
	LinkedTo   string  `json:"linked_to"`
	UploadedBy int64   `json:"uploaded_by"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type FileListResponse struct {
	Files []FileResponse `json:"files"`
}

type UpdateFileRequest struct {
	Name     *string `json:"name"`
	FolderID *int64  `json:"folder_id"`
}

type LinkFileRequest struct {
	LinkedTo string `json:"linked_to"`
}

type FolderResponse struct {
	ID        int64  `json:"id"`
	FacileID  string `json:"facile_id"`
	Name      string `json:"name"`
	ParentID  *int64 `json:"parent_id"`
	OwnerID   int64  `json:"owner_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type FolderDetailResponse struct {
	Folder  FolderResponse   `json:"folder"`
	Files   []FileResponse   `json:"files"`
	Folders []FolderResponse `json:"folders"`
}

type FolderListResponse struct {
	Folders []FolderResponse `json:"folders"`
}

type CreateFolderRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type UpdateFolderRequest struct {
	Name     *string `json:"name"`
	ParentID *int64  `json:"parent_id"`
}
