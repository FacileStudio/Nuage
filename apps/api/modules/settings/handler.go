package settings

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) checkAdmin(w http.ResponseWriter, r *http.Request) (bool, int64) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return false, 0
	}
	userID, err := strconv.ParseInt(identity.UserID, 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to parse user id", err))
		return false, 0
	}
	admin, err := h.service.isAdmin(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return false, 0
	}
	if !admin {
		httpjson.WriteError(w, errors.Forbidden("admin access required"))
		return false, 0
	}
	return true, userID
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if ok, _ := h.checkAdmin(w, r); !ok {
		return
	}

	records, err := h.service.listSettings(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := SettingsListResponse{Settings: make([]SettingResponse, 0, len(records))}
	for _, record := range records {
		value := record.Value
		if strings.Contains(record.Key, "secret") && len(value) > 4 {
			value = strings.Repeat("*", len(value)-4) + value[len(value)-4:]
		}
		resp.Settings = append(resp.Settings, SettingResponse{
			Key:       record.Key,
			Value:     value,
			UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	if ok, _ := h.checkAdmin(w, r); !ok {
		return
	}

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
	if ok, _ := h.checkAdmin(w, r); !ok {
		return
	}

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
