package mcp

import (
	"github.com/FacileStudio/tronc/apiref"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// Documentation describes the confirm page routes for the API reference.
// The MCP endpoint itself is outside /api: POST /mcp speaks the Model
// Context Protocol to any MCP client holding a Registre access token for
// this server, and /.well-known/oauth-protected-resource/mcp describes how
// to get one.
var Documentation = apiref.Module{
	Name: "mcp",
	Description: "Sensitive actions MCP clients asked for, waiting for the person to confirm them: moving a file or " +
		"folder to the trash, and creating a public link. A client that cannot ask the person itself hands them a " +
		"link to the pending page, which uses these routes. The MCP endpoint is POST /mcp, outside /api.",
	Routes: pendingRoutes(),
}

// pendingRoutes documents the three confirm page routes.
func pendingRoutes() []apiref.Route {
	return mcpkit.PendingRoutes[PendingItem](mcpkit.PendingDoc{
		List: "Open actions, newest first, with their file or folder as it is now. state is ready, changed " +
			"(the item was edited after the request, so it cannot be confirmed) or missing.",
		Confirm: "Runs the action once: trashes the item, or creates the public link and answers its url. An " +
			"item edited since the request is refused with 412 and the action closes.",
		Dismiss: "Closes the action without running it.",
		Changed: "The item changed after the request.",
	})
}
