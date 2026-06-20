package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

func (s *Service) initUpload(ctx context.Context, userID int64, req InitUploadRequest) (*schemas.UploadSession, error) {
	if req.FileName == "" {
		return nil, errors.Invalid("file_name is required")
	}
	if req.TotalSize <= 0 {
		return nil, errors.Invalid("total_size must be positive")
	}
	if req.MimeType == "" {
		req.MimeType = "application/octet-stream"
	}

	if req.FolderID != nil {
		var folder schemas.Folder
		if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", *req.FolderID).First(&folder).Error; err != nil {
			return nil, errors.NotFound("folder not found")
		}
	}

	session := &schemas.UploadSession{
		ID:        facile.NewID(),
		FileName:  req.FileName,
		MimeType:  req.MimeType,
		FolderID:  req.FolderID,
		OriginApp: req.OriginApp,
		UserID:    userID,
		TotalSize: req.TotalSize,
		SpaceID:   req.SpaceID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.orm.WithContext(ctx).Create(session).Error; err != nil {
		return nil, errors.Internal("failed to create upload session", err)
	}

	return session, nil
}

func (s *Service) uploadChunk(ctx context.Context, userID int64, sessionID string, partNumber int, reader io.Reader, size int64) (*schemas.UploadChunk, error) {
	var session schemas.UploadSession
	if err := s.orm.WithContext(ctx).Where("id = ? AND user_id = ? AND status = 'pending'", sessionID, userID).First(&session).Error; err != nil {
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
		s.orm.WithContext(ctx).Delete(&existing)
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

func (s *Service) completeUpload(ctx context.Context, userID int64, sessionID string) (*schemas.File, error) {
	var session schemas.UploadSession
	if err := s.orm.WithContext(ctx).Where("id = ? AND user_id = ? AND status = 'pending'", sessionID, userID).First(&session).Error; err != nil {
		return nil, errors.NotFound("upload session not found or already completed")
	}

	var chunks []schemas.UploadChunk
	if err := s.orm.WithContext(ctx).Where("session_id = ?", sessionID).Order("part_number asc").Find(&chunks).Error; err != nil {
		return nil, errors.Internal("failed to list chunks", err)
	}

	if len(chunks) == 0 {
		return nil, errors.Failed("no chunks uploaded")
	}

	var totalSize int64
	chunkKeys := make([]string, 0, len(chunks))
	for _, c := range chunks {
		totalSize += c.Size
		chunkKeys = append(chunkKeys, c.BucketKey)
	}

	if s.quota != nil {
		if err := s.quota.CheckQuota(ctx, userID, totalSize); err != nil {
			return nil, err
		}
	}

	name := s.deduplicateFileName(ctx, session.FileName, session.FolderID)
	fileID := facile.NewID()
	bucketKey := fmt.Sprintf("%d/%s/%s", userID, fileID, name)

	if err := s.storage.AssembleChunks(ctx, bucketKey, chunkKeys, totalSize, session.MimeType); err != nil {
		return nil, errors.Internal("failed to assemble file", err)
	}

	info, err := s.storage.StatObject(ctx, bucketKey)
	if err != nil {
		_ = s.storage.DeleteObject(ctx, bucketKey)
		return nil, errors.Internal("failed to verify assembled file", err)
	}

	record := &schemas.File{
		FacileID:   fileID,
		Name:       name,
		MimeType:   session.MimeType,
		Size:       info.Size,
		BucketKey:  bucketKey,
		FolderID:   session.FolderID,
		OriginApp:  session.OriginApp,
		UploadedBy: userID,
		SpaceID:    session.SpaceID,
	}

	if err := s.orm.WithContext(ctx).Create(record).Error; err != nil {
		_ = s.storage.DeleteObject(ctx, bucketKey)
		return nil, errors.Internal("failed to save file record", err)
	}

	s.orm.WithContext(ctx).Model(&session).Update("status", "completed")

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

	return record, nil
}

func (s *Service) getUploadStatus(ctx context.Context, userID int64, sessionID string) (*schemas.UploadSession, []schemas.UploadChunk, error) {
	var session schemas.UploadSession
	if err := s.orm.WithContext(ctx).Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
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
	if err := s.orm.WithContext(ctx).Where("id = ? AND user_id = ? AND status = 'pending'", sessionID, userID).First(&session).Error; err != nil {
		return errors.NotFound("upload session not found")
	}

	s.orm.WithContext(ctx).Model(&session).Update("status", "aborted")

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
