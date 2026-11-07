package users

// User is a user profile as returned by the API.
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url"`
	AvatarSource string `json:"avatar_source"`
	Color        string `json:"color"`
	Locale       string `json:"locale"`
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

// UpdateRequest is the body used to update the caller's profile. The email
// address is not editable: it comes from the Facile account at each sign-in.
type UpdateRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
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
//
// App restricts the token and needs SpaceFacileIDs. The token reaches only the
// listed spaces, cannot touch personal files or any other endpoint family, and
// stamps every upload with App as its origin_app. Provision is retired and a
// request that sets it is refused.
type CreateApiTokenRequest struct {
	Name           string   `json:"name"`
	App            string   `json:"app"`
	SpaceFacileIDs []string `json:"space_facile_ids"`
	Provision      bool     `json:"provision"`
}
