package settings

// SettingResponse is one application setting.
type SettingResponse struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

// SettingsListResponse is a list of application settings.
type SettingsListResponse struct {
	Settings []SettingResponse `json:"settings"`
}

// UpdateSettingsRequest is the body used to update settings in bulk.
type UpdateSettingsRequest struct {
	Settings map[string]string `json:"settings"`
}

// TestAntenneRequest holds the webhook connection to probe.
type TestAntenneRequest struct {
	URL     string `json:"url"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

// TestAntenneResponse reports whether a test delivery succeeded.
type TestAntenneResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeliveryResponse is one Antenne webhook delivery attempt.
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

// DeliveryListResponse is a page of Antenne delivery attempts.
type DeliveryListResponse struct {
	Deliveries []DeliveryResponse `json:"deliveries"`
	Total      int64              `json:"total"`
}
