package mcp

import (
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// stamp renders a time for a model: RFC 3339 in UTC.
func stamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// fileItem renders a file, with its path when known.
func fileItem(f schemas.File, path string, full bool) item {
	out := item{
		Type: typeFile, ID: f.ID, Name: f.Name, Path: path, SpaceID: f.SpaceID, FolderID: f.FolderID,
		MimeType: f.MimeType, Size: f.Size, UpdatedAt: stamp(f.UpdatedAt),
	}
	if full {
		out.itemDetail = itemDetail{Hash: f.Hash, OwnerID: f.UploadedBy, CreatedAt: stamp(f.CreatedAt)}
	}
	return out
}

// folderItem renders a folder, with its path when known.
func folderItem(f schemas.Folder, path string, full bool) item {
	out := item{
		Type: typeFolder, ID: f.ID, Name: f.Name, Path: path, SpaceID: f.SpaceID, FolderID: f.ParentID,
		UpdatedAt: stamp(f.UpdatedAt),
	}
	if full {
		out.itemDetail = itemDetail{OwnerID: f.OwnerID, CreatedAt: stamp(f.CreatedAt)}
	}
	return out
}

// childPath is the path of an item named name inside parent, "/" being
// the root.
func childPath(parent, name string) string {
	if parent == "" || parent == "/" {
		return "/" + name
	}
	return parent + "/" + name
}

// parentPath is the path of the folder an item sits in, from paths, or the
// root when it sits at the root.
func parentPath(paths map[int64]string, folderID *int64) string {
	if folderID == nil {
		return "/"
	}
	return paths[*folderID]
}
