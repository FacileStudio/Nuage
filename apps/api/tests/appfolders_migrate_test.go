package tests

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/modules/appfolders"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertObject checks that key holds want in the app's folder.
func assertObject(t *testing.T, ts *testServer, token, key, want string) {
	t.Helper()
	status, body := getObject(ts, token, key)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, want, body)
}

func fileID(t *testing.T, resp *http.Response) int64 {
	t.Helper()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)
	return file.ID
}

func spaceFileCount(t *testing.T, ts *testServer, spaceID int64) int64 {
	t.Helper()
	var count int64
	require.NoError(t, ts.db.Unscoped().Model(&schemas.File{}).Where("space_id = ?", spaceID).Count(&count).Error)
	return count
}

// assertDryRun checks that a dry run plans two moves and one skip, reports
// the unregistered app's space and changes nothing.
func assertDryRun(t *testing.T, ts *testServer, spaceID int64) {
	t.Helper()
	dry, err := ts.appFolders.Migrate(context.Background(), false)
	require.NoError(t, err)
	assert.False(t, dry.Applied)
	assert.Equal(t, 2, dry.Moved)
	assert.Equal(t, 1, dry.Skipped)
	require.Len(t, dry.Spaces, 2)
	assert.Contains(t, dry.Spaces[1].Skipped, "no registered app")
	assert.Equal(t, int64(3), spaceFileCount(t, ts, spaceID), "a dry run changes nothing")

}

func TestMigrationMovesProvisionedSpacesIdempotently(t *testing.T) {
	ts, verifier := setupAppServer(t)
	userID, session := registered(t, ts, "migrate-app@example.com")
	labelled := mintProvisionToken(t, ts, session, "plume")
	_, spaceID := provisionSpace(t, ts, labelled)
	other := mintProvisionToken(t, ts, session, "unknown-app")
	_, otherSpace := provisionSpace(t, ts, other)

	fileID(t, publish(ts, labelled, spaceID, "contract.pdf", "signed contract", "contract-42"))
	fileID(t, uploadFileToSpace(ts, labelled, "notes.txt", "some notes", spaceID))
	trashed := fileID(t, uploadFileToSpace(ts, labelled, "old.txt", "old", spaceID))
	require.Equal(t, http.StatusOK, doDelete(ts, fmt.Sprintf("/files/%d", trashed), labelled).StatusCode)
	fileID(t, uploadFileToSpace(ts, other, "elsewhere.txt", "x", otherSpace))

	var before struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(doGet(ts, "/quota/me", session), &before)

	assertDryRun(t, ts, spaceID)

	applied, err := ts.appFolders.Migrate(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, 2, applied.Moved)
	assert.Equal(t, 0, applied.Failed)
	assert.Equal(t, int64(1), spaceFileCount(t, ts, spaceID), "only the trashed file stays")

	plume := verifier.mint(userID, "plume-client")
	assertObject(t, ts, plume, "contract-42", "signed contract")
	assertObject(t, ts, plume, "notes.txt", "some notes")

	again, err := ts.appFolders.Migrate(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, 0, again.Moved)
	assert.Equal(t, 1, again.Skipped)
	assert.Len(t, listObjects(t, ts, plume, "").Objects, 2)

	var after struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(doGet(ts, "/quota/me", session), &after)
	assert.Equal(t, before.StorageUsed, after.StorageUsed, "moving keeps the person's usage")
}

func TestMigrationDeduplicatesAndKeepsConflicts(t *testing.T) {
	ts, verifier := setupAppServer(t)
	userID, session := registered(t, ts, "migrate-dup@example.com")
	labelled := mintProvisionToken(t, ts, session, "plume")
	_, spaceID := provisionSpace(t, ts, labelled)
	plume := verifier.mint(userID, "plume-client")

	fileID(t, publish(ts, labelled, spaceID, "same.pdf", "same bytes", "same"))
	fileID(t, publish(ts, labelled, spaceID, "clash.pdf", "space bytes", "clash"))
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "same", "same bytes").StatusCode)
	require.Equal(t, http.StatusCreated, putObject(ts, plume, "clash", "app bytes").StatusCode)

	report, err := ts.appFolders.Migrate(context.Background(), true)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Duplicates)
	assert.Equal(t, 1, report.Skipped)
	actions := map[string]string{}
	for _, file := range report.Spaces[0].Files {
		actions[file.Key] = file.Action
	}
	assert.Equal(t, appfolders.ActionDeduplicate, actions["same"])
	assert.Equal(t, appfolders.ActionSkip, actions["clash"])
	assert.Equal(t, int64(1), spaceFileCount(t, ts, spaceID), "the conflicting file stays in its space")
	_, body := getObject(ts, plume, "clash")
	assert.Equal(t, "app bytes", body)
}
