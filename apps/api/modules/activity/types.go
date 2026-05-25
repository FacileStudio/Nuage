package activity

type ActivityResponse struct {
	ID           int64  `json:"id"`
	UserID       int64  `json:"user_id"`
	EventType    string `json:"event_type"`
	ResourceType string `json:"resource_type"`
	ResourceID   int64  `json:"resource_id"`
	ResourceName string `json:"resource_name"`
	Metadata     string `json:"metadata,omitempty"`
	CreatedAt    string `json:"created_at"`
}

type ActivityListResponse struct {
	Activities []ActivityResponse `json:"activities"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	PerPage    int                `json:"per_page"`
}
