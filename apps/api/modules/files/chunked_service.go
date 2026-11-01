package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/internal/spacebucket"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/internal/tokenscope"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const (
	finishedSessionRetention = 6 * time.Hour
	assemblingClaimGrace     = time.Hour
	sessionSweepInterval     = time.Hour
)

// initUpload opens a chunked-upload session. A restricted token that passes
// LinkedTo publishes idempotently: when its app already holds a live file under
// that key in the space, that file is returned and no session is created.
func (s *Service) initUpload(ctx context.Context, userID int64, req InitUploadRequest) (*schemas.UploadSession, *schemas.File, error) {
	if req.FileName == "" {
		return nil, nil, errors.Invalid("file_name is required")
	}
	if req.TotalSize <= 0 {
		return nil, nil, errors.Invalid("total_size must be positive")
	}
	req.MimeType = mimetype.Resolve(req.FileName, req.MimeType)

	if err := tokenscope.RequireSpace(ctx, req.SpaceID); err != nil {
		return nil, nil, err
	}
	req.OriginApp = tokenscope.Origin(ctx, req.OriginApp)

	if req.SpaceID != nil {
		if err := spaceaccess.Require(ctx, s.orm, *req.SpaceID, userID); err != nil {
			return nil, nil, err
		}
	}

	req.LinkedTo = strings.TrimSpace(req.LinkedTo)
	if tokenscope.From(ctx) != nil && req.LinkedTo != "" {
		if !validLinkedTo(req.LinkedTo) {
			return nil, nil, errors.Invalid("linked_to must be printable and at most 200 characters")
		}
		existing, err := findPublished(s.orm.WithContext(ctx), *req.SpaceID, req.OriginApp, req.LinkedTo)
		if err != nil || existing != nil {
			return nil, existing, err
		}
	} else {
		req.LinkedTo = ""
	}

	if req.FolderID != nil {
		if err := s.requireWritableFolder(ctx, userID, *req.FolderID, req.SpaceID); err != nil {
			return nil, nil, err
		}
	}

	if s.quota != nil {
		if err := s.quota.CheckQuota(ctx, userID, req.TotalSize); err != nil {
			return nil, nil, err
		}
	}

	session := &schemas.UploadSession{
		ID:        facile.NewID(),
		FileName:  req.FileName,
		MimeType:  req.MimeType,
		FolderID:  req.FolderID,
		OriginApp: req.OriginApp,
		LinkedTo:  req.LinkedTo,
		UserID:    userID,
		TotalSize: req.TotalSize,
		SpaceID:   req.SpaceID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.orm.WithContext(ctx).Create(session).Error; err != nil {
		return nil, nil, errors.Internal("failed to create upload session", err)
	}

	return session, nil, nil
}

func (s *Service) uploadChunk(ctx context.Context, userID int64, sessionID string, partNumber int, reader io.Reader, size int64) (*schemas.UploadChunk, error) {
	var session schemas.UploadSession
	if err := scopedSessions(ctx, s.orm.WithContext(ctx)).Where("id = ? AND user_id = ? AND status = 'pending'", sessionID, userID).First(&session).Error; err != nil {
		return nil, errors.NotFound("upload session not found or already completed")
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.Failed("upload session has expired")
	}

	if partNumber < 1 {
		return nil, errors.Invalid("part_number must be >= 1")
	}

	var existing schemas.UploadChunk
	if err := s.orm.WithContext(ctx).Where("session_id = ? AND part_number = ?", sessionID, partNumber).First(&existing).Error; err == nil {
		_ = s.storage.DeleteObject(ctx, existing.BucketKey)
		if err := s.orm.WithContext(ctx).Delete(&existing).Error; err != nil {
			return nil, errors.Internal("failed to replace existing chunk", err)
		}
	}

	bucketKey := fmt.Sprintf("chunks/%s/%d", sessionID, partNumber)

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if err := s.storage.PutObject(ctx, bucketKey, tee, size, "application/octet-stream"); err != nil {
		return nil, errors.Internal("failed to store chunk", err)
	}

	chunkHash := hex.EncodeToString(hasher.Sum(nil))

	info, err := s.storage.StatObject(ctx, bucketKey)
	if err != nil {
		return nil, errors.Internal("failed to verify chunk", err)
	}

	var uploadedSize int64
	if err := s.orm.WithContext(ctx).Model(&schemas.UploadChunk{}).
		Where("session_id = ?", sessionID).
		Select("COALESCE(SUM(size), 0)").
		Scan(&uploadedSize).Error; err != nil {
		_ = s.storage.DeleteObject(ctx, bucketKey)
		return nil, errors.Internal("failed to check uploaded size", err)
	}
	if uploadedSize+info.Size > session.TotalSize {
		_ = s.storage.DeleteObject(ctx, bucketKey)
		return nil, errors.TooLarge("chunks exceed the declared total_size")
	}

	chunk := &schemas.UploadChunk{
		SessionID:  sessionID,
		PartNumber: partNumber,
		BucketKey:  bucketKey,
		Size:       info.Size,
		Hash:       chunkHash,
	}

	if err := s.orm.WithContext(ctx).Create(chunk).Error; err != nil {
		_ = s.storage.DeleteObject(ctx, bucketKey)
		return nil, errors.Internal("failed to record chunk", err)
	}

	return chunk, nil
}

func (s *Service) completeUpload(ctx context.Context, userID int64, sessionID string, targetFileID *int64) (*schemas.File, bool, error) {
	claim := scopedSessions(ctx, s.orm.WithContext(ctx)).Model(&schemas.UploadSession{}).
		Where("id = ? AND user_id = ? AND status = 'pending' AND expires_at > ?", sessionID, userID, time.Now()).
		Update("status", "assembling")
	if claim.Error != nil {
		return nil, false, errors.Internal("failed to claim upload session", claim.Error)
	}
	if claim.RowsAffected == 0 {
		return nil, false, errors.NotFound("upload session not found, expired, or already completed")
	}

	var session schemas.UploadSession
	if err := s.orm.WithContext(ctx).Where("id = ?", sessionID).First(&session).Error; err != nil {
		s.releaseSessionClaim(sessionID)
		return nil, false, errors.Internal("failed to read upload session", err)
	}

	var chunks []schemas.UploadChunk
	if err := s.orm.WithContext(ctx).Where("session_id = ?", sessionID).Order("part_number asc").Find(&chunks).Error; err != nil {
		s.releaseSessionClaim(sessionID)
		return nil, false, errors.Internal("failed to list chunks", err)
	}

	if len(chunks) == 0 {
		s.releaseSessionClaim(sessionID)
		return nil, false, errors.Failed("no chunks uploaded")
	}

	var totalSize int64
	chunkKeys := make([]string, 0, len(chunks))
	for i, c := range chunks {
		if c.PartNumber != i+1 {
			s.releaseSessionClaim(sessionID)
			return nil, false, errors.Failed(fmt.Sprintf("missing chunk part %d", i+1))
		}
		totalSize += c.Size
		chunkKeys = append(chunkKeys, c.BucketKey)
	}

	if totalSize != session.TotalSize {
		s.releaseSessionClaim(sessionID)
		return nil, false, errors.Failed("uploaded chunks do not match the declared total_size")
	}

	if session.LinkedTo != "" && session.SpaceID != nil {
		existing, err := findPublished(s.orm.WithContext(ctx), *session.SpaceID, session.OriginApp, session.LinkedTo)
		if err != nil {
			s.releaseSessionClaim(sessionID)
			return nil, false, err
		}
		if existing != nil {
			s.finishSession(ctx, sessionID)
			return existing, false, nil
		}
	}

	if s.quota != nil {
		if err := s.quota.CheckQuota(ctx, userID, totalSize); err != nil {
			s.releaseSessionClaim(sessionID)
			return nil, false, err
		}
	}

	target, err := s.uploadTarget(ctx, userID, targetFileID)
	if err != nil {
		s.releaseSessionClaim(sessionID)
		return nil, false, err
	}
	store, bucket, err := s.uploadStore(ctx, session.SpaceID, target)
	if err != nil {
		s.releaseSessionClaim(sessionID)
		return nil, false, err
	}
	fileID := facile.NewID()
	bucketKey := fmt.Sprintf("%d/%s", userID, fileID)

	fileHash, err := store.AssembleChunks(ctx, s.storage, bucketKey, chunkKeys, totalSize, session.MimeType)
	if err != nil {
		s.releaseSessionClaim(sessionID)
		return nil, false, errors.Internal("failed to assemble file", err)
	}

	info, err := store.StatObject(ctx, bucketKey)
	if err != nil {
		_ = store.DeleteObject(ctx, bucketKey)
		s.releaseSessionClaim(sessionID)
		return nil, false, errors.Internal("failed to verify assembled file", err)
	}

	if target != nil {
		versioned, err := s.finishVersionedUpload(ctx, store, userID, sessionID, target, bucketKey, fileHash, session.MimeType, info.Size)
		return versioned, err == nil, err
	}

	record := &schemas.File{
		FacileID:   fileID,
		MimeType:   session.MimeType,
		Size:       info.Size,
		Hash:       fileHash,
		BucketKey:  bucketKey,
		Bucket:     bucket,
		FolderID:   session.FolderID,
		OriginApp:  session.OriginApp,
		LinkedTo:   session.LinkedTo,
		UploadedBy: userID,
		SpaceID:    session.SpaceID,
	}

	var existing *schemas.File
	if session.LinkedTo != "" && session.SpaceID != nil {
		var created bool
		existing, created, err = s.savePublished(ctx, record, session.FileName)
		if created {
			existing = nil
		}
	} else {
		existing, err = s.saveNewFile(ctx, record, session.FileName)
	}
	if err != nil {
		_ = store.DeleteObject(ctx, bucketKey)
		s.releaseSessionClaim(sessionID)
		return nil, false, err
	}
	if existing != nil {
		_ = store.DeleteObject(ctx, bucketKey)
		s.finishSession(ctx, sessionID)
		return existing, false, nil
	}

	s.finishSession(ctx, sessionID)

	if s.quota != nil {
		s.quota.UpdateUsage(ctx, userID, info.Size)
	}

	s.notifier.Notify(ctx, userID, "file.uploaded", nook.EventData{
		File: &nook.FileData{ID: record.ID, Name: record.Name, MimeType: record.MimeType, Size: record.Size},
	})

	if s.activity != nil {
		s.activity.Log(ctx, activity.Entry{
			UserID: userID, EventType: "file.uploaded", ResourceType: "file",
			ResourceID: record.ID, ResourceName: record.Name,
		})
	}

	return record, true, nil
}

// uploadStore picks the bucket the assembled bytes land in. A replacement keeps
// the bucket of the file it replaces; a new file goes to its space's bucket, or
// the shared one in personal scope.
func (s *Service) uploadStore(ctx context.Context, spaceID *int64, target *schemas.File) (*storage.Client, string, error) {
	if target != nil {
		return s.storage.For(target.Bucket), target.Bucket, nil
	}
	return spacebucket.Open(ctx, s.orm, s.storage, spaceID)
}

// uploadTarget resolves the existing file a completed session replaces, or nil
// when the bytes become a new file.
//
// A session that names an existing file replaces that file's content and keeps
// its name, folder, id and history. One that names nothing is named by the
// caller, under the folder's name lock, when the record is inserted.
func (s *Service) uploadTarget(ctx context.Context, userID int64, targetFileID *int64) (*schemas.File, error) {
	if targetFileID == nil {
		return nil, nil
	}
	if tokenscope.From(ctx) != nil {
		return nil, errors.Forbidden("this token cannot replace an existing file")
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).
		Where("id = ? AND uploaded_by = ? AND deleted_at IS NULL", *targetFileID, userID).
		First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("file not found")
		}
		return nil, errors.Internal("failed to read the file being updated", err)
	}
	return &record, nil
}

// finishVersionedUpload commits assembled bytes into an existing file and closes
// the session, the way reuploadFile closes a single-request replacement.
func (s *Service) finishVersionedUpload(ctx context.Context, store *storage.Client, userID int64, sessionID string, target *schemas.File, bucketKey, fileHash, mimeType string, size int64) (*schemas.File, error) {
	if err := s.commitUploadedVersion(ctx, userID, target.ID, bucketKey, fileHash, mimeType, size); err != nil {
		_ = store.DeleteObject(ctx, bucketKey)
		s.releaseSessionClaim(sessionID)
		return nil, err
	}

	s.finishSession(ctx, sessionID)
	go s.cleanOldVersions(target.ID, userID)

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ?", target.ID).First(&record).Error; err != nil {
		return nil, errors.Internal("failed to read updated file", err)
	}

	s.notifier.Notify(ctx, userID, "file.versioned", nook.EventData{
		File: &nook.FileData{ID: record.ID, Name: record.Name, MimeType: record.MimeType, Size: record.Size},
	})

	if s.activity != nil {
		s.activity.Log(ctx, activity.Entry{
			UserID: userID, EventType: "file.versioned", ResourceType: "file",
			ResourceID: record.ID, ResourceName: record.Name,
		})
	}

	return &record, nil
}

// finishSession marks a completed session and clears its chunks in the
// background, once its bytes are part of a file.
func (s *Service) finishSession(ctx context.Context, sessionID string) {
	if err := s.orm.WithContext(ctx).Model(&schemas.UploadSession{}).Where("id = ?", sessionID).Update("status", "completed").Error; err != nil {
		slog.Warn("chunked: failed to mark session completed", slog.String("session_id", sessionID), slog.Any("error", err))
	}

	go func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.storage.DeletePrefix(cleanCtx, fmt.Sprintf("chunks/%s/", sessionID)); err != nil {
			slog.Warn("chunked: failed to clean chunks from storage", slog.Any("error", err))
		}
		if err := s.orm.WithContext(cleanCtx).Where("session_id = ?", sessionID).Delete(&schemas.UploadChunk{}).Error; err != nil {
			slog.Warn("chunked: failed to clean chunk records", slog.Any("error", err))
		}
	}()
}

func (s *Service) releaseSessionClaim(sessionID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.orm.WithContext(ctx).Model(&schemas.UploadSession{}).
		Where("id = ? AND status = 'assembling'", sessionID).
		Update("status", "pending").Error; err != nil {
		slog.Warn("chunked: failed to release upload session claim", slog.String("session_id", sessionID), slog.Any("error", err))
	}
}

func (s *Service) getUploadStatus(ctx context.Context, userID int64, sessionID string) (*schemas.UploadSession, []schemas.UploadChunk, error) {
	var session schemas.UploadSession
	if err := scopedSessions(ctx, s.orm.WithContext(ctx)).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, nil, errors.NotFound("upload session not found")
	}

	var chunks []schemas.UploadChunk
	if err := s.orm.WithContext(ctx).Where("session_id = ?", sessionID).Order("part_number asc").Find(&chunks).Error; err != nil {
		return nil, nil, errors.Internal("failed to list chunks", err)
	}

	return &session, chunks, nil
}

func (s *Service) abortUpload(ctx context.Context, userID int64, sessionID string) error {
	var session schemas.UploadSession
	if err := scopedSessions(ctx, s.orm.WithContext(ctx)).Where("id = ? AND user_id = ? AND status = 'pending'", sessionID, userID).First(&session).Error; err != nil {
		return errors.NotFound("upload session not found")
	}

	if err := s.orm.WithContext(ctx).Model(&session).Update("status", "aborted").Error; err != nil {
		return errors.Internal("failed to abort upload session", err)
	}

	go func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.storage.DeletePrefix(cleanCtx, fmt.Sprintf("chunks/%s/", sessionID)); err != nil {
			slog.Warn("chunked: failed to clean chunks on abort", slog.Any("error", err))
		}
		if err := s.orm.WithContext(cleanCtx).Where("session_id = ?", sessionID).Delete(&schemas.UploadChunk{}).Error; err != nil {
			slog.Warn("chunked: failed to clean chunk records on abort", slog.Any("error", err))
		}
	}()

	return nil
}

// SweepExpiredSessions garbage-collects chunked upload sessions: expired
// pending sessions are aborted, then finished or stale sessions have their
// chunk objects, chunk rows, and session row removed. It returns the number
// of sessions fully removed.
func (s *Service) SweepExpiredSessions(ctx context.Context) (int, error) {
	now := time.Now()
	if err := s.orm.WithContext(ctx).Model(&schemas.UploadSession{}).
		Where("status = 'pending' AND expires_at < ?", now).
		Update("status", "aborted").Error; err != nil {
		return 0, errors.Internal("failed to expire pending upload sessions", err)
	}

	var sessions []schemas.UploadSession
	if err := s.orm.WithContext(ctx).
		Where("(status IN ('completed', 'aborted') AND created_at < ?) OR (status = 'assembling' AND expires_at < ?)",
			now.Add(-finishedSessionRetention), now.Add(-assemblingClaimGrace)).
		Find(&sessions).Error; err != nil {
		return 0, errors.Internal("failed to list sweepable upload sessions", err)
	}

	swept := 0
	for _, session := range sessions {
		if err := s.storage.DeletePrefix(ctx, fmt.Sprintf("chunks/%s/", session.ID)); err != nil {
			slog.Warn("chunked: sweep failed to delete chunk objects", slog.String("session_id", session.ID), slog.Any("error", err))
			continue
		}
		if err := s.orm.WithContext(ctx).Where("session_id = ?", session.ID).Delete(&schemas.UploadChunk{}).Error; err != nil {
			slog.Warn("chunked: sweep failed to delete chunk records", slog.String("session_id", session.ID), slog.Any("error", err))
			continue
		}
		if err := s.orm.WithContext(ctx).Delete(&schemas.UploadSession{}, "id = ?", session.ID).Error; err != nil {
			slog.Warn("chunked: sweep failed to delete session", slog.String("session_id", session.ID), slog.Any("error", err))
			continue
		}
		swept++
	}
	return swept, nil
}

// StartSessionSweeper runs SweepExpiredSessions on a ticker until ctx is
// cancelled.
func (s *Service) StartSessionSweeper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(sessionSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := s.SweepExpiredSessions(ctx)
				if err != nil {
					slog.Warn("chunked: session sweep failed", slog.Any("error", err))
				} else if n > 0 {
					slog.Info("chunked: swept upload sessions", slog.Int("count", n))
				}
			}
		}
	}()
}

func scopedSessions(ctx context.Context, query *gorm.DB) *gorm.DB {
	scope := tokenscope.From(ctx)
	if scope == nil {
		return query
	}
	return query.Where("space_id IN ?", append([]int64{0}, scope.SpaceIDs...))
}
