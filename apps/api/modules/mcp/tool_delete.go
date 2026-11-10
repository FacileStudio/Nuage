package mcp

import (
	"context"

	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// waiting tells the model that nothing happened yet, and what to do.
const waiting = "Nothing happened yet: your client cannot ask the user to confirm. Give the user confirm_url and ask " +
	"them to confirm there. Do not call this tool again for the same item; the request stays open for 7 days."

// deleteItem is nuage_delete: the item goes to the trash once the user
// confirms.
func (t *toolset) deleteItem(ctx context.Context, req *sdk.CallToolRequest, in itemRef) (*sdk.CallToolResult, actionOutput, error) {
	if mcpkit.Retrying(req) {
		return t.resumeDelete(ctx, req)
	}
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, actionOutput{}, err
	}
	if err := checkType(in.Type); err != nil {
		return nil, actionOutput{}, err
	}
	file, folder, err := t.itemOf(ctx, userID, in.Type, in.ID, true)
	if err != nil {
		return nil, actionOutput{}, failure(err, hintItem+" Only the person who created a folder can delete it.")
	}
	result, outcome, err := t.confirm.Request(ctx, req, kindTrash, target(in.Type, in.ID), trashQuestion(file, folder))
	if err != nil {
		return nil, actionOutput{}, failure(err, hintItem)
	}
	if result != nil {
		return result, actionOutput{}, nil
	}
	return nil, actionOutput{Status: outcome.Status, Type: in.Type, ID: in.ID, ConfirmURL: outcome.ConfirmURL, Message: waiting}, nil
}

// resumeDelete handles the retry that carries the person's answer.
func (t *toolset) resumeDelete(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, actionOutput, error) {
	if _, err := t.userOf(ctx, req); err != nil {
		return nil, actionOutput{}, err
	}
	outcome, err := t.confirm.Resume(ctx, req, kindTrash)
	if err != nil {
		return nil, actionOutput{}, failure(err, "Call nuage_delete again to ask the user afresh.")
	}
	if outcome.Status == mcpkit.StatusDeclined {
		return nil, actionOutput{Status: mcpkit.StatusDeclined, Message: "The user declined. Nothing happened, and the item is kept."}, nil
	}
	r, _ := refOf(outcome.Action)
	return nil, actionOutput{Status: "trashed", Type: r.kind, ID: r.id, Message: "The user confirmed and the item is in the trash."}, nil
}
