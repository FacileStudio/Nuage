package schemas

import "time"

// TokenScope restricts one API token, which is a porte session, to a set of
// spaces and stamps the app that owns it. A token with no row is a personal
// token with the full reach of its user.
type TokenScope struct {
	SessionID      int64     `json:"session_id" gorm:"primaryKey"`
	App            string    `json:"app" gorm:"not null;default:''"`
	SpaceFacileIDs []string  `json:"space_facile_ids" gorm:"serializer:json;type:jsonb;not null"`
	CreatedAt      time.Time `json:"created_at"`
}

func (TokenScope) TableName() string { return "api_token_scopes" }

const tokenScopeSchema = `
CREATE TABLE IF NOT EXISTS api_token_scopes (
	session_id        bigint PRIMARY KEY REFERENCES porte_sessions(id) ON DELETE CASCADE,
	app               text NOT NULL DEFAULT '',
	space_facile_ids  jsonb NOT NULL DEFAULT '[]',
	created_at        timestamptz NOT NULL DEFAULT now()
);
`
