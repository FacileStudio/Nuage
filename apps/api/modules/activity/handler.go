package activity

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func newHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) listAll(w http.ResponseWriter, r *http.Request) {
	params := parseListParams(r)
	records, total, err := h.service.List(r.Context(), params)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	writeActivityList(w, records, total, params.Page, params.PerPage)
}

func (h *Handler) listMine(w http.ResponseWriter, r *http.Request) {
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

	params := parseListParams(r)
	params.UserID = &userID

	records, total, err := h.service.List(r.Context(), params)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	writeActivityList(w, records, total, params.Page, params.PerPage)
}

func (h *Handler) forFile(w http.ResponseWriter, r *http.Request) {
	fileID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("invalid file id"))
		return
	}

	page, perPage := parsePagination(r)
	records, total, err := h.service.ForFile(r.Context(), fileID, page, perPage)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	writeActivityList(w, records, total, page, perPage)
}

func parseListParams(r *http.Request) ListParams {
	q := r.URL.Query()
	page, perPage := parsePagination(r)

	params := ListParams{
		EventType:    q.Get("event_type"),
		ResourceType: q.Get("resource_type"),
		Page:         page,
		PerPage:      perPage,
	}

	if raw := q.Get("resource_id"); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			params.ResourceID = &id
		}
	}

	return params
}

func parsePagination(r *http.Request) (int, int) {
	q := r.URL.Query()
	page := 1
	perPage := 50

	if raw := q.Get("page"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			page = n
		}
	}
	if raw := q.Get("per_page"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 100 {
			perPage = n
		}
	}

	return page, perPage
}

func writeActivityList(w http.ResponseWriter, records []schemas.ActivityLog, total int64, page, perPage int) {
	activities := make([]ActivityResponse, 0, len(records))
	for _, r := range records {
		activities = append(activities, mapActivity(r))
	}
	httpjson.WriteJSON(w, http.StatusOK, ActivityListResponse{
		Activities: activities,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
	})
}
