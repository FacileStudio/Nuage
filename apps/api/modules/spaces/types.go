package spaces

// CreateSpaceRequest is the body used to create a space.
type CreateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateSpaceRequest is the body used to update a space.
type UpdateSpaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// AddMemberRequest is the body used to add a member to a space.
type AddMemberRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

// UpdateMemberRequest is the body used to change a member's role.
type UpdateMemberRequest struct {
	Role string `json:"role"`
}

// SpaceResponse is one space as returned by the API.
type SpaceResponse struct {
	ID          int64  `json:"id"`
	FacileID    string `json:"facile_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// SpaceListResponse is a list of the caller's spaces.
type SpaceListResponse struct {
	Spaces []SpaceResponse `json:"spaces"`
}

// MemberResponse is one space member with their role.
type MemberResponse struct {
	ID       int64       `json:"id"`
	UserID   int64       `json:"user_id"`
	Role     string      `json:"role"`
	JoinedAt string      `json:"joined_at"`
	User     *MemberUser `json:"user,omitempty"`
}

// MemberUser is the embedded user profile of a space member.
type MemberUser struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Color     string `json:"color"`
}

// MemberListResponse is a list of a space's members.
type MemberListResponse struct {
	Members []MemberResponse `json:"members"`
}
