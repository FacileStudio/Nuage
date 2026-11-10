package mcp

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxShareDays caps how long a link made through MCP lasts.
const maxShareDays = 365

// errDays is the refusal for an expiry out of range.
var errDays = stderrors.New("expires_in_days must be 1 to 365, or omitted for no expiry")

// createShare is nuage_create_share: a public link, once the user
// confirms.
func (t *toolset) createShare(ctx context.Context, req *sdk.CallToolRequest, in shareInput) (*sdk.CallToolResult, actionOutput, error) {
	if mcpkit.Retrying(req) {
		return t.resumeShare(ctx, req)
	}
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, actionOutput{}, err
	}
	if err := checkType(in.Type); err != nil {
		return nil, actionOutput{}, err
	}
	if in.ExpiresInDays < 0 || in.ExpiresInDays > maxShareDays {
		return nil, actionOutput{}, errDays
	}
	file, folder, err := t.itemOf(ctx, userID, in.Type, in.ID, false)
	if err != nil {
		return nil, actionOutput{}, failure(err, hintItem)
	}
	result, outcome, err := t.confirm.Request(ctx, req, kindShare, shareTarget(in.Type, in.ID, in.ExpiresInDays), shareQuestion(file, folder, in.ExpiresInDays))
	if err != nil {
		return nil, actionOutput{}, failure(err, hintItem)
	}
	if result != nil {
		return result, actionOutput{}, nil
	}
	return nil, actionOutput{Status: outcome.Status, Type: in.Type, ID: in.ID, ConfirmURL: outcome.ConfirmURL, Message: waiting}, nil
}

// resumeShare handles the retry that carries the person's answer, and
// returns the new link.
func (t *toolset) resumeShare(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, actionOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, actionOutput{}, err
	}
	outcome, err := t.confirm.Resume(ctx, req, kindShare)
	if err != nil {
		return nil, actionOutput{}, failure(err, "Call nuage_create_share again to ask the user afresh.")
	}
	if outcome.Status == mcpkit.StatusDeclined {
		return nil, actionOutput{Status: mcpkit.StatusDeclined, Message: "The user declined. No link was made."}, nil
	}
	r, _ := refOf(outcome.Action)
	made, err := t.linkFor(ctx, userID, r, outcome.Action.CreatedAt)
	if err != nil {
		return nil, actionOutput{}, failure(err, hintShare)
	}
	return nil, actionOutput{Status: "shared", Type: r.kind, ID: r.id, Share: made, Message: "The user confirmed. Give them share.url."}, nil
}

// linkFor finds the link an action made: the person's newest link to its
// item made since the action was asked for.
func (t *toolset) linkFor(ctx context.Context, userID int64, r ref, since time.Time) (*share, error) {
	list, err := t.Sharing.ListMine(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, s := range list {
		if matches(s, r) && !s.CreatedAt.Before(since.Add(-time.Second)) {
			out := t.shareOf(s)
			return &out, nil
		}
	}
	return nil, mcpkit.ErrUnavailable
}

// matches reports whether a link points at the item r names.
func matches(s schemas.Share, r ref) bool {
	if r.kind == typeFile {
		return s.FileID != nil && *s.FileID == r.id
	}
	return s.FolderID != nil && *s.FolderID == r.id
}

// shareOf renders a link.
func (t *toolset) shareOf(s schemas.Share) share {
	out := share{ShareID: s.ID, URL: t.shareURL(s.Token), CreatedAt: stamp(s.CreatedAt)}
	switch {
	case s.FileID != nil:
		out.Type, out.ID = typeFile, *s.FileID
		if s.File != nil {
			out.Name = s.File.Name
		}
	case s.FolderID != nil:
		out.Type, out.ID = typeFolder, *s.FolderID
		if s.Folder != nil {
			out.Name = s.Folder.Name
		}
	}
	if s.ExpiresAt != nil {
		out.ExpiresAt = stamp(*s.ExpiresAt)
	}
	return out
}

// revokeShare is nuage_revoke_share. Revoking only takes access away, so
// it needs no confirmation.
func (t *toolset) revokeShare(ctx context.Context, req *sdk.CallToolRequest, in shareRef) (*sdk.CallToolResult, actionOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, actionOutput{}, err
	}
	if err := t.Sharing.Revoke(ctx, userID, in.ShareID); err != nil {
		return nil, actionOutput{}, failure(err, hintShare)
	}
	return nil, actionOutput{Status: "revoked", ID: in.ShareID, Message: "The link no longer works."}, nil
}

// listShares is nuage_list_shares.
func (t *toolset) listShares(ctx context.Context, req *sdk.CallToolRequest, in sharesInput) (*sdk.CallToolResult, sharesOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, sharesOutput{}, err
	}
	var at position
	if err := mcpkit.DecodeCursor(in.Cursor, &at); err != nil || at.Offset < 0 {
		return nil, sharesOutput{}, mcpkit.ErrCursor
	}
	list, err := t.Sharing.ListMine(ctx, userID)
	if err != nil {
		return nil, sharesOutput{}, failure(err, "")
	}
	if at.Offset > len(list) {
		return nil, sharesOutput{}, mcpkit.ErrCursor
	}
	rest := list[at.Offset:min(at.Offset+clamp(in.Limit, defaultListLimit, maxListLimit), len(list))]
	shares := make([]share, len(rest))
	for i, s := range rest {
		shares[i] = t.shareOf(s)
	}
	keep := mcpkit.FitCount(len(shares), func(n int) int {
		return mcpkit.EncodedSize(sharesOutput{Shares: shares[:n], Total: len(list), NextCursor: "next-cursor-placeholder"})
	})
	out := sharesOutput{Shares: shares[:keep], Total: len(list)}
	if at.Offset+keep < len(list) {
		out.NextCursor = mcpkit.EncodeCursor(position{Offset: at.Offset + keep})
	}
	return nil, out, nil
}
