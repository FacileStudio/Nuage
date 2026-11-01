package thumbnail

import (
	"bytes"
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

const (
	// Small is the default thumbnail width in pixels.
	Small = 320
	// Large is the width for high-density screens.
	Large = 640
	// MimeType is the content type every thumbnail is stored and served as.
	MimeType = "image/jpeg"

	concurrency = 4
)

// Store renders thumbnails of files the caller has already been authorised to
// read, keeps each render in the file's own bucket, and serves it over HTTP.
type Store struct {
	generator *Generator
	storage   *storage.Client
}

// NewStore returns a Store that reads originals from, and caches renders in, the
// buckets behind client.
func NewStore(client *storage.Client) *Store {
	return &Store{generator: New(concurrency), storage: client}
}

type body struct {
	*bytes.Reader
}

func (body) Close() error { return nil }

// Width reads the `w` query parameter, accepting only the two sizes that exist.
func Width(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("w")
	if raw == "" {
		return Small, nil
	}
	width, err := strconv.Atoi(raw)
	if err != nil || (width != Small && width != Large) {
		return 0, errors.Invalid("w must be 320 or 640")
	}
	return width, nil
}

// Thumbnailable reports whether record is a raster image small enough to render.
func Thumbnailable(record *schemas.File) bool {
	mime := mimetype.Resolve(record.Name, record.MimeType)
	return strings.HasPrefix(mime, "image/") && mime != "image/svg+xml" && record.Size <= MaxBytes
}

// ObjectKey returns the storage key and ETag of record's thumbnail at width. The
// content hash is the version, so a reupload gets a new key.
func ObjectKey(record *schemas.File, width int) (key, etag string) {
	version := record.Hash
	if version == "" {
		version = strconv.FormatInt(record.UpdatedAt.Unix(), 10)
	}
	if len(version) > 16 {
		version = version[:16]
	}
	return Key(record.ID, version, width), `"` + version + "-" + strconv.Itoa(width) + `"`
}

// Serve writes record's thumbnail to w with the given Cache-Control. It answers
// 404 for anything that cannot be rendered, and never checks access: the caller
// must have authorised record first.
func (s *Store) Serve(w http.ResponseWriter, r *http.Request, record *schemas.File, width int, cacheControl string) error {
	data, etag, err := s.load(r.Context(), record, width)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", MimeType)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", cacheControl)
	http.ServeContent(w, r, "", time.Time{}, body{bytes.NewReader(data)})
	return nil
}

func (s *Store) load(ctx context.Context, record *schemas.File, width int) ([]byte, string, error) {
	if !Thumbnailable(record) {
		return nil, "", errors.NotFound("no thumbnail for this file")
	}
	bucket := s.storage.For(record.Bucket)
	key, etag := ObjectKey(record, width)
	if _, statErr := bucket.StatObject(ctx, key); statErr == nil {
		data, err := s.stored(ctx, record, key)
		return data, etag, err
	}

	source, err := bucket.GetObject(ctx, record.BucketKey)
	if err != nil {
		return nil, "", errors.Internal("failed to read file from storage", err)
	}
	defer source.Close()

	data, err := s.generator.Render(ctx, source, width)
	if stderrors.Is(err, ErrUnsupported) || stderrors.Is(err, ErrTooLarge) {
		return nil, "", errors.NotFound("no thumbnail for this file")
	}
	if err != nil {
		return nil, "", errors.Internal("failed to render thumbnail", err)
	}
	if err := bucket.PutObject(ctx, key, bytes.NewReader(data), int64(len(data)), MimeType); err != nil {
		slog.Warn("thumbnail: failed to store render", slog.String("key", key), slog.Any("error", err))
	}
	return data, etag, nil
}

func (s *Store) stored(ctx context.Context, record *schemas.File, key string) ([]byte, error) {
	reader, err := s.storage.For(record.Bucket).GetObject(ctx, key)
	if err != nil {
		return nil, errors.Internal("failed to read thumbnail from storage", err)
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return nil, errors.Internal("failed to read thumbnail from storage", err)
	}
	return buf.Bytes(), nil
}
