package mcp

import mcpkit "github.com/FacileStudio/tronc/mcp"

// PendingItem is one action waiting on the confirm page.
type PendingItem struct {
	mcpkit.PendingItem
	Item *PendingFile `json:"item,omitempty"`
	Days int          `json:"expires_in_days,omitempty"`
}

// PendingFile is the file or folder an action targets, as it is now.
type PendingFile struct {
	Type     string `json:"type"`
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type,omitempty"`
	Size     int64  `json:"size,omitempty"`
}
