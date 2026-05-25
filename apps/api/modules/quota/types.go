package quota

type UsageResponse struct {
	UserID       int64  `json:"user_id"`
	StorageUsed  int64  `json:"storage_used"`
	StorageLimit int64  `json:"storage_limit"`
	Percentage   float64 `json:"percentage"`
}

type SetQuotaRequest struct {
	StorageLimit int64 `json:"storage_limit"`
}

type AdminUsageResponse struct {
	Users []UsageResponse `json:"users"`
}
