package tests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/forfait"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// uploadedID uploads content and returns the new file's id.
func uploadedID(t *testing.T, e planEnv, name, content string) int64 {
	t.Helper()
	resp := uploadFile(e.ts, e.token, name, content, nil)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var file struct {
		ID int64 `json:"id"`
	}
	parseJSON(resp, &file)
	return file.ID
}

func TestPlanPersonalCeilingAndRefusalShape(t *testing.T) {
	e := newPlanEnv(t, "plan-personal@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-10)

	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "under.txt", "12345", nil).StatusCode)
	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "at.txt", "abcde", nil).StatusCode)

	resp := uploadFile(e.ts, e.token, "over.txt", "x", nil)
	require.Equal(t, http.StatusPaymentRequired, resp.StatusCode)
	var body planError
	parseJSON(resp, &body)
	assert.Equal(t, "payment_required", body.Error.Code)
	assert.Equal(t, "nuage.storage_gb", body.Error.Key)
	assert.Equal(t, int64(1), body.Error.Limit)
	assert.Equal(t, "https://sso.test/spaces", body.Error.UpgradeURL)
	assert.NotEmpty(t, body.Error.Message)

	var usage struct {
		StorageUsed  int64  `json:"storage_used"`
		StorageLimit int64  `json:"storage_limit"`
		Source       string `json:"source"`
	}
	parseJSON(doGet(e.ts, "/quota/me", e.token), &usage)
	assert.Equal(t, "plan", usage.Source)
	assert.Equal(t, gibibyte, usage.StorageLimit)
	assert.Equal(t, gibibyte, usage.StorageUsed)
}

func TestPlanSpaceCeilingIsTheSpacesOwn(t *testing.T) {
	e := newPlanEnv(t, "plan-space@example.com")
	spaceID := e.linkedSpace(t)
	e.enforce(t, 1, "100", "1")
	e.fill(t, &spaceID, gibibyte-4)

	assert.Equal(t, http.StatusCreated, uploadFileToSpace(e.ts, e.token, "at.txt", "abcd", spaceID).StatusCode)
	resp := uploadFileToSpace(e.ts, e.token, "over.txt", "e", spaceID)
	require.Equal(t, http.StatusPaymentRequired, resp.StatusCode)
	var body planError
	parseJSON(resp, &body)
	assert.Equal(t, "https://sso.test/spaces/spc_1", body.Error.UpgradeURL)

	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "mine.txt", "personal", nil).StatusCode)
}

func TestPlanUnlinkedSpaceRefusedOnlyWhenEnforcing(t *testing.T) {
	e := newPlanEnv(t, "plan-unlinked@example.com")
	spaceID := createSpace(t, e.ts, e.token, "Local")

	assert.Equal(t, http.StatusCreated, uploadFileToSpace(e.ts, e.token, "before.txt", "ok", spaceID).StatusCode)
	e.enforce(t, 1, "100", "")
	assert.Equal(t, http.StatusPreconditionFailed, uploadFileToSpace(e.ts, e.token, "after.txt", "no", spaceID).StatusCode)
}

func TestPlanFailsOpenBeforeTheFirstSnapshot(t *testing.T) {
	e := newPlanEnv(t, "plan-open@example.com")
	e.fill(t, nil, 6*gibibyte)

	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "first.txt", "fine", nil).StatusCode)
	e.enforce(t, 0, "0", "")
	assert.False(t, e.ts.plans.Enforcing())
	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "second.txt", "still fine", nil).StatusCode)

	var usage struct {
		Source string `json:"source"`
	}
	parseJSON(doGet(e.ts, "/quota/me", e.token), &usage)
	assert.Equal(t, "instance", usage.Source)
}

func TestPlanStateSurvivesARestart(t *testing.T) {
	e := newPlanEnv(t, "plan-restart@example.com")
	e.enforce(t, 5, "1", "")

	restarted := entitlements.New(e.ts.db, testIssuer, nil)
	require.False(t, restarted.Enforcing())
	require.NoError(t, restarted.Restore(context.Background()))
	assert.True(t, restarted.Enforcing())
	assert.Equal(t, int64(1), restarted.Limit(forfait.Scope{Subject: "sub-1"}))

	stale := directory.Snapshot{Version: 4, EntitlementsRevision: 4,
		People: []directory.Person{{Subject: "sub-1", Entitlements: storageGB("100")}}}
	require.NoError(t, restarted.Apply(context.Background(), stale))
	assert.Equal(t, int64(1), restarted.Limit(forfait.Scope{Subject: "sub-1"}))
}

func TestPlanAdminOverrideWins(t *testing.T) {
	e := newPlanEnv(t, "plan-override@example.com")
	e.enforce(t, 1, "0", "")
	require.Equal(t, http.StatusPaymentRequired, uploadFile(e.ts, e.token, "a.txt", "plan says no", nil).StatusCode)

	setLimit := func(limit int64) {
		resp := doJSON(e.ts, "PUT", fmt.Sprintf("/quota/users/%d", e.userID), map[string]int64{"storage_limit": limit}, e.token)
		require.Equal(t, http.StatusOK, resp.StatusCode)
	}
	setLimit(-1)
	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "b.txt", "override says yes", nil).StatusCode)

	e.enforce(t, 2, "100", "")
	setLimit(1)
	assert.Equal(t, http.StatusRequestEntityTooLarge, uploadFile(e.ts, e.token, "c.txt", "override says no", nil).StatusCode)
}

func TestPlanTrashKeepsCountingUntilPurged(t *testing.T) {
	e := newPlanEnv(t, "plan-trash@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-10)
	fileID := uploadedID(t, e, "keep.txt", "0123456789")
	require.Equal(t, http.StatusOK, doDelete(e.ts, fmt.Sprintf("/files/%d", fileID), e.token).StatusCode)

	assert.Equal(t, http.StatusPaymentRequired, uploadFile(e.ts, e.token, "again.txt", "x", nil).StatusCode,
		"trashing must not free room for new bytes")
	assert.Equal(t, http.StatusOK, doGet(e.ts, "/files", e.token).StatusCode)
	assert.Equal(t, http.StatusOK, doGet(e.ts, "/trash", e.token).StatusCode)
	assert.Equal(t, http.StatusOK,
		doJSON(e.ts, "POST", fmt.Sprintf("/trash/file/%d/restore", fileID), nil, e.token).StatusCode,
		"a restore adds nothing, so it passes at the ceiling")

	require.Equal(t, http.StatusOK, doDelete(e.ts, fmt.Sprintf("/files/%d", fileID), e.token).StatusCode)
	assert.Equal(t, http.StatusOK, doDelete(e.ts, fmt.Sprintf("/trash/file/%d", fileID), e.token).StatusCode)
	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "after-purge.txt", "x", nil).StatusCode)
}

func TestPlanUnreadableSavedStateEnforcesDefaults(t *testing.T) {
	for name, document := range map[string]string{
		"corrupt":    "not json",
		"revision 0": `{"version":1,"snapshot_version":3,"entitlements_revision":0,"spaces":{},"people":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			e := newPlanEnv(t, "plan-corrupt@example.com")
			require.NoError(t, e.ts.db.Exec("INSERT INTO entitlement_state (id, document, updated_at) VALUES (1, ?, now())", document).Error)

			restarted := entitlements.New(e.ts.db, testIssuer, nil)
			require.NoError(t, restarted.Restore(context.Background()))
			assert.True(t, restarted.Enforcing())
			assert.Equal(t, int64(5), restarted.Limit(forfait.Scope{Subject: "sub-1"}))

			real := directory.Snapshot{Version: 2, EntitlementsRevision: 1,
				People: []directory.Person{{Subject: "sub-1", Entitlements: storageGB("100")}}}
			require.NoError(t, restarted.Apply(context.Background(), real))
			assert.Equal(t, int64(100), restarted.Limit(forfait.Scope{Subject: "sub-1"}))
		})
	}
}

func TestPlanRestoreErrorIsReturnedNotFailedOpen(t *testing.T) {
	e := newPlanEnv(t, "plan-dberror@example.com")
	require.NoError(t, e.ts.db.Exec("DROP TABLE entitlement_state").Error)
	restarted := entitlements.New(e.ts.db, testIssuer, nil)
	assert.Error(t, restarted.Restore(context.Background()))
}

func TestPlanEmptySubjectReadsDefaults(t *testing.T) {
	ts := setupSettledServer(t)
	rawID, token := registerUser(ts, "plan-nosub@example.com", "password12345")
	e := planEnv{ts: ts, token: token}
	_, err := fmt.Sscan(rawID, &e.userID)
	require.NoError(t, err)
	require.NoError(t, ts.plans.Apply(context.Background(), directory.Snapshot{Version: 1, EntitlementsRevision: 1,
		People: []directory.Person{{Subject: "", Entitlements: storageGB("100")}}}))
	e.fill(t, nil, 5*gibibyte-5)

	assert.Equal(t, http.StatusPaymentRequired, uploadFile(ts, token, "x.txt", "0123456789", nil).StatusCode)
}

func TestPlanDownloadAndDeleteWorkOverTheLimit(t *testing.T) {
	e := newPlanEnv(t, "plan-read@example.com")
	fileID := uploadedID(t, e, "mine.txt", "my bytes")
	e.enforce(t, 1, "0", "")

	resp := doGet(e.ts, fmt.Sprintf("/files/%d/download", fileID), e.token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "my bytes", string(body))
	assert.Equal(t, http.StatusOK, doDelete(e.ts, fmt.Sprintf("/files/%d", fileID), e.token).StatusCode)
}
