package tests

import (
	"net/http"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

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
