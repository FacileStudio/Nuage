package files

// InitUploadRequest is the body used to start a chunked upload.
type InitUploadRequest struct {
	FileName  string `json:"file_name"`
	MimeType  string `json:"mime_type"`
	TotalSize int64  `json:"total_size"`
	FolderID  *int64 `json:"folder_id"`
	OriginApp string `json:"origin_app"`
	SpaceID   *int64 `json:"space_id"`
}

// InitUploadResponse identifies a started chunked-upload session.
type InitUploadResponse struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

// ChunkResponse describes one uploaded chunk.
type ChunkResponse struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Hash       string `json:"hash"`
}

// SessionStatusResponse reports the progress of a chunked-upload session.
type SessionStatusResponse struct {
	SessionID      string          `json:"session_id"`
	FileName       string          `json:"file_name"`
	TotalSize      int64           `json:"total_size"`
	Status         string          `json:"status"`
	UploadedChunks []ChunkResponse `json:"uploaded_chunks"`
	ExpiresAt      string          `json:"expires_at"`
}

// CompleteUploadResponse is the finished file after completing an upload.
type CompleteUploadResponse struct {
	File FileResponse `json:"file"`
}
