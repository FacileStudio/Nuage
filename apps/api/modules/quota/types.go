package quota

// UsageResponse is a user's current storage usage against their limit.
type UsageResponse struct {
	UserID       int64   `json:"user_id"`
	StorageUsed  int64   `json:"storage_used"`
	StorageLimit int64   `json:"storage_limit"`
	Percentage   float64 `json:"percentage"`
	Source       string  `json:"source"`
	UpgradeURL   string  `json:"upgrade_url,omitempty"`
}

const (
	// SourcePlan marks a limit read from the person's Registre plan.
	SourcePlan = "plan"
	// SourceOverride marks a limit an admin set on the user.
	SourceOverride = "override"
	// SourceInstance marks the instance default quota, which applies until
	// Registre serves plans.
	SourceInstance = "instance"
)

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
