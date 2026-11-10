package mcp

import (
	"context"
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/internal/authcontext"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/go-chi/chi/v5"
)

// webActor marks, in the audit, a confirmation made on the confirm page.
const webActor = "nuage-web"

// errDelegated refuses an agent confirming its own request.
var errDelegated = errors.Forbidden("only the person's own session can confirm actions")

// pendingUser is the signed-in person of a confirm page request. Only the
// person's own session passes: a delegated token or a restricted API token
// must never confirm what an agent asked for.
func pendingUser(r *http.Request) (mcpkit.Caller, error) {
	authenticated, ok := porte.From(r.Context())
	identity, found := authcontext.IdentityFromContext(r.Context())
	if !ok || !found {
		return mcpkit.Caller{}, errors.Unauthorized("missing auth")
	}
	if authenticated.SessionID == 0 || authenticated.Actor != "" {
		return mcpkit.Caller{}, errDelegated
	}
	return mcpkit.Caller{UserID: identity.UserID, Actor: webActor}, nil
}

// handleList answers the person's open actions with their items.
func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	who, err := pendingUser(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	list, err := s.tools.pendingList(r.Context(), who.UserID)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, list)
}

// handleConfirm runs one action, and answers the link a share made.
func (s *Service) handleConfirm(w http.ResponseWriter, r *http.Request) {
	who, err := pendingUser(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	outcome, err := s.tools.confirm.Confirm(r.Context(), who, chi.URLParam(r, "id"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	result := PendingResult{Status: "trashed"}
	if outcome.Action.Kind == kindShare {
		result.Status = "shared"
		if ref, err := refOf(outcome.Action); err == nil {
			if made, err := s.tools.linkFor(r.Context(), ref.userID, ref, outcome.Action.CreatedAt); err == nil {
				result.URL = made.URL
			}
		}
	}
	httpjson.WriteJSON(w, http.StatusOK, result)
}

// handleDismiss closes one action without running it.
func (s *Service) handleDismiss(w http.ResponseWriter, r *http.Request) {
	who, err := pendingUser(r)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	if err := s.tools.confirm.Dismiss(r.Context(), who, chi.URLParam(r, "id")); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, PendingResult{Status: mcpkit.OutcomeDismissed})
}

// pendingList renders the person's open actions with their items as they
// are now.
func (t *toolset) pendingList(ctx context.Context, userID string) (PendingList, error) {
	actions, err := t.confirm.Open(ctx, userID)
	if err != nil {
		return PendingList{}, err
	}
	list := PendingList{Actions: make([]PendingItem, 0, len(actions))}
	for _, action := range actions {
		item := PendingItem{
			ID: action.ID, Kind: action.Kind, ClientID: action.ClientID, Summary: action.Summary,
			CreatedAt: action.CreatedAt, ExpiresAt: action.ExpiresAt, State: t.confirm.State(ctx, action),
		}
		if r, err := refOf(action); err == nil {
			item.Days, item.Item = r.days, t.pendingFile(ctx, r, action.Kind == kindTrash)
		}
		list.Actions = append(list.Actions, item)
	}
	return list, nil
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
