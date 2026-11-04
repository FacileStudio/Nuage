package files

import (
	"fmt"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// maxOpenSessions is how many chunked uploads one user may have receiving
// chunks at once. Each one reserves its declared size against the plan.
const maxOpenSessions = 10

// openSession records a new session unless its user already has
// maxOpenSessions open. A per-user lock makes the count and the insert one
// step, so parallel inits cannot all slip under the cap.
func openSession(tx *gorm.DB, session *schemas.UploadSession) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		fmt.Sprintf("nuage:uploads:user:%d", session.UserID)).Error; err != nil {
		return errors.Internal("failed to lock upload sessions", err)
	}
	var open int64
	if err := tx.Model(&schemas.UploadSession{}).
		Where("user_id = ? AND status = 'pending' AND expires_at > ?", session.UserID, time.Now()).
		Count(&open).Error; err != nil {
		return errors.Internal("failed to count upload sessions", err)
	}
	if open >= maxOpenSessions {
		return errors.RateLimited("too many uploads in progress: finish or cancel one first")
	}
	if err := tx.Create(session).Error; err != nil {
		return errors.Internal("failed to create upload session", err)
	}
	return nil
}

// landingSpace is the space a completed session's bytes count against: the
// replaced file's when there is one, else the session's.
func landingSpace(spaceID *int64, target *schemas.File) *int64 {
	if target != nil {
		return target.SpaceID
	}
	return spaceID
}
