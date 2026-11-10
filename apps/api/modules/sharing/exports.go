package sharing

import (
	"context"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// Create makes a public link to a file or folder the person can reach, as
// POST /shares does. The MCP server calls it once the person confirmed.
func (s *Service) Create(ctx context.Context, userID int64, req CreateShareRequest) (*schemas.Share, error) {
	return s.createShare(ctx, userID, req)
}

// Revoke deletes one of the person's links, as DELETE /shares/{id} does.
func (s *Service) Revoke(ctx context.Context, userID, shareID int64) error {
	return s.deleteShare(ctx, userID, strconv.FormatInt(shareID, 10))
}

// ListMine lists every link the person made, newest first, with its file
// or folder, as GET /shares/by-me does without a space filter.
func (s *Service) ListMine(ctx context.Context, userID int64) ([]schemas.Share, error) {
	return s.listSharedByMe(ctx, userID, false, nil)
}
