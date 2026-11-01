package files

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/thumbnail"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) thumbnail(w http.ResponseWriter, r *http.Request) {
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
	width, err := thumbnail.Width(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	record, err := h.service.getFile(r.Context(), userID, chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := h.service.thumbs.Serve(w, r, record, width, downloadCacheControl(r)); err != nil {
		httpjson.WriteError(w, err)
	}
}
