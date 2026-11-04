package spaces

import (
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

// callerAndSpace reads the authenticated user and the space id of a billing
// request, answering the error itself when either is missing.
func callerAndSpace(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	identity, ok := authcontext.IdentityFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, errors.Unauthorized("missing auth"))
		return 0, 0, false
	}
	userID, err := strconv.ParseInt(identity.UserID, 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Internal("failed to parse user id", err))
		return 0, 0, false
	}
	spaceID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("invalid space id"))
		return 0, 0, false
	}
	return userID, spaceID, true
}

// answer writes 204 on success and the error otherwise.
func answer(w http.ResponseWriter, err error) {
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) offerBilling(w http.ResponseWriter, r *http.Request) {
	userID, spaceID, ok := callerAndSpace(w, r)
	if !ok {
		return
	}
	var req TransferBillingRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	answer(w, h.service.offerBilling(r.Context(), userID, spaceID, req.UserID))
}

func (h *Handler) revokeBilling(w http.ResponseWriter, r *http.Request) {
	if userID, spaceID, ok := callerAndSpace(w, r); ok {
		answer(w, h.service.revokeBilling(r.Context(), userID, spaceID))
	}
}

func (h *Handler) acceptBilling(w http.ResponseWriter, r *http.Request) {
	if userID, spaceID, ok := callerAndSpace(w, r); ok {
		answer(w, h.service.acceptBilling(r.Context(), userID, spaceID))
	}
}
