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

type TestNookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
