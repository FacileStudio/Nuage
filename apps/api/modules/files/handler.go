package files

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
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

	name := header.Filename
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	var folderID *int64
	if raw := r.FormValue("folder_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid folder_id"))
			return
		}
		folderID = &id
	}

	originApp := r.FormValue("origin_app")

	record, err := h.service.uploadFile(r.Context(), userID, name, mimeType, header.Size, file, folderID, originApp)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, mapFile(*record))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
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

	query := r.URL.Query()

	var folderID *int64
	if raw := query.Get("folder_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid folder_id"))
			return
		}
		folderID = &id
	}

	records, err := h.service.listFiles(r.Context(), userID, folderID, query.Get("search"), query.Get("linked_to"), query.Get("origin_app"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := FileListResponse{Files: make([]FileResponse, 0, len(records))}
	for _, record := range records {
		resp.Files = append(resp.Files, mapFile(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
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

	record, err := h.service.getFile(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
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

	reader, record, err := h.service.downloadFile(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", record.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(record.Name)))
	w.Header().Set("Content-Length", strconv.FormatInt(record.Size, 10))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (h *Handler) presign(w http.ResponseWriter, r *http.Request) {
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

	var req PresignRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := httpjson.DecodeJSON(w, r, &req); err != nil {
			httpjson.WriteError(w, err)
			return
		}
	}

	expiresIn := int64(3600)
	if req.ExpiresIn != nil {
		expiresIn = *req.ExpiresIn
	}
	if expiresIn < 60 {
		httpjson.WriteError(w, errors.Invalid("expires_in must be at least 60 seconds"))
		return
	}
	if expiresIn > 604800 {
		httpjson.WriteError(w, errors.Invalid("expires_in must not exceed 604800 seconds (7 days)"))
		return
	}

	dur := time.Duration(expiresIn) * time.Second
	presignedURL, expiresAt, err := h.service.presignFile(r.Context(), userID, chi.URLParam(r, "id"), dur)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, PresignResponse{
		URL:       presignedURL,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
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
	if err := h.service.deleteFile(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) updateFile(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateFileRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			httpjson.WriteError(w, errors.Invalid("name cannot be empty"))
			return
		}
		req.Name = &trimmed
	}

	record, err := h.service.updateFile(r.Context(), userID, chi.URLParam(r, "id"), req.Name, req.FolderID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}

func (h *Handler) linkFile(w http.ResponseWriter, r *http.Request) {
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

	var req LinkFileRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if strings.TrimSpace(req.LinkedTo) == "" {
		httpjson.WriteError(w, errors.Invalid("linked_to is required"))
		return
	}

	record, err := h.service.linkFile(r.Context(), userID, chi.URLParam(r, "id"), strings.TrimSpace(req.LinkedTo))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}

func (h *Handler) createFolder(w http.ResponseWriter, r *http.Request) {
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

	var req CreateFolderRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpjson.WriteError(w, errors.Invalid("folder name is required"))
		return
	}

	record, err := h.service.createFolder(r.Context(), userID, name, req.ParentID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusCreated, mapFolder(*record, 0))
}

func (h *Handler) listFolders(w http.ResponseWriter, r *http.Request) {
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

	var parentID *int64
	if raw := r.URL.Query().Get("parent_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid parent_id"))
			return
		}
		parentID = &id
	}

	records, err := h.service.listFolders(r.Context(), userID, parentID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	folderIDs := make([]int64, len(records))
	for i, rec := range records {
		folderIDs[i] = rec.ID
	}
	sizes, err := h.service.folderSizes(r.Context(), folderIDs)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := FolderListResponse{Folders: make([]FolderResponse, 0, len(records))}
	for _, record := range records {
		resp.Folders = append(resp.Folders, mapFolder(record, sizes[record.ID]))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) getFolder(w http.ResponseWriter, r *http.Request) {
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

	folder, childFiles, childFolders, err := h.service.getFolder(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	folderIDs := make([]int64, 0, 1+len(childFolders))
	folderIDs = append(folderIDs, folder.ID)
	for _, f := range childFolders {
		folderIDs = append(folderIDs, f.ID)
	}
	sizes, err := h.service.folderSizes(r.Context(), folderIDs)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := FolderDetailResponse{
		Folder:  mapFolder(*folder, sizes[folder.ID]),
		Files:   make([]FileResponse, 0, len(childFiles)),
		Folders: make([]FolderResponse, 0, len(childFolders)),
	}
	for _, f := range childFiles {
		resp.Files = append(resp.Files, mapFile(f))
	}
	for _, f := range childFolders {
		resp.Folders = append(resp.Folders, mapFolder(f, sizes[f.ID]))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) updateFolder(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateFolderRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			httpjson.WriteError(w, errors.Invalid("folder name cannot be empty"))
			return
		}
		req.Name = &trimmed
	}

	record, err := h.service.updateFolder(r.Context(), userID, chi.URLParam(r, "id"), req.Name, req.ParentID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	sizes, sErr := h.service.folderSizes(r.Context(), []int64{record.ID})
	if sErr != nil {
		httpjson.WriteError(w, sErr)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFolder(*record, sizes[record.ID]))
}

func (h *Handler) deleteFolder(w http.ResponseWriter, r *http.Request) {
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
	if err := h.service.deleteFolder(r.Context(), userID, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
