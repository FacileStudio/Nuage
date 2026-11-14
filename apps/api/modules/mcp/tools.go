package mcp

import (
	"context"
	stderrors "errors"
	"fmt"
	"strconv"

	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool names. Every name carries the app prefix, so a client holding many
// servers can tell them apart.
const (
	toolList        = "nuage_list_folder"
	toolSearch      = "nuage_search"
	toolRead        = "nuage_read_file"
	toolSave        = "nuage_save_file"
	toolMkdir       = "nuage_create_folder"
	toolMove        = "nuage_move"
	toolDelete      = "nuage_delete"
	toolShare       = "nuage_create_share"
	toolRevoke      = "nuage_revoke_share"
	toolListShares  = "nuage_list_shares"
	typeFile        = "file"
	typeFolder      = "folder"
	hintItem        = "Call nuage_list_folder or nuage_search for the ids you can use."
	hintShare       = "Call nuage_list_shares for the links you made."
	errTypeArgument = `type must be "file" or "folder"`
)

// Refusals for a caller Nuage's API would refuse too.
var (
	errNoCaller   = stderrors.New("this request carries no valid access token. Reconnect Nuage")
	errAppActor   = stderrors.New("a registered app can only reach its app folder, not the user's drive")
	errOtherActor = stderrors.New("this delegated token's client may not act in Nuage")
)

// toolset holds what the tool handlers share.
type toolset struct {
	Deps
	confirm *mcpkit.Confirmations
	appURL  string
}

// userOf reads the Nuage user id from the verified token, applying the
// rule RequireAuth applies to delegated tokens: an actor reaches the drive
// only when it is listed as a full-access client, and a registered app
// never does.
func (t *toolset) userOf(ctx context.Context, req *sdk.CallToolRequest) (int64, error) {
	identity, ok := mcpauth.Identity(req.Extra.TokenInfo)
	if !ok {
		return 0, errNoCaller
	}
	if identity.Actor == "" {
		return identity.UserID, nil
	}
	app, fullAccess, err := t.Actors.ActorAccess(ctx, identity.Actor)
	switch {
	case err != nil:
		return 0, mcpkit.ErrUnavailable
	case app != nil:
		return 0, errAppActor
	case !fullAccess:
		return 0, errOtherActor
	}
	return identity.UserID, nil
}

// optional turns an id argument into a pointer, nil for 0.
func optional(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

// checkType refuses a type other than file or folder.
func checkType(kind string) error {
	if kind != typeFile && kind != typeFolder {
		return stderrors.New(errTypeArgument)
	}
	return nil
}

// target is the pending-action reference of an item: "file:12".
func target(kind string, id int64) string {
	return kind + ":" + strconv.FormatInt(id, 10)
}

// shareURL is the public address of a link.
func (t *toolset) shareURL(token string) string {
	return fmt.Sprintf("%s/s/%s", t.appURL, token)
}
