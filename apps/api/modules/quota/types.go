package quota

type UsageResponse struct {
	UserID       int64   `json:"user_id"`
	StorageUsed  int64   `json:"storage_used"`
	StorageLimit int64   `json:"storage_limit"`
	Percentage   float64 `json:"percentage"`
}

// UnlimitedStorageLimit is the only accepted negative storage_limit value; it
// grants unlimited storage. A limit of 0 applies the instance default quota.
const UnlimitedStorageLimit int64 = -1

type SetQuotaRequest struct {
	StorageLimit int64 `json:"storage_limit"`
}

type AdminUsageResponse struct {
	Users []UsageResponse `json:"users"`
}
