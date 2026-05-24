package sharing

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

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
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

	var req CreateShareRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	record, err := h.service.createShare(r.Context(), userID, req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, mapShare(*record))
}

func (h *Handler) listSharedWithMe(w http.ResponseWriter, r *http.Request) {
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

	records, err := h.service.listSharedWithMe(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := ShareListResponse{Shares: make([]ShareResponse, 0, len(records))}
	for _, record := range records {
		resp.Shares = append(resp.Shares, mapShare(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) listSharedByMe(w http.ResponseWriter, r *http.Request) {
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

	records, err := h.service.listSharedByMe(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := ShareListResponse{Shares: make([]ShareResponse, 0, len(records))}
	for _, record := range records {
		resp.Shares = append(resp.Shares, mapShare(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) deleteShare(w http.ResponseWriter, r *http.Request) {
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

	if err := h.service.deleteShare(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) getPublic(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	record, err := h.service.getByToken(r.Context(), token)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := PublicShareResponse{
		Token:      record.Token,
		Permission: record.Permission,
	}

	if record.File != nil {
		resp.File = &PublicFile{
			ID:       record.File.ID,
			FacileID: record.File.FacileID,
			Name:     record.File.Name,
			MimeType: record.File.MimeType,
			Size:     record.File.Size,
		}
	}
	if record.Folder != nil {
		resp.Folder = &PublicFolder{
			ID:       record.Folder.ID,
			FacileID: record.Folder.FacileID,
			Name:     record.Folder.Name,
		}
	}

	httpjson.WriteJSON(w, http.StatusOK, resp)
}
