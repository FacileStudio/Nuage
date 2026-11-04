package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// provisionedID provisions a space with an app token and returns its id.
func provisionedID(t *testing.T, ts *testServer, appToken string) int64 {
	t.Helper()
	_, id := provisionSpace(t, ts, appToken)
	return id
}

func TestPlanProvisionedSpacesShareTheCreatorsPersonalPlan(t *testing.T) {
	e := newPlanEnv(t, "plan-provision@example.com")
	app := mintProvisionToken(t, e.ts, e.token, "ardoise")
	first, second := provisionedID(t, e.ts, app), provisionedID(t, e.ts, app)

	var charged []int64
	require.NoError(t, e.ts.db.Model(&schemas.Space{}).Where("id IN ?", []int64{first, second}).Pluck("charged_to", &charged).Error)
	assert.Equal(t, []int64{e.userID, e.userID}, charged)

	e.enforce(t, 1, "1", "")
	e.fill(t, nil, gibibyte-30)
	assert.Equal(t, http.StatusCreated, uploadFileToSpace(e.ts, app, "a.pdf", "0123456789", first).StatusCode)
	assert.Equal(t, http.StatusCreated, uploadFileToSpace(e.ts, app, "b.pdf", "0123456789", second).StatusCode)
	assert.Equal(t, http.StatusCreated, uploadFile(e.ts, e.token, "mine.txt", "0123456789", nil).StatusCode)

	assert.Equal(t, http.StatusPaymentRequired, uploadFileToSpace(e.ts, app, "c.pdf", "x", first).StatusCode)
	assert.Equal(t, http.StatusPaymentRequired, uploadFileToSpace(e.ts, app, "d.pdf", "x", second).StatusCode)
	assert.Equal(t, http.StatusPaymentRequired, uploadFile(e.ts, e.token, "more.txt", "x", nil).StatusCode)

	var usage struct {
		StorageUsed int64 `json:"storage_used"`
	}
	parseJSON(doGet(e.ts, "/quota/me", e.token), &usage)
	assert.Equal(t, gibibyte, usage.StorageUsed, "the meter counts the provisioned spaces too")
}

func TestPlanStrangerCannotChargeSomeoneElse(t *testing.T) {
	e := newPlanEnv(t, "plan-victim@example.com")
	victimApp := mintProvisionToken(t, e.ts, e.token, "ardoise")
	victimSpace := ensureSpace(e.ts, victimApp, facile.NewID())
	require.Equal(t, http.StatusCreated, victimSpace.StatusCode)
	var space provisioned
	parseJSON(victimSpace, &space)

	_, strangerToken := registerUser(e.ts, "plan-stranger@example.com", "password12345")
	strangerApp := mintProvisionToken(t, e.ts, strangerToken, "ardoise")
	e.enforce(t, 1, "1", "")

	assert.Equal(t, http.StatusNotFound, ensureSpace(e.ts, strangerApp, space.FacileID).StatusCode)
	assert.NotEqual(t, http.StatusCreated, uploadFileToSpace(e.ts, strangerApp, "x.pdf", "stranger bytes", space.ID).StatusCode)
	assert.NotEqual(t, http.StatusCreated, uploadFileToSpace(e.ts, strangerToken, "y.pdf", "stranger bytes", space.ID).StatusCode)

	own := provisionedID(t, e.ts, strangerApp)
	var chargedTo int64
	require.NoError(t, e.ts.db.Model(&schemas.Space{}).Where("id = ?", own).Pluck("charged_to", &chargedTo).Error)
	assert.NotEqual(t, e.userID, chargedTo)

	var victimBytes int64
	require.NoError(t, e.ts.db.Model(&schemas.File{}).Where("space_id = ?", space.ID).Count(&victimBytes).Error)
	assert.Zero(t, victimBytes)
}

func TestPlanBackfillChargesProvisionedSpacesToTheirOwner(t *testing.T) {
	e := newPlanEnv(t, "plan-backfill@example.com")
	app := mintProvisionToken(t, e.ts, e.token, "plume")
	spaceID := provisionedID(t, e.ts, app)
	require.NoError(t, e.ts.db.Exec("UPDATE spaces SET charged_to = NULL WHERE id = ?", spaceID).Error)

	require.NoError(t, schemas.Migrate(e.ts.db))
	var chargedTo int64
	require.NoError(t, e.ts.db.Model(&schemas.Space{}).Where("id = ?", spaceID).Pluck("charged_to", &chargedTo).Error)
	assert.Equal(t, e.userID, chargedTo)

	orphan := provisionedID(t, e.ts, app)
	require.NoError(t, e.ts.db.Exec("UPDATE spaces SET charged_to = NULL WHERE id = ?", orphan).Error)
	require.NoError(t, e.ts.db.Exec("DELETE FROM space_members WHERE space_id = ?", orphan).Error)
	require.NoError(t, schemas.Migrate(e.ts.db))
	e.enforce(t, 1, "100", "")
	require.NoError(t, e.ts.db.Exec("INSERT INTO space_members (space_id, user_id, role) VALUES (?, ?, 'member')", orphan, e.userID).Error)
	assert.Equal(t, http.StatusPreconditionFailed, uploadFileToSpace(e.ts, app, "z.pdf", "no payer", orphan).StatusCode)
}

// memberOf adds userID to the space with role and returns the membership id.
func memberOf(t *testing.T, ts *testServer, spaceID, userID int64, role string) int64 {
	t.Helper()
	member := schemas.SpaceMember{SpaceID: spaceID, UserID: userID, Role: role}
	require.NoError(t, ts.db.Create(&member).Error)
	return member.ID
}

// registered registers a user and returns its id and token.
func registered(t *testing.T, ts *testServer, email string) (int64, string) {
	t.Helper()
	rawID, token := registerUser(ts, email, "password12345")
	var id int64
	_, err := fmt.Sscan(rawID, &id)
	require.NoError(t, err)
	return id, token
}

func TestPlanPayerOverrideStillCapsCoMembers(t *testing.T) {
	e := newPlanEnv(t, "plan-payer@example.com")
	spaceID := provisionedID(t, e.ts, mintProvisionToken(t, e.ts, e.token, "ardoise"))
	memberID, memberToken := registered(t, e.ts, "plan-comember@example.com")
	memberOf(t, e.ts, spaceID, memberID, "member")
	resp := doJSON(e.ts, "PUT", fmt.Sprintf("/quota/users/%d", e.userID), map[string]int64{"storage_limit": 1000}, e.token)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	e.enforce(t, 1, "100", "")

	assert.Equal(t, http.StatusCreated, uploadFileToSpace(e.ts, memberToken, "a.bin", strings.Repeat("a", 600), spaceID).StatusCode)
	assert.Equal(t, http.StatusRequestEntityTooLarge,
		uploadFileToSpace(e.ts, memberToken, "b.bin", strings.Repeat("b", 600), spaceID).StatusCode)
	assert.Equal(t, http.StatusRequestEntityTooLarge, uploadFile(e.ts, e.token, "c.bin", strings.Repeat("c", 600), nil).StatusCode)
}

func TestPlanDeletedPayerFailsClosedUntilBackfill(t *testing.T) {
	ts := setupSettledServer(t)
	registered(t, ts, "plan-first@example.com")
	payerID, payerToken := registered(t, ts, "plan-gone@example.com")
	spaceID := provisionedID(t, ts, mintProvisionToken(t, ts, payerToken, "ardoise"))
	coID, coToken := registered(t, ts, "plan-heir@example.com")
	memberOf(t, ts, spaceID, coID, "owner")
	require.NoError(t, ts.db.Exec("DELETE FROM users WHERE id = ?", payerID).Error)
	require.NoError(t, ts.plans.Apply(context.Background(), directory.Snapshot{Version: 1, EntitlementsRevision: 1}))

	assert.Equal(t, http.StatusPreconditionFailed, uploadFileToSpace(ts, coToken, "a.pdf", "orphaned", spaceID).StatusCode)
	require.NoError(t, schemas.Migrate(ts.db))
	assert.Equal(t, coID, chargedTo(t, ts, spaceID))
	assert.Equal(t, http.StatusCreated, uploadFileToSpace(ts, coToken, "b.pdf", "inherited", spaceID).StatusCode)
}
