package mcp

import (
	"cmp"
	"context"
	stderrors "errors"
	"slices"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Limits of the listing and search tools.
const (
	defaultListLimit   = 100
	maxListLimit       = 200
	defaultSearchLimit = 50
	maxSearchLimit     = 100
	searchMore         = "More items match. Make query more precise, set type, or set space_id."
)

// errQuery is the refusal for an empty search.
var errQuery = stderrors.New("query is required: give part of a name, such as invoice or .pdf")

// position is the cursor of a paged tool: how many items earlier pages
// returned.
type position struct {
	Offset int `json:"o"`
}

// listFolder is nuage_list_folder.
func (t *toolset) listFolder(ctx context.Context, req *sdk.CallToolRequest, in listInput) (*sdk.CallToolResult, listOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, listOutput{}, err
	}
	full, err := mcpkit.Detailed(in.ResponseFormat)
	if err != nil {
		return nil, listOutput{}, err
	}
	var at position
	if err := mcpkit.DecodeCursor(in.Cursor, &at); err != nil || at.Offset < 0 {
		return nil, listOutput{}, mcpkit.ErrCursor
	}
	out, items, err := t.open(ctx, userID, in, full)
	if err != nil {
		return nil, listOutput{}, failure(err, hintItem)
	}
	if at.Offset > len(items) {
		return nil, listOutput{}, mcpkit.ErrCursor
	}
	if out.FolderID == nil && out.SpaceID == nil && at.Offset == 0 {
		list, err := t.spaces(ctx, userID)
		if err != nil {
			return nil, listOutput{}, failure(err, "")
		}
		out.Spaces = &list
	}
	return nil, pageOf(out, items, at.Offset, clamp(in.Limit, defaultListLimit, maxListLimit)), nil
}

// open lists one folder, or a root, folders first then files, each by name.
func (t *toolset) open(ctx context.Context, userID int64, in listInput, full bool) (listOutput, []item, error) {
	out := listOutput{Location: "/", SpaceID: optional(in.SpaceID)}
	var parent *int64
	if in.FolderID != 0 {
		folder, err := t.Files.Folder(ctx, userID, in.FolderID)
		if err != nil {
			return out, nil, err
		}
		paths, err := t.Files.FolderPaths(ctx, []int64{folder.ID})
		if err != nil {
			return out, nil, err
		}
		parent, out.FolderID, out.SpaceID, out.Location = &folder.ID, &folder.ID, folder.SpaceID, paths[folder.ID]
	}
	folders, err := t.Files.ListFolders(ctx, userID, parent, out.SpaceID)
	if err != nil {
		return out, nil, err
	}
	fileList, err := t.Files.ListFiles(ctx, userID, parent, out.SpaceID)
	if err != nil {
		return out, nil, err
	}
	slices.SortStableFunc(fileList, func(a, b schemas.File) int { return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	items := make([]item, 0, len(folders)+len(fileList))
	for _, f := range folders {
		items = append(items, folderItem(f, childPath(out.Location, f.Name), full))
	}
	for _, f := range fileList {
		items = append(items, fileItem(f, childPath(out.Location, f.Name), full))
	}
	return out, items, nil
}

// spaces lists the person's spaces for the personal root listing.
func (t *toolset) spaces(ctx context.Context, userID int64) ([]spaceSummary, error) {
	list, err := t.Spaces.ListMine(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]spaceSummary, 0, len(list))
	for _, s := range list {
		out = append(out, spaceSummary{SpaceID: s.ID, Name: s.Name, Role: s.Role})
	}
	return out, nil
}

// pageOf cuts items from offset to the limit and the size budget, with the
// cursor of the next page.
func pageOf(out listOutput, items []item, offset, limit int) listOutput {
	rest := items[offset:min(offset+limit, len(items))]
	keep := mcpkit.FitCount(len(rest), func(n int) int {
		probe := out
		probe.Items, probe.Total, probe.NextCursor = rest[:n], len(items), "next-cursor-placeholder"
		return mcpkit.EncodedSize(probe)
	})
	out.Items, out.Total = rest[:keep], len(items)
	if offset+keep < len(items) {
		out.NextCursor = mcpkit.EncodeCursor(position{Offset: offset + keep})
	}
	return out
}

// clamp bounds a page size, with fallback for zero or less.
func clamp(n, fallback, ceiling int) int {
	if n <= 0 {
		return fallback
	}
	return min(n, ceiling)
}

// search is nuage_search.
func (t *toolset) search(ctx context.Context, req *sdk.CallToolRequest, in searchInput) (*sdk.CallToolResult, searchOutput, error) {
	userID, err := t.userOf(ctx, req)
	if err != nil {
		return nil, searchOutput{}, err
	}
	full, err := mcpkit.Detailed(in.ResponseFormat)
	if err != nil {
		return nil, searchOutput{}, err
	}
	if strings.TrimSpace(in.Query) == "" {
		return nil, searchOutput{}, errQuery
	}
	if in.Type != "" {
		if err := checkType(in.Type); err != nil {
			return nil, searchOutput{}, err
		}
	}
	limit := clamp(in.Limit, defaultSearchLimit, maxSearchLimit)
	hits, err := t.Files.SearchNames(ctx, userID, in.Query, optional(in.SpaceID), limit+1)
	if err != nil {
		return nil, searchOutput{}, failure(err, "Call nuage_list_folder for the spaces you belong to.")
	}
	items, err := t.hitItems(ctx, hits.Files, hits.Folders, in.Type, full)
	if err != nil {
		return nil, searchOutput{}, failure(err, "")
	}
	out := searchOutput{Items: items[:min(limit, len(items))], More: len(items) > limit}
	keep := mcpkit.FitCount(len(out.Items), func(n int) int {
		return mcpkit.EncodedSize(searchOutput{Items: out.Items[:n], More: true, Note: searchMore})
	})
	out.More = out.More || keep < len(out.Items)
	out.Items = out.Items[:keep]
	if out.More {
		out.Note = searchMore
	}
	return nil, out, nil
}

// hitItems renders search hits of kind, or both kinds, newest first, with
// their paths.
func (t *toolset) hitItems(ctx context.Context, fileHits []schemas.File, folderHits []schemas.Folder, kind string, full bool) ([]item, error) {
	var ids []int64
	for _, f := range fileHits {
		if f.FolderID != nil {
			ids = append(ids, *f.FolderID)
		}
	}
	for _, f := range folderHits {
		ids = append(ids, f.ID)
	}
	paths, err := t.Files.FolderPaths(ctx, ids)
	if err != nil {
		return nil, err
	}
	var items []item
	if kind != typeFolder {
		for _, f := range fileHits {
			items = append(items, fileItem(f, childPath(parentPath(paths, f.FolderID), f.Name), full))
		}
	}
	if kind != typeFile {
		for _, f := range folderHits {
			items = append(items, folderItem(f, paths[f.ID], full))
		}
	}
	slices.SortStableFunc(items, func(a, b item) int { return cmp.Compare(b.UpdatedAt, a.UpdatedAt) })
	return items, nil
}
