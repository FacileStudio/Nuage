package files

import (
	"context"
	"io"
	"strconv"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// The methods below are the file operations the MCP server calls. Each is
// the same service call its REST route makes, so an agent gets exactly the
// reach and the refusals a person gets through the API.

// File reads one live file the person can reach, as GET /files/{id} does.
func (s *Service) File(ctx context.Context, userID, fileID int64) (*schemas.File, error) {
	return s.getFile(ctx, userID, strconv.FormatInt(fileID, 10))
}

// Open reads one file and opens its bytes, as GET /files/{id}/download does.
func (s *Service) Open(ctx context.Context, userID, fileID int64) (io.ReadSeekCloser, *schemas.File, error) {
	return s.downloadFile(ctx, userID, strconv.FormatInt(fileID, 10))
}

// Upload stores a new file, as POST /files does.
func (s *Service) Upload(ctx context.Context, userID int64, name, mimeType string, size int64, body io.Reader, folderID, spaceID *int64) (*schemas.File, bool, error) {
	return s.uploadFile(ctx, userID, name, mimeType, size, body, folderID, "", spaceID, "")
}

// Replace writes new content to a file the person uploaded and keeps the old
// content as a version, as POST /files/{id}/reupload does.
func (s *Service) Replace(ctx context.Context, userID, fileID int64, body io.Reader, size int64, mimeType string) (*schemas.File, error) {
	return s.reuploadFile(ctx, userID, strconv.FormatInt(fileID, 10), body, size, mimeType)
}

// ListFiles lists the files directly in a folder, or at the root when
// folderID is nil, of the personal drive or of a space, as GET /files does.
func (s *Service) ListFiles(ctx context.Context, userID int64, folderID, spaceID *int64) ([]schemas.File, error) {
	return s.listFiles(ctx, userID, folderID, "", "", "", spaceID)
}

// ListFolders lists the folders directly under parentID, as GET /folders
// does.
func (s *Service) ListFolders(ctx context.Context, userID int64, parentID, spaceID *int64) ([]schemas.Folder, error) {
	return s.listFolders(ctx, userID, parentID, spaceID)
}

// CreateFolder creates a folder, as POST /folders does.
func (s *Service) CreateFolder(ctx context.Context, userID int64, name string, parentID, spaceID *int64) (*schemas.Folder, error) {
	return s.createFolder(ctx, userID, name, parentID, spaceID)
}

// UpdateFile renames or moves a file, as PUT /files/{id} does: a nil field
// is unchanged and a folder id of 0 is the root.
func (s *Service) UpdateFile(ctx context.Context, userID, fileID int64, name *string, folderID *int64) (*schemas.File, error) {
	return s.updateFile(ctx, userID, strconv.FormatInt(fileID, 10), name, folderID)
}

// UpdateFolder renames or moves a folder the person created, as PUT
// /folders/{id} does.
func (s *Service) UpdateFolder(ctx context.Context, userID, folderID int64, name *string, parentID *int64) (*schemas.Folder, error) {
	return s.updateFolder(ctx, userID, strconv.FormatInt(folderID, 10), name, parentID)
}

// TrashFile moves a file to the trash, as DELETE /files/{id} does.
func (s *Service) TrashFile(ctx context.Context, userID, fileID int64) error {
	return s.deleteFile(ctx, userID, strconv.FormatInt(fileID, 10))
}

// TrashFolder moves a folder the person created, and everything in it, to
// the trash, as DELETE /folders/{id} does.
func (s *Service) TrashFolder(ctx context.Context, userID, folderID int64) error {
	return s.deleteFolder(ctx, userID, strconv.FormatInt(folderID, 10))
}

// OwnedFolder reads a live folder the person created, the only folders
// rename, move and delete accept.
func (s *Service) OwnedFolder(ctx context.Context, userID, folderID int64) (*schemas.Folder, error) {
	folder, _, _, err := s.getFolder(ctx, userID, strconv.FormatInt(folderID, 10))
	return folder, err
}
