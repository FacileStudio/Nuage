package schemas

import "time"

type UserQuota struct {
	UserID       int64     `json:"user_id" gorm:"primaryKey"`
	StorageUsed  int64     `json:"storage_used" gorm:"not null;default:0"`
	StorageLimit int64     `json:"storage_limit" gorm:"not null;default:0"`
	UpdatedAt    time.Time `json:"updated_at"`
	User         User      `json:"-" gorm:"foreignKey:UserID"`
}

func (UserQuota) TableName() string { return "user_quotas" }
