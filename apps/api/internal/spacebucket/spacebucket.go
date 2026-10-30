package spacebucket

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Open returns the store a new object must be written to, and the name to
// record on its row. A nil space is personal scope and gets the shared bucket
// with an empty name. A space gets its own bucket, created here if it is
// missing, so a write never depends on the bucket having been made earlier.
func Open(ctx context.Context, orm *gorm.DB, store *storage.Client, spaceID *int64) (*storage.Client, string, error) {
	if spaceID == nil {
		return store, "", nil
	}

	var space schemas.Space
	if err := orm.WithContext(ctx).Select("facile_id").Where("id = ?", *spaceID).First(&space).Error; err != nil {
		return nil, "", errors.Internal("failed to read space", err)
	}

	name := storage.SpaceBucket(space.FacileID)
	spaceStore := store.For(name)
	if err := spaceStore.EnsureBucketOnce(ctx); err != nil {
		return nil, "", errors.Internal("failed to prepare space storage", err)
	}
	return spaceStore, name, nil
}
