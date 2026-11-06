package appfolders

import (
	"net/http"

	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

func (h *handler) usage(w http.ResponseWriter, r *http.Request) {
	userID, _, err := caller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	apps, err := h.service.Usage(r.Context(), userID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, AppDataResponse{Apps: apps})
}

func (h *handler) deleteAppData(w http.ResponseWriter, r *http.Request) {
	userID, _, err := caller(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := h.service.DeleteAppData(r.Context(), userID, chi.URLParam(r, "app")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
