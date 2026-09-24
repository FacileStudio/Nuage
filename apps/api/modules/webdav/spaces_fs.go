package webdav

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"

	"golang.org/x/net/webdav"
	"gorm.io/gorm"
)

// spacesFS is the read-only filesystem behind /webdav/spaces/, listing one
// entry per space the caller belongs to.
type spacesFS struct {
	db     *gorm.DB
	userID int64
}

// newSpacesFS builds the read-only spaces index filesystem for one user.
func newSpacesFS(db *gorm.DB, userID int64) webdav.FileSystem {
	return &spacesFS{db: db, userID: userID}
}

// indexTarget resolves a path inside the spaces index to the space id it names,
// or zero for the index root. The webdav walker probes children by their bare
// name, so "1" and "/1" must both address space 1.
func (s *spacesFS) indexTarget(ctx context.Context, name string) (int64, error) {
	if indexIsRoot(name) {
		return 0, nil
	}
	id, ok := indexSpaceID(name)
	if !ok {
		return 0, os.ErrNotExist
	}
	if _, err := resolveSpaceScope(ctx, s.db, s.userID, id); err != nil {
		return 0, err
	}
	return id, nil
}

// Stat reports the index root and each of the caller's spaces as directories.
func (s *spacesFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	id, err := s.indexTarget(ctx, name)
	if err != nil {
		return nil, err
	}
	if id == 0 {
		return &DirInfo{name: "/", modTime: time.Now()}, nil
	}
	return &DirInfo{name: strconv.FormatInt(id, 10), modTime: time.Now()}, nil
}

// OpenFile opens the index root, listing the caller's spaces as its children. A
// space entry opens empty: its contents live on the space mount, not here.
func (s *spacesFS) OpenFile(ctx context.Context, name string, _ int, _ os.FileMode) (webdav.File, error) {
	id, err := s.indexTarget(ctx, name)
	if err != nil {
		return nil, err
	}
	if id != 0 {
		return &spacesDir{modTime: time.Now()}, nil
	}
	ids, err := spaceaccess.MemberIDs(ctx, s.db, s.userID)
	if err != nil {
		return nil, err
	}
	children := make([]os.FileInfo, 0, len(ids))
	for _, spaceID := range ids {
		children = append(children, &DirInfo{name: strconv.FormatInt(spaceID, 10), modTime: time.Now()})
	}
	return &spacesDir{children: children, modTime: time.Now()}, nil
}

// Mkdir is refused: the spaces index is read-only by design.
func (s *spacesFS) Mkdir(context.Context, string, os.FileMode) error { return os.ErrPermission }

// RemoveAll is refused: the spaces index is read-only by design.
func (s *spacesFS) RemoveAll(context.Context, string) error { return os.ErrPermission }

// Rename is refused: the spaces index is read-only by design.
func (s *spacesFS) Rename(context.Context, string, string) error { return os.ErrPermission }
