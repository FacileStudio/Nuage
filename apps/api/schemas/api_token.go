package schemas

import "time"

type ApiToken struct {
	ID        int64     `gorm:"column:id;primaryKey;autoIncrement"`
	Token     string    `gorm:"column:token;uniqueIndex"`
	UserID    int64     `gorm:"column:user_id;index"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (ApiToken) TableName() string { return "api_tokens" }
