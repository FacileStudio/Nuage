package mcp

import "time"

// PendingItem is one action waiting on the confirm page.
type PendingItem struct {
	ID        string       `json:"id"`
	Kind      string       `json:"kind"`
	ClientID  string       `json:"client_id"`
	Summary   string       `json:"summary"`
	CreatedAt time.Time    `json:"created_at"`
	ExpiresAt time.Time    `json:"expires_at"`
	State     string       `json:"state"`
	Item      *PendingFile `json:"item,omitempty"`
	Days      int          `json:"expires_in_days,omitempty"`
}

// PendingFile is the file or folder an action targets, as it is now.
type PendingFile struct {
	Type     string `json:"type"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

// PendingList is the body of GET /api/mcp/pending.
type PendingList struct {
	Actions []PendingItem `json:"actions"`
}

// PendingResult is the body answered when an action is confirmed or
// dismissed. URL is the new public link when a share was confirmed.
type PendingResult struct {
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

// EmptyRequest documents a route that takes no body.
type EmptyRequest struct{}
