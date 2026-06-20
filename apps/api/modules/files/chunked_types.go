package files

type InitUploadRequest struct {
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	TotalSize int64  `json:"total_size"`
	FolderID  *int64 `json:"folder_id"`
	OriginApp string `json:"origin_app"`
	SpaceID   *int64 `json:"space_id"`
}

type InitUploadResponse struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

type ChunkResponse struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
}

type SessionStatusResponse struct {
	SessionID      string          `json:"session_id"`
	FileName       string          `json:"file_name"`
	TotalSize      int64           `json:"total_size"`
	Status         string          `json:"status"`
	UploadedChunks []ChunkResponse `json:"uploaded_chunks"`
	ExpiresAt      string          `json:"expires_at"`
}

type CompleteUploadResponse struct {
	File FileResponse `json:"file"`
}
