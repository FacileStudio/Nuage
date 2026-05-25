package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/errors"
	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"gorm.io/gorm"
)

const defaultMaxVersions = 5

func (s *Service) reuploadFile(ctx context.Context, userID int64, fileID string, reader io.Reader, size int64, mimeType string) (*schemas.File, error) {
	id, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid file id")
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("file not found")
		}
		return nil, errors.Internal("failed to read file", err)
	}

	var maxVersion int
	s.orm.WithContext(ctx).Model(&schemas.FileVersion{}).
		Where("file_id = ?", id).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)

	version := &schemas.FileVersion{
		FileID:    record.ID,
		Version:   maxVersion + 1,
		BucketKey: record.BucketKey,
		Hash:      record.Hash,
		Size:      record.Size,
		CreatedBy: userID,
	}

	if err := s.orm.WithContext(ctx).Create(version).Error; err != nil {
		return nil, errors.Internal("failed to create version record", err)
	}

	newFacileID := facile.NewID()
	newBucketKey := fmt.Sprintf("%d/%s/%s", userID, newFacileID, record.Name)

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)

	if err := s.storage.PutObject(ctx, newBucketKey, tee, size, mimeType); err != nil {
		return nil, errors.Internal("failed to upload new version", err)
	}

	fileHash := hex.EncodeToString(hasher.Sum(nil))
	info, err := s.storage.StatObject(ctx, newBucketKey)
	if err != nil {
		return nil, errors.Internal("failed to stat new version", err)
	}

	updates := map[string]any{
		"bucket_key": newBucketKey,
		"hash":       fileHash,
		"size":       info.Size,
		"mime_type":  mimeType,
		"updated_at": time.Now(),
	}

	if err := s.orm.WithContext(ctx).Model(&schemas.File{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, errors.Internal("failed to update file record", err)
	}

	go s.cleanOldVersions(record.ID)

	if err := s.orm.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
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

func (s *Service) listVersions(ctx context.Context, fileID string) ([]schemas.FileVersion, error) {
	id, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid file id")
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("file not found")
		}
		return nil, errors.Internal("failed to read file", err)
	}

	var versions []schemas.FileVersion
	if err := s.orm.WithContext(ctx).Where("file_id = ?", id).Order("version desc").Find(&versions).Error; err != nil {
		return nil, errors.Internal("failed to list versions", err)
	}

	return versions, nil
}

func (s *Service) restoreVersion(ctx context.Context, userID int64, fileID string, versionID string) (*schemas.File, error) {
	fid, err := strconv.ParseInt(fileID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid file id")
	}
	vid, err := strconv.ParseInt(versionID, 10, 64)
	if err != nil {
		return nil, errors.Invalid("invalid version id")
	}

	var record schemas.File
	if err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", fid).First(&record).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("file not found")
		}
		return nil, errors.Internal("failed to read file", err)
	}

	var version schemas.FileVersion
	if err := s.orm.WithContext(ctx).Where("id = ? AND file_id = ?", vid, fid).First(&version).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.NotFound("version not found")
		}
		return nil, errors.Internal("failed to read version", err)
	}

	var maxVersion int
	s.orm.WithContext(ctx).Model(&schemas.FileVersion{}).
		Where("file_id = ?", fid).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)

	currentVersion := &schemas.FileVersion{
		FileID:    record.ID,
		Version:   maxVersion + 1,
		BucketKey: record.BucketKey,
		Hash:      record.Hash,
		Size:      record.Size,
		CreatedBy: userID,
	}
	if err := s.orm.WithContext(ctx).Create(currentVersion).Error; err != nil {
		return nil, errors.Internal("failed to save current as version", err)
	}

	newBucketKey := fmt.Sprintf("%d/%s/%s", userID, facile.NewID(), record.Name)
	if err := s.storage.CopyObject(ctx, version.BucketKey, newBucketKey); err != nil {
		return nil, errors.Internal("failed to restore version from storage", err)
	}

	info, err := s.storage.StatObject(ctx, newBucketKey)
	if err != nil {
		return nil, errors.Internal("failed to stat restored file", err)
	}

	updates := map[string]any{
		"bucket_key": newBucketKey,
		"hash":       version.Hash,
		"size":       info.Size,
		"updated_at": time.Now(),
	}

	if err := s.orm.WithContext(ctx).Model(&schemas.File{}).Where("id = ?", fid).Updates(updates).Error; err != nil {
		return nil, errors.Internal("failed to update file record", err)
	}

	if err := s.orm.WithContext(ctx).Where("id = ?", fid).First(&record).Error; err != nil {
		return nil, errors.Internal("failed to read restored file", err)
	}

	go s.cleanOldVersions(fid)

	return &record, nil
}

func (s *Service) cleanOldVersions(fileID int64) {
	ctx := context.Background()

	maxVersions := defaultMaxVersions
	var setting schemas.Setting
	if err := s.orm.WithContext(ctx).Where("key = ?", "max_file_versions").First(&setting).Error; err == nil {
		if n, err := strconv.Atoi(setting.Value); err == nil && n > 0 {
			maxVersions = n
		}
	}

	var versions []schemas.FileVersion
	s.orm.WithContext(ctx).Where("file_id = ?", fileID).Order("version desc").Find(&versions)

	if len(versions) <= maxVersions {
		return
	}

	toDelete := versions[maxVersions:]
	for _, v := range toDelete {
		_ = s.storage.DeleteObject(ctx, v.BucketKey)
		s.orm.WithContext(ctx).Delete(&v)
	}
}

func mapVersion(v schemas.FileVersion) VersionResponse {
	return VersionResponse{
		ID:        v.ID,
		FileID:    v.FileID,
		Version:   v.Version,
		Hash:      v.Hash,
		Size:      v.Size,
		CreatedBy: v.CreatedBy,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339),
	}
}
