package users

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	AvatarURL    string `json:"avatar_url"`
	AvatarSource string `json:"avatar_source"`
	Color        string `json:"color"`
	CreatedAt    string `json:"created_at"`
}

type MeResponse struct {
	User User `json:"user"`
}

type ListResponse struct {
	Users []User `json:"users"`
}

type UpdateRequest struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Password *string `json:"password"`
	Color    *string `json:"color"`
}

type ApiTokenResponse struct {
	ID        int64  `json:"id"`
	Token     string `json:"token,omitempty"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type ApiTokenListResponse struct {
	Tokens []ApiTokenResponse `json:"tokens"`
}

type CreateApiTokenRequest struct {
	Name string `json:"name"`
}
