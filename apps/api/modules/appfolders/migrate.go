package appfolders

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/tronc/errors"
)

// provisionedSpace is a space an app token created and charged to a user.
type provisionedSpace struct {
	ID            int64
	FacileID      string
	Name          string
	ProvisionedBy string
	ChargedTo     int64
}

// candidate is a file with what decides whether it may move.
type candidate struct {
	schemas.File
	Versions int64
	Shares   int64
}

// Migrate moves the files of every app-provisioned space (charged to a user,
// not linked to Registre) into that user's app folder for the registered app
// whose key equals the space's provisioned_by label. Without apply it changes
// nothing and reports what it would do.
//
// Only live files with no versions and no share links, published by the
// space's own app (origin_app equals provisioned_by), move; the others stay in
// the space and are reported, so nothing is lost, silently dropped or handed
// to an app that did not write it (Plume publishes signed contracts into
// Ardoise's space). A
// file keeps its linked_to as its key when that is a valid key, otherwise its
// folder path and name. A key already holding the same bytes deduplicates; a
// key holding other bytes is a conflict and the file stays. Moved files leave
// the space in the same transaction that records the object, so rerunning
// finds only what it skipped: the migration is idempotent. Spaces are never
// deleted here.
func (s *Service) Migrate(ctx context.Context, apply bool) (*MigrationReport, error) {
	var spaces []provisionedSpace
	if err := s.orm.WithContext(ctx).Raw(`SELECT id, facile_id, name, provisioned_by, charged_to FROM spaces
		WHERE provisioned_by <> '' AND charged_to IS NOT NULL AND registre_space_id IS NULL ORDER BY id`).
		Scan(&spaces).Error; err != nil {
		return nil, errors.Internal("failed to list provisioned spaces", err)
	}
	report := &MigrationReport{Applied: apply, Spaces: []SpaceReport{}}
	for _, space := range spaces {
		spaceReport, err := s.migrateSpace(ctx, space, apply)
		if err != nil {
			return report, err
		}
		report.add(spaceReport)
	}
	return report, nil
}

// spaceRun is the migration of one space into one app's folder for the user
// the space is charged to. planned holds the keys already claimed in this
// run, with the hash claiming them.
type spaceRun struct {
	service *Service
	space   provisionedSpace
	app     schemas.RegisteredApp
	paths   map[int64]string
	planned map[string]string
}

// migrateSpace plans, and with apply performs, the move of one space's files.
func (s *Service) migrateSpace(ctx context.Context, space provisionedSpace, apply bool) (SpaceReport, error) {
	report := SpaceReport{SpaceID: space.ID, FacileID: space.FacileID, Name: space.Name,
		App: space.ProvisionedBy, ChargedTo: space.ChargedTo, Files: []FileReport{}}
	var apps []schemas.RegisteredApp
	if err := s.orm.WithContext(ctx).Where("key = ?", space.ProvisionedBy).Limit(1).Find(&apps).Error; err != nil {
		return report, errors.Internal("failed to read the registered apps", err)
	}
	if len(apps) == 0 {
		report.Skipped = fmt.Sprintf("no registered app has the key %q", space.ProvisionedBy)
		return report, nil
	}
	files, paths, err := s.spaceContents(ctx, space.ID)
	if err != nil {
		return report, err
	}
	run := &spaceRun{service: s, space: space, app: apps[0], paths: paths, planned: map[string]string{}}
	for _, file := range files {
		report.Files = append(report.Files, run.migrate(ctx, file, apply))
	}
	return report, nil
}

// migrate plans one file and, with apply, moves it, recording any failure.
func (r *spaceRun) migrate(ctx context.Context, file candidate, apply bool) FileReport {
	entry := r.plan(ctx, file)
	if !apply || entry.Action == ActionSkip {
		return entry
	}
	if err := r.move(ctx, file.File, entry); err != nil {
		entry.Error = err.Error()
	}
	return entry
}

// spaceContents loads the space's files with their version and share counts,
// and the path of every folder in it.
func (s *Service) spaceContents(ctx context.Context, spaceID int64) ([]candidate, map[int64]string, error) {
	var files []candidate
	if err := s.orm.WithContext(ctx).Raw(`SELECT f.*,
		(SELECT COUNT(*) FROM file_versions v WHERE v.file_id = f.id) AS versions,
		(SELECT COUNT(*) FROM shares sh WHERE sh.file_id = f.id) AS shares
		FROM files f WHERE f.space_id = ? ORDER BY f.id`, spaceID).Scan(&files).Error; err != nil {
		return nil, nil, errors.Internal("failed to list the space's files", err)
	}
	var folders []schemas.Folder
	if err := s.orm.WithContext(ctx).Unscoped().Where("space_id = ?", spaceID).Find(&folders).Error; err != nil {
		return nil, nil, errors.Internal("failed to list the space's folders", err)
	}
	return files, folderPaths(folders), nil
}

// folderPaths maps each folder id to its slash-joined path from the space
// root. A cycle or a missing parent ends the walk, never loops.
func folderPaths(folders []schemas.Folder) map[int64]string {
	byID := make(map[int64]schemas.Folder, len(folders))
	for _, folder := range folders {
		byID[folder.ID] = folder
	}
	paths := make(map[int64]string, len(folders))
	for _, folder := range folders {
		var segments []string
		seen := map[int64]bool{}
		for current, ok := folder, true; ok && !seen[current.ID]; {
			seen[current.ID] = true
			segments = append([]string{current.Name}, segments...)
			if current.ParentID == nil {
				break
			}
			current, ok = byID[*current.ParentID]
		}
		paths[folder.ID] = strings.Join(segments, "/")
	}
	return paths
}

// plan decides what happens to one file.
func (r *spaceRun) plan(ctx context.Context, file candidate) FileReport {
	entry := FileReport{FileID: file.ID, Name: file.Name, Size: file.Size, Action: ActionSkip}
	if reason := blocked(file); reason != "" {
		entry.Reason = reason
		return entry
	}
	if file.OriginApp != r.space.ProvisionedBy {
		entry.Reason = fmt.Sprintf("published by %q, not by the space's app %q", file.OriginApp, r.space.ProvisionedBy)
		return entry
	}
	entry.Key = keyFor(file.File, r.paths)
	if validKey(entry.Key) != nil {
		entry.Reason = "no valid key can be derived from its name"
		return entry
	}
	hash, claimed := r.planned[entry.Key]
	if !claimed {
		if existing, err := r.service.Find(ctx, r.space.ChargedTo, r.app.ID, entry.Key); err == nil {
			hash, claimed = existing.Hash, true
		}
	}
	switch {
	case !claimed:
		entry.Action = ActionMove
		r.planned[entry.Key] = file.Hash
	case file.Hash != "" && hash == file.Hash:
		entry.Action = ActionDeduplicate
	default:
		entry.Reason = "the key already holds different content"
	}
	return entry
}

// blocked is why a file must stay in its space, or empty when it may move.
func blocked(file candidate) string {
	switch {
	case file.DeletedAt != nil:
		return "in the trash"
	case file.Versions > 0:
		return "has versions"
	case file.Shares > 0:
		return "has share links"
	}
	return ""
}

// keyFor is the key a file takes in the app folder: its linked_to when that
// is a valid key, else its folder path and name.
func keyFor(file schemas.File, paths map[int64]string) string {
	if file.LinkedTo != "" && validKey(file.LinkedTo) == nil {
		return file.LinkedTo
	}
	if file.FolderID != nil && paths[*file.FolderID] != "" {
		return paths[*file.FolderID] + "/" + file.Name
	}
	return file.Name
}
