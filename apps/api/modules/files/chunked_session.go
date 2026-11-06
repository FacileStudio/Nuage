package files

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// maxOpenSessions is how many unexpired pending chunked uploads one user may
// hold, lapsed ones included, so idle sessions cannot pile up to be revived.
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

// admitChunk records a chunk's row and returns the object key of the part it
// replaced, for the caller to delete once the row is committed. A session that
// lapsed first re-admits its declared size under the plan lock, in the same
// transaction, so a revived upload cannot stage bytes the plan no longer has
// room for; the row it inserts makes the session active again.
func (s *Service) admitChunk(ctx context.Context, session schemas.UploadSession, chunk *schemas.UploadChunk) (string, error) {
	var replaced string
	record := func(tx *gorm.DB) error {
		if chunk == nil {
			return nil
		}
		var err error
		replaced, err = insertChunk(tx, session, chunk)
		return err
	}
	lapsed, err := schemas.SessionLapsed(s.orm.WithContext(ctx), session.ID)
	if err != nil {
		return "", errors.Internal("failed to read the upload session", err)
	}
	if !lapsed {
		return replaced, s.orm.WithContext(ctx).Transaction(record)
	}
	err = s.quota.Readmit(ctx, session.UserID, session.SpaceID, session.TotalSize, record)
	return replaced, err
}

// insertChunk records the chunk under the session's row lock, once it has
// checked the session is still receiving chunks: a chunk arriving while the
// session is assembled is refused rather than slipped into the file. Chunks of
// one session are counted one after the other, so together they never stage
// more than the declared total_size. A chunk replacing a part already received
// takes its row, and the replaced object's key is returned.
func insertChunk(tx *gorm.DB, session schemas.UploadSession, chunk *schemas.UploadChunk) (string, error) {
	var locked schemas.UploadSession
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", session.ID).Take(&locked).Error; err != nil {
		return "", err
	}
	if locked.Status != "pending" {
		return "", errors.NotFound("upload session not found or already completed")
	}
	var others int64
	if err := tx.Model(&schemas.UploadChunk{}).Where("session_id = ? AND part_number <> ?", session.ID, chunk.PartNumber).
		Select("COALESCE(SUM(size), 0)").Scan(&others).Error; err != nil {
		return "", err
	}
	if others > session.TotalSize-chunk.Size {
		return "", errors.TooLarge("chunks exceed the declared total_size")
	}
	var previous schemas.UploadChunk
	err := tx.Where("session_id = ? AND part_number = ?", session.ID, chunk.PartNumber).Limit(1).Find(&previous).Error
	if err != nil {
		return "", err
	}
	if previous.ID != 0 {
		if err := tx.Delete(&previous).Error; err != nil {
			return "", err
		}
	}
	return previous.BucketKey, tx.Create(chunk).Error
}

// recordFailure keeps a plan refusal or an API error as it is and wraps any
// other failure to record a chunk as an internal error.
func recordFailure(err error) error {
	var apiErr *errors.Error
	if stderrors.Is(err, forfait.ErrLimit) || stderrors.As(err, &apiErr) {
		return err
	}
	return errors.Internal("failed to record chunk", err)
}
