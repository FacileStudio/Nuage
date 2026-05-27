package settings

type SettingResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

type SettingsListResponse struct {
	Settings []SettingResponse `json:"settings"`
}

type UpdateSettingsRequest struct {
	Settings map[string]string `json:"settings"`
}

type TestNookRequest struct {
	URL     string `json:"url"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

type TestNookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type DeliveryResponse struct {
	ID           int64   `json:"id"`
	EventType    string  `json:"event_type"`
	Status       string  `json:"status"`
	Attempts     int     `json:"attempts"`
	ResponseCode *int    `json:"response_code,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	LatencyMs    *int    `json:"latency_ms,omitempty"`
	CreatedAt    string  `json:"created_at"`
	DeliveredAt  *string `json:"delivered_at,omitempty"`
}

type DeliveryListResponse struct {
	Deliveries []DeliveryResponse `json:"deliveries"`
	Total      int64              `json:"total"`
}
