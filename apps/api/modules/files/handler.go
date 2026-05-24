package files

import (
	"io"
	"net/http"
	"strconv"
	"strings"

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

	records, err := h.service.listFiles(r.Context(), folderID, query.Get("search"), query.Get("linked_to"), query.Get("origin_app"))
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
	record, err := h.service.getFile(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}

func (h *Handler) download(w http.ResponseWriter, r *http.Request) {
	reader, record, err := h.service.downloadFile(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", record.MimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+record.Name+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(record.Size, 10))
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if err := h.service.deleteFile(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (h *Handler) updateFile(w http.ResponseWriter, r *http.Request) {
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

	record, err := h.service.updateFile(r.Context(), chi.URLParam(r, "id"), req.Name, req.FolderID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFile(*record))
}

func (h *Handler) linkFile(w http.ResponseWriter, r *http.Request) {
	var req LinkFileRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if strings.TrimSpace(req.LinkedTo) == "" {
		httpjson.WriteError(w, errors.Invalid("linked_to is required"))
		return
	}

	record, err := h.service.linkFile(r.Context(), chi.URLParam(r, "id"), strings.TrimSpace(req.LinkedTo))
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
	httpjson.WriteJSON(w, http.StatusCreated, mapFolder(*record))
}

func (h *Handler) listFolders(w http.ResponseWriter, r *http.Request) {
	var parentID *int64
	if raw := r.URL.Query().Get("parent_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid parent_id"))
			return
		}
		parentID = &id
	}

	records, err := h.service.listFolders(r.Context(), parentID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := FolderListResponse{Folders: make([]FolderResponse, 0, len(records))}
	for _, record := range records {
		resp.Folders = append(resp.Folders, mapFolder(record))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) getFolder(w http.ResponseWriter, r *http.Request) {
	folder, childFiles, childFolders, err := h.service.getFolder(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	resp := FolderDetailResponse{
		Folder:  mapFolder(*folder),
		Files:   make([]FileResponse, 0, len(childFiles)),
		Folders: make([]FolderResponse, 0, len(childFolders)),
	}
	for _, f := range childFiles {
		resp.Files = append(resp.Files, mapFile(f))
	}
	for _, f := range childFolders {
		resp.Folders = append(resp.Folders, mapFolder(f))
	}
	httpjson.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) updateFolder(w http.ResponseWriter, r *http.Request) {
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

	record, err := h.service.updateFolder(r.Context(), chi.URLParam(r, "id"), req.Name, req.ParentID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, mapFolder(*record))
}

func (h *Handler) deleteFolder(w http.ResponseWriter, r *http.Request) {
	if err := h.service.deleteFolder(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}
