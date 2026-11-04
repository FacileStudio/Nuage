package spaces

import (
	"context"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// billingOfferTTL is how long an offer to take over a space's billing stays
// open before the payer has to make it again.
const billingOfferTTL = 7 * 24 * time.Hour

// TransferBillingRequest names the owner offered to pay for a space another
// app provisioned.
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
		return errors.Conflict("this member pays for the space's storage; another owner must accept its billing first")
	}
	return nil
}

// offerBilling lets the current payer offer the space's billing to another
// owner. Nothing moves until that owner accepts; a new offer replaces the
// previous one.
func (s *Service) offerBilling(ctx context.Context, userID, spaceID, toUserID int64) error {
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMembers(ctx, tx, spaceID); err != nil {
			return err
		}
		var payer int64
		if err := tx.Model(&schemas.Space{}).Where("id = ? AND charged_to = ?", spaceID, userID).Count(&payer).Error; err != nil {
			return errors.Internal("failed to check who pays for the space", err)
		}
		if payer == 0 {
			return errors.Forbidden("only the member who pays for this space can change its billing")
		}
		if toUserID == userID || !isOwner(tx, spaceID, toUserID) {
			return errors.Invalid("billing can only be offered to another owner of the space")
		}
		end := time.Now().Add(billingOfferTTL)
		offered := tx.Model(&schemas.Space{}).Where("id = ? AND charged_to = ?", spaceID, userID).
			Updates(map[string]any{"billing_offer_to": toUserID, "billing_offer_end": end})
		return payerOnly(offered)
	})
}

// revokeBilling withdraws the payer's pending offer.
func (s *Service) revokeBilling(ctx context.Context, userID, spaceID int64) error {
	revoked := s.orm.WithContext(ctx).Model(&schemas.Space{}).Where("id = ? AND charged_to = ?", spaceID, userID).
		Updates(map[string]any{"billing_offer_to": nil, "billing_offer_end": nil})
	return payerOnly(revoked)
}

// acceptBilling makes the caller the payer when the payer offered it to them,
// the offer is still open and they are still an owner. Both people's plan
// locks are held while it moves, so no upload is admitted against the old
// payer's plan halfway through. The space row is locked FOR NO KEY UPDATE: an
// upload holding a plan lock needs KEY SHARE on it for the files.space_id
// foreign key, which FOR UPDATE would block, deadlocking the two.
func (s *Service) acceptBilling(ctx context.Context, userID, spaceID int64) error {
	return s.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockMembers(ctx, tx, spaceID); err != nil {
			return err
		}
		var space schemas.Space
		if err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Where("id = ?", spaceID).Take(&space).Error; err != nil {
			return errors.NotFound("space not found")
		}
		if !offeredTo(space, userID, time.Now()) || !isOwner(tx, spaceID, userID) {
			return errors.Forbidden("there is no open billing offer for you on this space")
		}
		if err := quota.LockPeople(tx, *space.ChargedTo, userID); err != nil {
			return err
		}
		return tx.Model(&schemas.Space{}).Where("id = ?", spaceID).
			Updates(map[string]any{"charged_to": userID, "billing_offer_to": nil, "billing_offer_end": nil}).Error
	})
}

// offeredTo reports whether the space holds an open billing offer to userID.
func offeredTo(space schemas.Space, userID int64, now time.Time) bool {
	return space.ChargedTo != nil && space.BillingOfferTo != nil && *space.BillingOfferTo == userID &&
		space.BillingOfferEnd != nil && space.BillingOfferEnd.After(now)
}

// isOwner reports whether userID is an owner of the space.
func isOwner(tx *gorm.DB, spaceID, userID int64) bool {
	var owners int64
	err := tx.Model(&schemas.SpaceMember{}).Where("space_id = ? AND user_id = ? AND role = 'owner'", spaceID, userID).Count(&owners).Error
	return err == nil && owners > 0
}

// payerOnly turns an update guarded on charged_to into a refusal when the
// caller is not the payer.
func payerOnly(result *gorm.DB) error {
	if result.Error != nil {
		return errors.Internal("failed to update billing", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.Forbidden("only the member who pays for this space can change its billing")
	}
	return nil
}
