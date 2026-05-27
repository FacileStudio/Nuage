package schemas

import "time"

type NookDelivery struct {
	ID           int64      `gorm:"primaryKey"`
	EventType    string     `gorm:"not null;index"`
	Payload      string     `gorm:"type:text;not null"`
	Status       string     `gorm:"not null;default:pending;index:idx_nook_status_retry"`
	Attempts     int        `gorm:"not null;default:0"`
	NextRetryAt  *time.Time `gorm:"index:idx_nook_status_retry"`
	ResponseCode *int
	ResponseBody *string    `gorm:"type:text"`
	ErrorMessage *string    `gorm:"type:text"`
	LatencyMs    *int
	CreatedAt    time.Time  `gorm:"autoCreateTime"`
	DeliveredAt  *time.Time
}
