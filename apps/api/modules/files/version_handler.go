package files

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) reupload(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(64 << 20); err != nil {
		httpjson.WriteError(w, errors.TooLarge("file is too large"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("file is required"))
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	record, err := h.service.reuploadFile(r.Context(), userID, chi.URLParam(r, "id"), file, header.Size, mimeType)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}

func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := h.service.listVersions(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := VersionListResponse{Versions: make([]VersionResponse, 0, len(versions))}
	for _, v := range versions {
		resp.Versions = append(resp.Versions, mapVersion(v))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) restoreVersion(w http.ResponseWriter, r *http.Request) {
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

	record, err := h.service.restoreVersion(r.Context(), userID, chi.URLParam(r, "id"), chi.URLParam(r, "versionId"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}
