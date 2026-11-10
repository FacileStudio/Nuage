package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxSaveBytes caps what nuage_save_file stores in one call.
const maxSaveBytes = 5 << 20

// Refusals of the write tools, each saying what to change.
var (
	errName        = stderrors.New("name must be 1 to 255 characters, with no slash, backslash or control character, and not . or ..")
	errEncoding    = stderrors.New(`encoding must be "text" or "base64"`)
	errBase64      = stderrors.New("content is not valid base64. Send standard base64, or set encoding to text")
	errTooLarge    = fmt.Errorf("content is over %d MB. Split it into several files", maxSaveBytes>>20)
	errNotUploader = stderrors.New("only the person who uploaded this file can overwrite it. Save it as a new file instead, without file_id")
	errNoChange    = stderrors.New("give new_name, destination_folder_id or both")
)

// cleanName trims a name and refuses one Nuage or WebDAV cannot hold.
func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	bad := name == "" || name == "." || name == ".." || len(name) > 255 || strings.ContainsAny(name, `/\`) ||
		strings.ContainsFunc(name, unicode.IsControl)
	if bad {
		return "", errName
	}
	return name, nil
}

// decode reads the content argument and reports whether it is text.
func decode(content, encoding string) ([]byte, bool, error) {
	var data []byte
	switch encoding {
	case "", "text":
		data = []byte(content)
	case "base64":
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content))
		if err != nil {
			return nil, false, errBase64
		}
		data = raw
	default:
		return nil, false, errEncoding
	}
	if len(data) > maxSaveBytes {
		return nil, false, errTooLarge
	}
	return data, encoding != "base64" || looksText(data), nil
}

// saveFile is nuage_save_file.
func (t *toolset) saveFile(ctx context.Context, req *sdk.CallToolRequest, in saveInput) (*sdk.CallToolResult, writeOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, writeOutput{}, err
	}
	data, text, err := decode(in.Content, in.Encoding)
	if err != nil {
		return nil, writeOutput{}, err
	}
	if in.FileID != 0 {
		return t.replace(ctx, userID, in.FileID, data, text)
	}
	name, err := cleanName(in.Name)
	if err != nil {
		return nil, writeOutput{}, err
	}
	folderID, spaceID, err := t.placeOf(ctx, userID, in.FolderID, in.SpaceID)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	mimeType := savedType(name, text, mimetype.Resolve(name, ""))
	f, created, err := t.Files.Upload(ctx, userID, name, mimeType, int64(len(data)), bytes.NewReader(data), folderID, spaceID)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	out := writeOutput{Status: "created", Item: fileItem(*f, "", false)}
	switch {
	case !created:
		out.Status, out.Note = "unchanged", "An identical file with this name was already there, so nothing new was stored."
	case f.Name != name:
		out.Note = fmt.Sprintf("A file named %q was already there, so this one was saved as %q. To overwrite instead, call again with file_id.", name, f.Name)
	}
	return nil, out, nil
}

// replace writes data over a file the person uploaded.
func (t *toolset) replace(ctx context.Context, userID, fileID int64, data []byte, text bool) (*sdk.CallToolResult, writeOutput, error) {
	current, err := t.Files.File(ctx, userID, fileID)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	if current.UploadedBy != userID {
		return nil, writeOutput{}, errNotUploader
	}
	mimeType := savedType(current.Name, text, mimetype.Resolve(current.Name, current.MimeType))
	f, err := t.Files.Replace(ctx, userID, fileID, bytes.NewReader(data), int64(len(data)), mimeType)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	return nil, writeOutput{Status: "replaced", Item: fileItem(*f, "", false), Note: "The previous content is kept as a version in Nuage."}, nil
}

// createFolder is nuage_create_folder. A folder of that name already in
// place is returned as it is, so calling twice makes one folder.
func (t *toolset) createFolder(ctx context.Context, req *sdk.CallToolRequest, in folderInput) (*sdk.CallToolResult, writeOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, writeOutput{}, err
	}
	name, err := cleanName(in.Name)
	if err != nil {
		return nil, writeOutput{}, err
	}
	parentID, spaceID, err := t.placeOf(ctx, userID, in.ParentID, in.SpaceID)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	siblings, err := t.Files.ListFolders(ctx, userID, parentID, spaceID)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	for _, f := range siblings {
		if f.Name == name {
			return nil, writeOutput{Status: "exists", Item: folderItem(f, "", false)}, nil
		}
	}
	f, err := t.Files.CreateFolder(ctx, userID, name, parentID, spaceID)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	return nil, writeOutput{Status: "created", Item: folderItem(*f, "", false)}, nil
}

// move is nuage_move: rename, move, or both.
func (t *toolset) move(ctx context.Context, req *sdk.CallToolRequest, in moveInput) (*sdk.CallToolResult, writeOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, writeOutput{}, err
	}
	if err := checkType(in.Type); err != nil {
		return nil, writeOutput{}, err
	}
	var name *string
	if in.NewName != "" {
		clean, err := cleanName(in.NewName)
		if err != nil {
			return nil, writeOutput{}, err
		}
		name = &clean
	}
	if name == nil && in.DestinationID == nil {
		return nil, writeOutput{}, errNoChange
	}
	moved, err := t.relocate(ctx, userID, in, name)
	if err != nil {
		return nil, writeOutput{}, failure(err, hintItem)
	}
	return nil, writeOutput{Status: "moved", Item: moved}, nil
}

// relocate runs the rename or move through the matching REST service call.
func (t *toolset) relocate(ctx context.Context, userID int64, in moveInput, name *string) (item, error) {
	if in.Type == typeFile {
		f, err := t.Files.UpdateFile(ctx, userID, in.ID, name, in.DestinationID)
		if err != nil {
			return item{}, err
		}
		return t.withPath(ctx, fileItem(*f, "", false), f.FolderID)
	}
	f, err := t.Files.UpdateFolder(ctx, userID, in.ID, name, in.DestinationID)
	if err != nil {
		return item{}, err
	}
	return t.withPath(ctx, folderItem(*f, "", false), f.ParentID)
}
