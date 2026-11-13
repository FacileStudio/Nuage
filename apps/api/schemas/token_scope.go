package schemas

import "time"

// TokenScope restricts one API token, which is a porte session, to a set of
// spaces and stamps the app that owns it. A token with no row is a personal
// token with the full reach of its user.
type TokenScope struct {
	SessionID      int64     `json:"session_id" gorm:"primaryKey"`
	App            string    `json:"app" gorm:"not null;default:''"`
	Provision      bool      `json:"provision" gorm:"not null;default:false"`
	SpaceFacileIDs []string  `json:"space_facile_ids" gorm:"serializer:json;type:jsonb;not null"`
	CreatedAt      time.Time `json:"created_at"`
}

func (TokenScope) TableName() string { return "api_token_scopes" }
