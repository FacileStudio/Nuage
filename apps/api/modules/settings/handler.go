package settings

import (
	"net/http"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	records, err := h.service.listSettings(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := SettingsListResponse{Settings: make([]SettingResponse, 0, len(records))}
	for _, record := range records {
		resp.Settings = append(resp.Settings, SettingResponse{
			Key:       record.Key,
			Value:     record.Value,
			UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if len(req.Settings) == 0 {
		httpjson.WriteError(w, errors.Invalid("settings map is required"))
		return
	}

	records, err := h.service.updateSettings(r.Context(), req.Settings)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := SettingsListResponse{Settings: make([]SettingResponse, 0, len(records))}
	for _, record := range records {
		resp.Settings = append(resp.Settings, SettingResponse{
			Key:       record.Key,
			Value:     record.Value,
			UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) testNook(w http.ResponseWriter, r *http.Request) {
	var req TestNookRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	success, message, err := h.service.testNook(r.Context(), req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, TestNookResponse{
		Success: success,
		Message: message,
	})
}
