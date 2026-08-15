package auth

// RegisterRequest is the body of a register call.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the body of a login call.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is the result of a successful register or login.
type AuthResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

// Data holds the authenticated caller's profile exposed on the auth token.
type Data struct {
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

func (d *Data) GetEmail() string {
	if d == nil {
		return ""
	}
	return d.Email
}

func (d *Data) GetIsAdmin() bool {
	if d == nil {
		return false
	}
	return d.IsAdmin
}
