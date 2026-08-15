package quota

// UsageResponse is a user's current storage usage against their limit.
type UsageResponse struct {
	UserID       int64   `json:"user_id"`
	StorageUsed  int64   `json:"storage_used"`
	StorageLimit int64   `json:"storage_limit"`
	Percentage   float64 `json:"percentage"`
}

// UnlimitedStorageLimit is the only accepted negative storage_limit value; it
// grants unlimited storage. A limit of 0 applies the instance default quota.
const UnlimitedStorageLimit int64 = -1

// SetQuotaRequest is the body used to set a user's storage limit.
type SetQuotaRequest struct {
	StorageLimit int64 `json:"storage_limit"`
}

// AdminUsageResponse lists the usage of every user.
type AdminUsageResponse struct {
	Users []UsageResponse `json:"users"`
}
