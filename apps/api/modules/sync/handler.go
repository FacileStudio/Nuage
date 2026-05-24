package sync

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
)

type Handler struct {
	service *Service
}

func (h *Handler) changes(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return
	}

	userID, err := strconv.ParseInt(identity.UserID, 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to parse user id", err))
		return
	}

	raw := r.URL.Query().Get("since")
	if raw == "" {
		httpjson.WriteError(w, errors.Invalid("since parameter is required"))
		return
	}

	since, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("since must be RFC3339 format"))
		return
	}

	resp, err := h.service.changes(r.Context(), userID, since)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return
	}

	userID, err := strconv.ParseInt(identity.UserID, 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to parse user id", err))
		return
	}

	resp, err := h.service.state(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, resp)
}
