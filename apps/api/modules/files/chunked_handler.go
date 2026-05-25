package files

import (
	"net/http"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) initUpload(w http.ResponseWriter, r *http.Request) {
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

	var req InitUploadRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	session, err := h.service.initUpload(r.Context(), userID, req)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, InitUploadResponse{
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) uploadChunk(w http.ResponseWriter, r *http.Request) {
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

	sessionID := chi.URLParam(r, "sessionId")
	partNumber, err := strconv.Atoi(chi.URLParam(r, "partNumber"))
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("invalid part number"))
		return
	}

	defer r.Body.Close()

	chunk, err := h.service.uploadChunk(r.Context(), userID, sessionID, partNumber, r.Body, r.ContentLength)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, ChunkResponse{
		PartNumber: chunk.PartNumber,
		Size:       chunk.Size,
		Hash:       chunk.Hash,
	})
}

func (h *Handler) completeUpload(w http.ResponseWriter, r *http.Request) {
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

	sessionID := chi.URLParam(r, "sessionId")

	record, err := h.service.completeUpload(r.Context(), userID, sessionID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, CompleteUploadResponse{
		File: mapFile(*record),
	})
}

func (h *Handler) getUploadStatus(w http.ResponseWriter, r *http.Request) {
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

	sessionID := chi.URLParam(r, "sessionId")
	session, chunks, err := h.service.getUploadStatus(r.Context(), userID, sessionID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	uploaded := make([]ChunkResponse, 0, len(chunks))
	for _, c := range chunks {
		uploaded = append(uploaded, ChunkResponse{
			PartNumber: c.PartNumber,
			Size:       c.Size,
			Hash:       c.Hash,
		})
	}

	httpjson.WriteJSON(w, http.StatusOK, SessionStatusResponse{
		SessionID:      session.ID,
		FileName:       session.FileName,
		TotalSize:      session.TotalSize,
		Status:         session.Status,
		UploadedChunks: uploaded,
		ExpiresAt:      session.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) abortUpload(w http.ResponseWriter, r *http.Request) {
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

	sessionID := chi.URLParam(r, "sessionId")
	if err := h.service.abortUpload(r.Context(), userID, sessionID); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"aborted": true})
}
