package users

// User is a user profile as returned by the API.
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url"`
	AvatarSource string `json:"avatar_source"`
	Color        string `json:"color"`
	CreatedAt    string `json:"created_at"`
}

// MeResponse wraps the profile of the current user.
type MeResponse struct {
	User User `json:"user"`
}

// ListResponse is a list of user profiles.
type ListResponse struct {
	Users []User `json:"users"`
}

// UpdateRequest is the body used to update the caller's profile.
type UpdateRequest struct {
	Name            *string `json:"name"`
	Email           *string `json:"email"`
	Password        *string `json:"password"`
	CurrentPassword *string `json:"current_password"`
	Color           *string `json:"color"`
}

// ApiTokenResponse is one named API token.
type ApiTokenResponse struct {
	ID        int64  `json:"id"`
	Token     string `json:"token,omitempty"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

// ApiTokenListResponse is a list of the caller's API tokens.
type ApiTokenListResponse struct {
	Tokens []ApiTokenResponse `json:"tokens"`
}

// CreateApiTokenRequest is the body used to create an API token.
type CreateApiTokenRequest struct {
	Name string `json:"name"`
}
