package mcp

import (
	"context"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// withPath sets the path of an item that sits in parentID.
func (t *toolset) withPath(ctx context.Context, it item, parentID *int64) (item, error) {
	var ids []int64
	if parentID != nil {
		ids = []int64{*parentID}
	}
	paths, err := t.Files.FolderPaths(ctx, ids)
	if err != nil {
		return item{}, err
	}
	it.Path = childPath(parentPath(paths, parentID), it.Name)
	return it, nil
}

// itemOf is the record an action targets, with the reach of the REST route
// that runs it: any file the person can open, and a folder they created
// when owned is set (trashing), else any folder they can open (sharing).
func (t *toolset) itemOf(ctx context.Context, userID int64, kind string, id int64, owned bool) (*schemas.File, *schemas.Folder, error) {
	switch {
	case kind == typeFile:
		f, err := t.Files.File(ctx, userID, id)
		return f, nil, err
	case owned:
		f, err := t.Files.OwnedFolder(ctx, userID, id)
		return nil, f, err
	default:
		f, err := t.Files.Folder(ctx, userID, id)
		return nil, f, err
	}
}

// placeOf resolves where a new item goes: inside folderID, which then
// decides the space, or at the root of spaceID or of the personal drive.
func (t *toolset) placeOf(ctx context.Context, userID, folderID, spaceID int64) (*int64, *int64, error) {
	if folderID == 0 {
		return nil, optional(spaceID), nil
	}
	folder, err := t.Files.Folder(ctx, userID, folderID)
	if err != nil {
		return nil, nil, err
	}
	return &folder.ID, folder.SpaceID, nil
}
