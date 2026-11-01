package files

import (
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	"github.com/FacileStudio/Nuage/apps/api/internal/thumbnail"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/go-chi/chi/v5"
)

const (
	thumbnailConcurrency = 4
	thumbnailSmall       = 320
	thumbnailLarge       = 640
	thumbnailMimeType    = "image/jpeg"
)

type thumbnailBody struct {
	*bytes.Reader
}

func (thumbnailBody) Close() error { return nil }

func thumbnailWidth(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("w")
	if raw == "" {
		return thumbnailSmall, nil
	}
	width, err := strconv.Atoi(raw)
	if err != nil || (width != thumbnailSmall && width != thumbnailLarge) {
		return 0, errors.Invalid("w must be 320 or 640")
	}
	return width, nil
}

func thumbnailKey(record *schemas.File, width int) (key, etag string) {
	version := record.Hash
	if version == "" {
		version = strconv.FormatInt(record.UpdatedAt.Unix(), 10)
	}
	if len(version) > 16 {
		version = version[:16]
	}
	etag = fmt.Sprintf(`"%s-%d"`, version, width)
	return thumbnail.Key(record.ID, version, width), etag
}

func isThumbnailable(record *schemas.File) bool {
	mime := mimetype.Resolve(record.Name, record.MimeType)
	return strings.HasPrefix(mime, "image/") && mime != "image/svg+xml" &&
		record.Size <= thumbnail.MaxBytes
}

// thumbnail returns the JPEG preview of a file at the given width. Renders are
// stored beside the original, in its own bucket, under a key that carries the
// content hash, so a reupload gets a fresh thumbnail and a purge can find it.
func (s *Service) thumbnail(ctx context.Context, userID int64, fileID string, width int) (*thumbnailBody, string, error) {
	record, err := s.getFile(ctx, userID, fileID)
	if err != nil {
		return nil, "", err
	}
	if !isThumbnailable(record) {
		return nil, "", errors.NotFound("no thumbnail for this file")
	}

	store := s.storage.For(record.Bucket)
	key, etag := thumbnailKey(record, width)
	if _, statErr := store.StatObject(ctx, key); statErr == nil {
		return s.storedThumbnail(ctx, record, key, etag)
	}

	source, err := store.GetObject(ctx, record.BucketKey)
	if err != nil {
		return nil, "", errors.Internal("failed to read file from storage", err)
	}
	defer source.Close()

	data, err := s.thumbs.Render(ctx, source, width)
	if stderrors.Is(err, thumbnail.ErrUnsupported) || stderrors.Is(err, thumbnail.ErrTooLarge) {
		return nil, "", errors.NotFound("no thumbnail for this file")
	}
	if err != nil {
		return nil, "", errors.Internal("failed to render thumbnail", err)
	}
	if err := store.PutObject(ctx, key, bytes.NewReader(data), int64(len(data)), thumbnailMimeType); err != nil {
		slog.Warn("files: failed to store thumbnail", slog.String("key", key), slog.Any("error", err))
	}
	return &thumbnailBody{bytes.NewReader(data)}, etag, nil
}

func (s *Service) storedThumbnail(ctx context.Context, record *schemas.File, key, etag string) (*thumbnailBody, string, error) {
	reader, err := s.storage.For(record.Bucket).GetObject(ctx, key)
	if err != nil {
		return nil, "", errors.Internal("failed to read thumbnail from storage", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, "", errors.Internal("failed to read thumbnail from storage", err)
	}
	return &thumbnailBody{bytes.NewReader(buf.Bytes())}, etag, nil
}

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
	width, err := thumbnailWidth(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	body, etag, err := h.service.thumbnail(r.Context(), userID, chi.URLParam(r, "id"), width)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	defer body.Close()

	w.Header().Set("Content-Type", thumbnailMimeType)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", downloadCacheControl(r))
	http.ServeContent(w, r, "", time.Time{}, body)
}
