package files

type VersionResponse struct {
	ID        int64  `json:"id"`
	FileID    int64  `json:"file_id"`
	Version   int    `json:"version"`
	Hash      string `json:"hash"`
	Size      int64  `json:"size"`
	CreatedBy int64  `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type VersionListResponse struct {
	Versions []VersionResponse `json:"versions"`
}

type VersionDiffResponse struct {
	Version    int    `json:"version"`
	SizeBefore int64  `json:"size_before"`
	SizeAfter  int64  `json:"size_after"`
	SizeDelta  int64  `json:"size_delta"`
	CreatedAt  string `json:"created_at"`
}
