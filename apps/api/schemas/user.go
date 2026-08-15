package schemas

import "time"

// User is an application account.
type User struct {
	ID               int64     `gorm:"column:id;primaryKey"`
	Email            string    `gorm:"column:email;uniqueIndex"`
	Name             string    `gorm:"column:name"`
	AvatarURL        string    `gorm:"column:avatar_url"`
	AvatarSource     string    `gorm:"column:avatar_source"`
	AvatarUploadPath string    `gorm:"column:avatar_upload_path"`
	OIDCPictureURL   string    `gorm:"column:oidc_picture_url"`
	OIDCSubject      *string   `gorm:"column:oidc_subject;uniqueIndex"`
	Color            string    `gorm:"column:color"`
	PasswordHash     string    `gorm:"column:password_hash"`
	IsAdmin          bool      `gorm:"column:is_admin;default:false"`
	OIDCAccessToken  string    `gorm:"column:oidc_access_token"`
	OIDCRefreshToken string    `gorm:"column:oidc_refresh_token"`
	OIDCTokenExpiry  time.Time `gorm:"column:oidc_token_expiry"`
	ProfileSyncedAt  time.Time `gorm:"column:profile_synced_at"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (User) TableName() string { return "users" }

// AvatarFilePrefix is the route main.go mounts the avatar file server on. The derived
// avatar carries it so that no caller ever concatenates a base URL onto the value — the
// same string has to work when it is an absolute Porte URL instead of a local file, and
// `/api` + `https://…` is how this broke elsewhere in the suite. main_test.go asserts this
// constant still matches the route.
const AvatarFilePrefix = "/api/avatars/"

// AvatarSelectExpr is Avatar() as SQL, for any join that reads a user's picture without
// loading the row. It has to stay in step with Avatar below — hence both living here, one
// above the other, rather than one in Go and one buried in a Select string.
const AvatarSelectExpr = `COALESCE(NULLIF(users.oidc_picture_url, ''), ` +
	`NULLIF('` + AvatarFilePrefix + `' || COALESCE(users.avatar_upload_path, ''), '` + AvatarFilePrefix + `'), '')`

// Avatar is the picture to render. It is derived from the two sources rather than stored
// beside them: a photo set in Porte always wins, an upload shows only when the IdP offers
// none, and because nothing is written there is no third value that can drift out of
// agreement with the two that matter.
func (u User) Avatar() string {
	if u.OIDCPictureURL != "" {
		return u.OIDCPictureURL
	}
	if u.AvatarUploadPath != "" {
		return AvatarFilePrefix + u.AvatarUploadPath
	}
	return ""
}

// AvatarOrigin names where Avatar came from, so the client can say *why* uploading is
// unavailable instead of just hiding the button.
func (u User) AvatarOrigin() string {
	switch {
	case u.OIDCPictureURL != "":
		return "oidc"
	case u.AvatarUploadPath != "":
		return "upload"
	default:
		return ""
	}
}
