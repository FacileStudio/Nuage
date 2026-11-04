package schemas

import "time"

// EntitlementStateID is the primary key of the only EntitlementState row.
const EntitlementStateID int64 = 1

// EntitlementState holds the last plan entitlements Registre served, as the
// document forfait's Store.Export produces. It is one row, rewritten after
// every directory sync and read back at boot, so a restart keeps enforcing the
// last known plans instead of failing open again.
type EntitlementState struct {
	ID        int64     `json:"id" gorm:"primaryKey"`
	Document  string    `json:"-" gorm:"type:text;not null"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (EntitlementState) TableName() string { return "entitlement_state" }
