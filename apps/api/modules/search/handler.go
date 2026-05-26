package search

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	q := query.Get("q")
	if q == "" {
		httpjson.WriteError(w, errors.Invalid("q parameter is required"))
		return
	}

	filterType := query.Get("type")

	var folderID *int64
	if raw := query.Get("folder_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid folder_id"))
			return
		}
		folderID = &id
	}

	limit := 50
	if raw := query.Get("limit"); raw != "" {
		l, err := strconv.Atoi(raw)
		if err != nil {
			httpjson.WriteError(w, errors.Invalid("invalid limit"))
			return
		}
		limit = l
	}

	resp, err := h.service.Search(r.Context(), q, filterType, folderID, limit)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, resp)
}
