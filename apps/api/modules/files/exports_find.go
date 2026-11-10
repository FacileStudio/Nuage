package files

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

// Folder reads a live folder the person can open: one of their personal
// folders, or any folder of a space they belong to. Listing it then goes
// through ListFiles and ListFolders, which check the space again.
func (s *Service) Folder(ctx context.Context, userID, folderID int64) (*schemas.Folder, error) {
	reach, err := s.reach(ctx, userID, "owner_id")
	if err != nil {
		return nil, err
	}
	var folder schemas.Folder
	err = reach.Where("id = ? AND deleted_at IS NULL", folderID).First(&folder).Error
	if stderrors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.NotFound("folder not found")
	}
	if err != nil {
		return nil, errors.Internal("failed to read folder", err)
	}
	return &folder, nil
}

// SearchHits are the files and folders whose names match a search.
type SearchHits struct {
	Files   []schemas.File
	Folders []schemas.Folder
}

// SearchNames finds the live files and folders whose name holds query,
// ignoring case, newest first, within the personal drive and every space
// the person belongs to, or within one space when spaceID is set. limit caps
// each of the two lists.
func (s *Service) SearchNames(ctx context.Context, userID int64, query string, spaceID *int64, limit int) (SearchHits, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	if spaceID != nil {
		if err := spaceaccess.Require(ctx, s.orm, *spaceID, userID); err != nil {
			return SearchHits{}, err
		}
	}
	files, err := s.reach(ctx, userID, "uploaded_by")
	if err != nil {
		return SearchHits{}, err
	}
	folders, err := s.reach(ctx, userID, "owner_id")
	if err != nil {
		return SearchHits{}, err
	}
	var hits SearchHits
	files = files.Where("deleted_at IS NULL AND lower(name) LIKE ?", pattern)
	folders = folders.Where("deleted_at IS NULL AND lower(name) LIKE ?", pattern)
	if spaceID != nil {
		files, folders = files.Where("space_id = ?", *spaceID), folders.Where("space_id = ?", *spaceID)
	}
	if err := files.Order("updated_at desc, id desc").Limit(limit).Find(&hits.Files).Error; err != nil {
		return SearchHits{}, errors.Internal("search failed", err)
	}
	if err := folders.Order("updated_at desc, id desc").Limit(limit).Find(&hits.Folders).Error; err != nil {
		return SearchHits{}, errors.Internal("search failed", err)
	}
	return hits, nil
}

// FolderPaths returns the path from the root, such as "/Clients/Acme", of
// each folder id given. The ids must be folders the person reached.
func (s *Service) FolderPaths(ctx context.Context, ids []int64) (map[int64]string, error) {
	paths := map[int64]string{}
	if len(ids) == 0 {
		return paths, nil
	}
	var rows []struct {
		ID   int64
		Path string
	}
	err := s.orm.WithContext(ctx).Raw(`
		WITH RECURSIVE up AS (
			SELECT id AS start, parent_id, name::text AS path FROM folders WHERE id IN ?
			UNION ALL
			SELECT up.start, f.parent_id, f.name || '/' || up.path FROM folders f JOIN up ON f.id = up.parent_id
		)
		SELECT start AS id, '/' || path AS path FROM up WHERE parent_id IS NULL
	`, ids).Scan(&rows).Error
	if err != nil {
		return nil, errors.Internal("failed to resolve folder paths", err)
	}
	for _, row := range rows {
		paths[row.ID] = row.Path
	}
	return paths, nil
}

// reach scopes a files or folders query to what the person can open: their
// personal items, keyed on ownerColumn, and the items of their spaces. It is
// the reach GET /files/{id} applies.
func (s *Service) reach(ctx context.Context, userID int64, ownerColumn string) (*gorm.DB, error) {
	spaceIDs, err := spaceaccess.MemberIDs(ctx, s.orm, userID)
	if err != nil {
		return nil, err
	}
	return s.orm.WithContext(ctx).Where(
		s.orm.Where(ownerColumn+" = ? AND space_id IS NULL", userID).Or("space_id IN ?", append([]int64{0}, spaceIDs...)),
	), nil
}
