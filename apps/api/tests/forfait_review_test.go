package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staleIdentity gives the user a second, more recently synced identity under
// the issuer, the shape an Authentik-era subject left behind has.
func staleIdentity(t *testing.T, e planEnv) {
	t.Helper()
	require.NoError(t, e.ts.db.Exec(
		"INSERT INTO porte_identities (user_id, provider, subject, synced_at) VALUES (?, ?, 'stale-sub', now())",
		e.userID, testIssuer).Error)
}

func TestPlanReadsTheSubjectRegistreKnows(t *testing.T) {
	e := newPlanEnv(t, "plan-twoids@example.com")
	staleIdentity(t, e)
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-5)

	assert.Equal(t, http.StatusPaymentRequired, uploadFile(e.ts, e.token, "a.txt", "0123456789", nil).StatusCode,
		"the 1 GB plan of sub-1 applies, not the stale subject's 5 GB default")

	var logs bytes.Buffer
	quota.NewService(e.ts.db).WithPlans(e.ts.plans).ReportIdentities(context.Background(), slog.New(slog.NewTextHandler(&logs, nil)))
	assert.Contains(t, logs.String(), "several identities")
	assert.Contains(t, logs.String(), "plan_subject=sub-1")
	var count int64
	require.NoError(t, e.ts.db.Table("porte_identities").Where("user_id = ? AND provider = ?", e.userID, testIssuer).Count(&count).Error)
	assert.EqualValues(t, 2, count, "no identity is deleted")
}

func TestPlanFallsBackToTheMostRecentIdentity(t *testing.T) {
	e := newPlanEnv(t, "plan-recent@example.com")
	staleIdentity(t, e)
	require.NoError(t, e.ts.plans.Apply(context.Background(), directory.Snapshot{Version: 1, EntitlementsRevision: 1,
		People: []directory.Person{{Subject: "unrelated", Entitlements: storageGB("1")}}}))
	e.fill(t, nil, gibibyte)

	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "a.txt", "0123456789", nil).StatusCode,
		"neither subject is known, so the most recently synced one reads the 5 GB default")
}

func TestReportOverPlanListsPeopleOverTheirCeiling(t *testing.T) {
	e := newPlanEnv(t, "plan-report@example.com")
	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte+10)

	var logs bytes.Buffer
	quota.NewService(e.ts.db).WithPlans(e.ts.plans).ReportOverPlan(context.Background(), slog.New(slog.NewTextHandler(&logs, nil)))
	assert.Contains(t, logs.String(), "person over their plan")
	assert.Contains(t, logs.String(), fmt.Sprintf("user_id=%d", e.userID))
}

func TestSessionCannotFindOrCreateSpacesRegistreOwns(t *testing.T) {
	ts := setupTestServerWith(t, delegatedEnv(), delegatedVerifier{userID: new(int64)})
	_, token := registered(t, ts, "plan-session-space@example.test")

	assert.Equal(t, http.StatusConflict, ensureSpace(ts, token, "fac_0123456789abcdef01234567").StatusCode)
	app := mintProvisionToken(t, ts, token, "ardoise")
	assert.Equal(t, http.StatusCreated, ensureSpace(ts, app, "fac_0123456789abcdef01234567").StatusCode)
}

func TestQuotaMeterUnderAnOverrideShowsChargedUsage(t *testing.T) {
	e := newPlanEnv(t, "plan-meter@example.com")
	app := mintProvisionToken(t, e.ts, e.token, "ardoise")
	spaceID := provisionedID(t, e.ts, app)
	require.Equal(t, http.StatusOK,
		doJSON(e.ts, "PUT", fmt.Sprintf("/quota/users/%d", e.userID), map[string]int64{"storage_limit": 2000}, e.token).StatusCode)
	e.enforce(t, 1, "1", "")
	require.Equal(t, http.StatusCreated, uploadFileToSpace(e.ts, app, "a.pdf", strings.Repeat("a", 700), spaceID).StatusCode)

	var usage struct {
		StorageUsed  int64  `json:"storage_used"`
		StorageLimit int64  `json:"storage_limit"`
		Source       string `json:"source"`
	}
	parseJSON(doGet(e.ts, "/quota/me", e.token), &usage)
	assert.Equal(t, "override", usage.Source)
	assert.Equal(t, int64(2000), usage.StorageLimit)
	assert.Equal(t, int64(700), usage.StorageUsed)
}

func TestParallelChunksCannotStagePastTheDeclaredSize(t *testing.T) {
	e := newPlanEnv(t, "plan-chunks@example.com")
	sessionID := initSession(t, e.ts, e.token, 100)

	var wg sync.WaitGroup
	for part := 1; part <= 10; part++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			putChunk(e.ts, e.token, sessionID, part, make([]byte, 30))
		}()
	}
	wg.Wait()

	var staged int64
	require.NoError(t, e.ts.db.Model(&schemas.UploadChunk{}).Where("session_id = ?", sessionID).
		Select("COALESCE(SUM(size), 0)").Scan(&staged).Error)
	assert.LessOrEqual(t, staged, int64(100))
}

func TestPlanIgnoredSnapshotKeepsTheKnownSubject(t *testing.T) {
	e := newPlanEnv(t, "pr6-restart@example.com")
	require.NoError(t, e.ts.db.Exec(
		"INSERT INTO porte_identities (user_id, provider, subject, synced_at) VALUES (?, ?, 'sub-stale', now())", e.userID, testIssuer).Error)
	require.NoError(t, e.ts.plans.Apply(context.Background(), directory.Snapshot{Version: 5, EntitlementsRevision: 1,
		People: []directory.Person{{Subject: "sub-1", Entitlements: storageGB("100")}}}))

	restarted := entitlements.New(e.ts.db, testIssuer, nil)
	require.NoError(t, restarted.Restore(context.Background()))
	limitOf := func() int64 {
		t.Helper()
		usage, err := quota.NewService(e.ts.db).WithPlans(restarted).GetUsage(context.Background(), e.userID)
		require.NoError(t, err)
		return usage.StorageLimit
	}
	require.Equal(t, int64(100)<<30, limitOf(), "after a restart the restored people decide the subject")

	stale := directory.Snapshot{Version: 4, EntitlementsRevision: 1,
		People: []directory.Person{{Subject: "sub-stale", Entitlements: storageGB("1")}}}
	require.NoError(t, restarted.Apply(context.Background(), stale))
	assert.Equal(t, int64(100)<<30, limitOf(), "an older snapshot the store ignored must not change the subject")
}

func TestLateChunkCannotLandDuringAssembly(t *testing.T) {
	e := newPlanEnv(t, "plan-late-chunk@example.com")
	sessionID := initSession(t, e.ts, e.token, 100)
	require.Equal(t, http.StatusCreated, putChunk(e.ts, e.token, sessionID, 1, make([]byte, 50)).StatusCode)

	assembling := e.ts.db.Begin()
	require.NoError(t, assembling.Error)
	defer assembling.Rollback()
	require.NoError(t, assembling.Exec("UPDATE upload_sessions SET status = 'assembling' WHERE id = ?", sessionID).Error)

	late := make(chan int, 1)
	go func() { late <- putChunk(e.ts, e.token, sessionID, 2, make([]byte, 50)).StatusCode }()
	time.Sleep(300 * time.Millisecond)
	require.NoError(t, assembling.Commit().Error)

	assert.Equal(t, http.StatusNotFound, <-late)
	var parts int64
	require.NoError(t, e.ts.db.Model(&schemas.UploadChunk{}).Where("session_id = ?", sessionID).Count(&parts).Error)
	assert.EqualValues(t, 1, parts, "the late chunk is not recorded")
}

func TestReplacingAPartDeletesOnlyTheOldObject(t *testing.T) {
	e := newPlanEnv(t, "plan-replace-chunk@example.com")
	sessionID := initSession(t, e.ts, e.token, 10)
	require.Equal(t, http.StatusCreated, putChunk(e.ts, e.token, sessionID, 1, []byte("aaaaaaaaaa")).StatusCode)
	require.Equal(t, http.StatusCreated, putChunk(e.ts, e.token, sessionID, 1, []byte("bbbbbbbbbb")).StatusCode)
	resp := completeChunked(e.ts, e.token, sessionID)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var result struct {
		File struct {
			ID int64 `json:"id"`
		} `json:"file"`
	}
	parseJSON(resp, &result)
	body, _ := io.ReadAll(doGet(e.ts, fmt.Sprintf("/files/%d/download", result.File.ID), e.token).Body)
	assert.Equal(t, "bbbbbbbbbb", string(body), "the replacement's bytes are assembled, the old object is gone")
}
