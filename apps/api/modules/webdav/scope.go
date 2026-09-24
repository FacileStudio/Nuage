package webdav

import (
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"gorm.io/gorm"
)

// scope selects the rows one WebDAV mount may reach: a user's personal tree, or
// a single space the user is a member of.
type scope struct {
	userID  int64
	spaceID *int64
}

// folders narrows a folders query to the mount's scope.
func (s scope) folders(orm *gorm.DB) *gorm.DB {
	if s.spaceID != nil {
		return orm.Where("space_id = ?", *s.spaceID)
	}
	return orm.Where("owner_id = ? AND space_id IS NULL", s.userID)
}

// files narrows a files query to the mount's scope.
func (s scope) files(orm *gorm.DB) *gorm.DB {
	if s.spaceID != nil {
		return orm.Where("space_id = ?", *s.spaceID)
	}
	return orm.Where("uploaded_by = ? AND space_id IS NULL", s.userID)
}

// ownsFolder reports whether the mount may mutate the folder. In space scope any
// member may, the membership check having already run.
func (s scope) ownsFolder(f *schemas.Folder) bool {
	return s.spaceID != nil || f.OwnerID == s.userID
}

// ownsFile reports whether the mount may mutate the file. In space scope any
// member may, the membership check having already run.
func (s scope) ownsFile(f *schemas.File) bool {
	return s.spaceID != nil || f.UploadedBy == s.userID
}

// isSpace reports whether the scope is a space mount rather than personal.
func (s scope) isSpace() bool { return s.spaceID != nil }
