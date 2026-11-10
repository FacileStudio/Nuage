package mcp

import "github.com/FacileStudio/tronc/apiref"

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
	unauthorized := apiref.Error{Status: 401, Code: "unauthenticated", Description: "The request carries no valid session."}
	delegated := apiref.Error{Status: 403, Code: "permission_denied", Description: "Only the person's own session can confirm actions."}
	gone := apiref.Error{Status: 404, Code: "not_found", Description: "The action ran, was dismissed or expired."}
	idParam := []apiref.Field{{Name: "id", Type: "string", Description: "The action id from the confirm link"}}
	return []apiref.Route{
		{
			Method: "GET", Path: "/mcp/pending", Summary: "List actions waiting for confirmation",
			Description: "Open actions, newest first, with their file or folder as it is now. state is ready, changed " +
				"(the item was edited after the request, so it cannot be confirmed) or missing.",
			Auth: "session cookie", ResponseBody: PendingList{},
			Errors: []apiref.Error{unauthorized, delegated},
		},
		{
			Method: "POST", Path: "/mcp/pending/{id}/confirm", Summary: "Confirm an action",
			Description: "Runs the action once: trashes the item, or creates the public link and answers its url. An " +
				"item edited since the request is refused with 412 and the action closes.",
			Auth: "session cookie", PathParams: idParam, RequestBody: EmptyRequest{}, ResponseBody: PendingResult{},
			Errors: []apiref.Error{unauthorized, delegated, gone,
				{Status: 412, Code: "failed_precondition", Description: "The item changed after the request."}},
		},
		{
			Method: "POST", Path: "/mcp/pending/{id}/dismiss", Summary: "Dismiss an action",
			Description: "Closes the action without running it.",
			Auth:        "session cookie", PathParams: idParam, RequestBody: EmptyRequest{}, ResponseBody: PendingResult{},
			Errors: []apiref.Error{unauthorized, delegated, gone},
		},
	}
}
