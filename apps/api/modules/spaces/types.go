package spaces

type CreateSpaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateSpaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type AddMemberRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

type UpdateMemberRequest struct {
	Role string `json:"role"`
}

type SpaceResponse struct {
	ID          int64  `json:"id"`
	FacileID    string `json:"facile_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type SpaceListResponse struct {
	Spaces []SpaceResponse `json:"spaces"`
}

type MemberResponse struct {
	ID       int64        `json:"id"`
	UserID   int64        `json:"user_id"`
	Role     string       `json:"role"`
	JoinedAt string       `json:"joined_at"`
	User     *MemberUser  `json:"user,omitempty"`
}

type MemberUser struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Color     string `json:"color"`
}

type MemberListResponse struct {
	Members []MemberResponse `json:"members"`
}
