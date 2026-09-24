package webdav

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/tronc/errors"

	"gorm.io/gorm"
)

const spacesPrefix = "/webdav/spaces/"

// mountKind distinguishes the personal tree, the spaces index and a space mount.
type mountKind int

const (
	mountPersonal mountKind = iota
	mountIndex
	mountSpace
)

// mountPath is a parsed WebDAV URL: which mount it addresses and the prefix the
// webdav.Handler must strip.
type mountPath struct {
	kind    mountKind
	spaceID int64
	prefix  string
}

// parseMountPath splits a WebDAV URL into the mount it addresses. It matches
// whole path segments, never a string prefix, so /webdav/spaces/30 is space 30
// and never space 3 with a remainder of "0".
func parseMountPath(urlPath string) (mountPath, bool) {
	segments := strings.Split(strings.Trim(urlPath, "/"), "/")
	if len(segments) == 0 || segments[0] != "webdav" {
		return mountPath{}, false
	}
	if len(segments) == 1 || segments[1] != "spaces" {
		return mountPath{kind: mountPersonal, prefix: "/webdav"}, true
	}
	if len(segments) == 2 {
		return mountPath{kind: mountIndex, prefix: spacesPrefix}, true
	}
	id, err := strconv.ParseInt(segments[2], 10, 64)
	if err != nil {
		return mountPath{}, false
	}
	return mountPath{kind: mountSpace, spaceID: id, prefix: spacesPrefix + segments[2] + "/"}, true
}

// needsTrailingSlashRedirect reports whether a spaces URL is missing the
// trailing slash its mount prefix requires to be stripped correctly.
func needsTrailingSlashRedirect(urlPath string, m mountPath) bool {
	if m.kind == mountPersonal {
		return false
	}
	return urlPath == strings.TrimSuffix(m.prefix, "/")
}

// destinationEscapesMount reports whether a MOVE or COPY Destination addresses a
// different mount than the request. The webdav library would otherwise strip its
// own prefix from the target and mangle or reject it.
func destinationEscapesMount(m mountPath, destinationHeader string) bool {
	if destinationHeader == "" {
		return false
	}
	target, err := url.Parse(destinationHeader)
	if err != nil {
		return true
	}
	parsed, ok := parseMountPath(target.Path)
	if !ok {
		return true
	}
	return parsed.kind != m.kind || parsed.spaceID != m.spaceID
}

// resolveSpaceScope checks membership and returns the scope of a space mount. A
// non-member gets os.ErrNotExist, never a 403, so an authenticated stranger
// cannot probe which space ids exist.
func resolveSpaceScope(ctx context.Context, orm *gorm.DB, userID int64, spaceID int64) (scope, error) {
	if err := spaceaccess.Require(ctx, orm, spaceID, userID); err != nil {
		if errors.Status(err) == http.StatusForbidden {
			return scope{}, os.ErrNotExist
		}
		return scope{}, err
	}
	return scope{userID: userID, spaceID: &spaceID}, nil
}

// indexIsRoot reports whether a path inside the spaces index names the index
// root. The webdav walker hands the root to the filesystem as an empty name.
func indexIsRoot(name string) bool {
	return path.Clean("/"+name) == "/"
}

// indexSpaceID extracts the space id a path inside the spaces index names. The
// walker probes children by their bare name, so "1" and "/1" both address
// space 1.
func indexSpaceID(name string) (int64, bool) {
	trimmed := strings.TrimPrefix(path.Clean("/"+name), "/")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
