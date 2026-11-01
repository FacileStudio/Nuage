package trash

import (
	"context"
	"log/slog"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/thumbnail"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

type storedObject struct {
	bucket string
	key    string
	prefix bool
}

// mayHaveThumbnails reports whether a file could have stored thumbnails: only
// images are rendered, and a file whose type was never resolved might be one.
// It spares a bucket listing per purged PDF or archive.
func mayHaveThumbnails(file schemas.File) bool {
	return strings.HasPrefix(file.MimeType, "image/") ||
		file.MimeType == "" || file.MimeType == "application/octet-stream"
}

// fileObjects lists what purging a file must remove from storage: the stored
// bytes, and every thumbnail of it by prefix, whatever its width or the content
// version it was rendered from.
func fileObjects(file schemas.File) []storedObject {
	objects := []storedObject{{bucket: file.Bucket, key: file.BucketKey}}
	if mayHaveThumbnails(file) {
		objects = append(objects, storedObject{bucket: file.Bucket, key: thumbnail.Prefix(file.ID), prefix: true})
	}
	return objects
}

func (s *Service) deleteObjects(ctx context.Context, objects []storedObject) {
	for _, object := range objects {
		if err := s.removeObject(ctx, object); err != nil {
			slog.Warn("trash: failed to delete object from storage", slog.String("bucket", object.bucket), slog.String("key", object.key), slog.Any("error", err))
		}
	}
}

func (s *Service) removeObject(ctx context.Context, object storedObject) error {
	store := s.storage.For(object.bucket)
	if object.prefix {
		return store.DeletePrefix(ctx, object.key)
	}
	return store.DeleteObject(ctx, object.key)
}
