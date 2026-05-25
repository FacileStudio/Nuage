package quota

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) getMyUsage(w http.ResponseWriter, r *http.Request) {
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

	usage, err := h.service.GetUsage(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, usage)
}

func (h *Handler) listAllUsage(w http.ResponseWriter, r *http.Request) {
	usages, err := h.service.ListAllUsage(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, AdminUsageResponse{Users: usages})
}

func (h *Handler) setUserQuota(w http.ResponseWriter, r *http.Request) {
	targetUserID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("invalid user id"))
		return
	}

	var req SetQuotaRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if err := h.service.SetLimit(r.Context(), targetUserID, req.StorageLimit); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	usage, err := h.service.GetUsage(r.Context(), targetUserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, usage)
}

func (h *Handler) recalculate(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.RecalculateUsage(r.Context(), userID); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	usage, err := h.service.GetUsage(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, usage)
}
