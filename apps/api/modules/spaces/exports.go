package spaces

import "context"

// ListMine lists the spaces the person belongs to with their role, as GET
// /spaces does. The MCP server shows them next to the personal drive.
func (s *Service) ListMine(ctx context.Context, userID int64) ([]SpaceResponse, error) {
	return s.listSpaces(ctx, userID)
}
