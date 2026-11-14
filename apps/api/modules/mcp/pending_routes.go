package mcp

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/tronc/errors"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// webActor marks, in the audit, a confirmation made on the confirm page.
const webActor = "nuage-web"

// pendingUser is the signed-in person of a confirm page request. Only the
// person's own browser session passes (mcpauth.RequireOwnSession): a bearer
// of any kind, a CLI or mobile session, an API token or a delegated token
// must never confirm what an agent asked for.
func pendingUser(r *http.Request) (string, error) {
	if _, err := mcpauth.RequireOwnSession(r); err != nil {
		return "", err
	}
	identity, found := authcontext.IdentityFromContext(r.Context())
	if !found {
		return "", errors.Unauthorized("missing auth")
	}
	return identity.UserID, nil
}

// describe adds the file or folder an action targets, as it is now, and
// the days a link would last.
func (t *toolset) describe(ctx context.Context, item mcpkit.PendingItem, a mcpkit.Action) PendingItem {
	out := PendingItem{PendingItem: item}
	if r, err := refOf(a); err == nil {
		out.Days, out.Item = r.days, t.pendingFile(ctx, r, a.Kind == kindTrash)
	}
	return out
}

// done is the body answered after a confirmed run: the link a share made,
// or the trashing.
func (t *toolset) done(ctx context.Context, o mcpkit.Outcome) mcpkit.PendingResult {
	if o.Action.Kind != kindShare {
		return mcpkit.PendingResult{Status: "trashed"}
	}
	result := mcpkit.PendingResult{Status: "shared"}
	if ref, err := refOf(o.Action); err == nil {
		if made, err := t.linkFor(ctx, ref.userID, ref, o.Action.CreatedAt); err == nil {
			result.URL = made.URL
		}
	}
	return result
}

// pendingFile is the item an action targets, or nil once it is gone.
func (t *toolset) pendingFile(ctx context.Context, r ref, owned bool) *PendingFile {
	file, folder, err := t.itemOf(ctx, r.userID, r.kind, r.id, owned)
	switch {
	case err != nil:
		return nil
	case file != nil:
		return &PendingFile{Type: typeFile, ID: file.ID, Name: file.Name, MimeType: file.MimeType, Size: file.Size}
	default:
		return &PendingFile{Type: typeFolder, ID: folder.ID, Name: folder.Name}
	}
}
