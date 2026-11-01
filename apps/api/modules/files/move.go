package files

import (
	"context"
	stderrors "errors"

	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// moveTarget verifies that folderID is a live folder the caller may move items
// into from spaceID. The folder must sit in the same space as the item, so a
// move never carries a file across buckets. In a space any member's folder is
// a valid target; in personal storage only the caller's own folders are.
func (s *Service) moveTarget(ctx context.Context, userID, folderID int64, spaceID *int64) error {
	var folder schemas.Folder
	err := s.orm.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", folderID).First(&folder).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return errors.NotFound("folder not found")
	}
	if err != nil {
		return errors.Internal("failed to verify folder", err)
	}
	if !sameSpace(folder.SpaceID, spaceID) {
		return errors.NotFound("folder not found")
	}
	if spaceID != nil {
		return spaceaccess.Require(ctx, s.orm, *spaceID, userID)
	}
	if folder.OwnerID != userID {
		return errors.NotFound("folder not found")
	}
	return nil
}

// isWithin reports whether candidateID is folderID or sits anywhere below it.
// UNION rather than UNION ALL keeps the walk finite even over a tree that
// already holds a cycle.
func (s *Service) isWithin(ctx context.Context, candidateID, folderID int64) (bool, error) {
	var found bool
	err := s.orm.WithContext(ctx).Raw(`
		WITH RECURSIVE ancestors AS (
			SELECT id, parent_id FROM folders WHERE id = ?
			UNION
			SELECT f.id, f.parent_id FROM folders f INNER JOIN ancestors a ON f.id = a.parent_id
		)
		SELECT EXISTS (SELECT 1 FROM ancestors WHERE id = ?)
	`, candidateID, folderID).Scan(&found).Error
	if err != nil {
		return false, errors.Internal("failed to resolve folder tree", err)
	}
	return found, nil
}

// folderParent resolves the parent_id update for moving folder into parentID,
// where zero means the root.
func (s *Service) folderParent(ctx context.Context, userID int64, folder *schemas.Folder, parentID int64) (any, error) {
	if parentID == 0 {
		return nil, nil
	}
	if err := s.moveTarget(ctx, userID, parentID, folder.SpaceID); err != nil {
		return nil, err
	}
	inside, err := s.isWithin(ctx, parentID, folder.ID)
	if err != nil {
		return nil, err
	}
	if inside {
		return nil, errors.Invalid("a folder cannot be moved into itself or one of its subfolders")
	}
	return parentID, nil
}

func sameSpace(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
