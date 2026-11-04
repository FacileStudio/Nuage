package spaces

import (
	"context"
	"net/http"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// TransferBillingRequest names the owner who starts paying for a space
// another app provisioned.
type TransferBillingRequest struct {
	UserID int64 `json:"user_id"`
}

// refusePayerRemoval refuses taking the member whose personal plan pays for
// the space out of it while anyone else stays in it: the others would keep
// storing bytes on the plan of someone who can no longer see them. It must
// run under lockMembers.
func refusePayerRemoval(tx *gorm.DB, spaceID, memberUserID int64) error {
	var others int64
	err := tx.Model(&schemas.SpaceMember{}).
		Where("space_id = ? AND user_id <> ? AND EXISTS (SELECT 1 FROM spaces WHERE id = ? AND charged_to = ?)",
			spaceID, memberUserID, spaceID, memberUserID).Count(&others).Error
	if err != nil {
		return errors.Internal("failed to check who pays for the space", err)
	}
	if others > 0 {
		return errors.Conflict("this member pays for the space's storage; transfer billing to another owner first")
	}
	return nil
}

// transferBilling moves who pays for a provisioned space from the caller to
// another owner of it. Only the current payer may hand it over, and only to a
// member who is already an owner, all under the members lock.
func (s *Service) transferBilling(ctx context.Context, userID, spaceID, toUserID int64) error {
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMembers(ctx, tx, spaceID); err != nil {
			return err
		}
		var owners int64
		if err := tx.Model(&schemas.SpaceMember{}).
			Where("space_id = ? AND user_id = ? AND role = 'owner'", spaceID, toUserID).Count(&owners).Error; err != nil {
			return errors.Internal("failed to check the new payer", err)
		}
		if owners == 0 {
			return errors.Invalid("billing can only move to an owner of the space")
		}
		moved := tx.Model(&schemas.Space{}).Where("id = ? AND charged_to = ?", spaceID, userID).Update("charged_to", toUserID)
		if moved.Error != nil {
			return errors.Internal("failed to transfer billing", moved.Error)
		}
		if moved.RowsAffected == 0 {
			return errors.Forbidden("only the member who pays for this space can transfer its billing")
		}
		return nil
	})
}

func (h *Handler) transferBilling(w http.ResponseWriter, r *http.Request) {
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
	spaceID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("invalid space id"))
		return
	}
	var req TransferBillingRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := h.service.transferBilling(r.Context(), userID, spaceID, req.UserID); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
