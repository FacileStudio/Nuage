package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/modules/sharing"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// The two sensitive actions: moving to the trash, and publishing a link
// anyone can open, which is how content leaves the user's control.
const (
	kindTrash = "trash"
	kindShare = "share"
)

// errTarget is the refusal for a pending action whose target is malformed.
var errTarget = errors.NotFound("this action names no file or folder")

// ref is what a pending action targets.
type ref struct {
	userID int64
	kind   string
	id     int64
	days   int
}

// kinds are the sensitive actions. Each fingerprints its item, so an edit
// after the question closes the action instead of acting on content the
// person did not see.
func (t *toolset) kinds() map[string]mcpkit.Kind {
	return map[string]mcpkit.Kind{
		kindTrash: {Tool: toolDelete, Fingerprint: t.fingerprint(true), Run: t.runTrash},
		kindShare: {Tool: toolShare, Fingerprint: t.fingerprint(false), Run: t.runShare},
	}
}

// shareTarget is the target of a share action: "file:12:30", with the
// days the link lasts, 0 for no expiry.
func shareTarget(kind string, id int64, days int) string {
	return target(kind, id) + ":" + strconv.Itoa(days)
}

// refOf parses an action's user and target.
func refOf(a mcpkit.Action) (ref, error) {
	parts := strings.Split(a.Target, ":")
	userID, errUser := strconv.ParseInt(a.UserID, 10, 64)
	if errUser != nil || len(parts) < 2 || checkType(parts[0]) != nil {
		return ref{}, errTarget
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ref{}, errTarget
	}
	r := ref{userID: userID, kind: parts[0], id: id}
	if len(parts) == 3 {
		if r.days, err = strconv.Atoi(parts[2]); err != nil {
			return ref{}, errTarget
		}
	}
	return r, nil
}

// fingerprint digests the targeted item as it is now, through the reach of
// the action's route.
func (t *toolset) fingerprint(owned bool) func(context.Context, mcpkit.Action) (string, error) {
	return func(ctx context.Context, a mcpkit.Action) (string, error) {
		r, err := refOf(a)
		if err != nil {
			return "", err
		}
		file, folder, err := t.itemOf(ctx, r.userID, r.kind, r.id, owned)
		if err != nil {
			return "", err
		}
		return digest(file, folder), nil
	}
}

// digest fingerprints a file or a folder.
func digest(file *schemas.File, folder *schemas.Folder) string {
	var fields []string
	if file != nil {
		fields = []string{"file", strconv.FormatInt(file.ID, 10), file.Name, file.Hash, ptr(file.FolderID), ptr(file.SpaceID), file.UpdatedAt.UTC().Format(time.RFC3339Nano)}
	} else {
		fields = []string{"folder", strconv.FormatInt(folder.ID, 10), folder.Name, ptr(folder.ParentID), ptr(folder.SpaceID), folder.UpdatedAt.UTC().Format(time.RFC3339Nano)}
	}
	sum := sha256.Sum256([]byte(strings.Join(fields, "\x00")))
	return hex.EncodeToString(sum[:16])
}

// ptr renders an optional id.
func ptr(id *int64) string {
	if id == nil {
		return "-"
	}
	return strconv.FormatInt(*id, 10)
}

// runTrash moves the targeted item to the trash.
func (t *toolset) runTrash(ctx context.Context, a mcpkit.Action) error {
	r, err := refOf(a)
	if err != nil {
		return err
	}
	if r.kind == typeFile {
		return t.Files.TrashFile(ctx, r.userID, r.id)
	}
	return t.Files.TrashFolder(ctx, r.userID, r.id)
}

// runShare creates the public link, in the item's own space.
func (t *toolset) runShare(ctx context.Context, a mcpkit.Action) error {
	r, err := refOf(a)
	if err != nil {
		return err
	}
	file, folder, err := t.itemOf(ctx, r.userID, r.kind, r.id, false)
	if err != nil {
		return err
	}
	req := sharing.CreateShareRequest{}
	if file != nil {
		req.FileID, req.SpaceID = &file.ID, file.SpaceID
	} else {
		req.FolderID, req.SpaceID = &folder.ID, folder.SpaceID
	}
	if r.days > 0 {
		expires := time.Now().Add(time.Duration(r.days) * 24 * time.Hour).UTC().Format(time.RFC3339)
		req.ExpiresAt = &expires
	}
	_, err = t.Sharing.Create(ctx, r.userID, req)
	return err
}
