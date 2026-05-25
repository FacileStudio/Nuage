package schemas

import "time"

type ActivityLog struct {
	ID           int64     `json:"id" gorm:"primaryKey"`
	UserID       int64     `json:"user_id" gorm:"index;not null"`
	EventType    string    `json:"event_type" gorm:"index;not null"`
	ResourceType string    `json:"resource_type" gorm:"index;not null"`
	ResourceID   int64     `json:"resource_id"`
	ResourceName string    `json:"resource_name"`
	Metadata     string    `json:"metadata" gorm:"type:text"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
	User         User      `json:"-" gorm:"foreignKey:UserID"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
