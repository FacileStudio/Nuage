package tests

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// chargedTo reads who pays for the space.
func chargedTo(t *testing.T, ts *testServer, spaceID int64) int64 {
	t.Helper()
	var payer int64
	require.NoError(t, ts.db.Model(&schemas.Space{}).Where("id = ?", spaceID).Pluck("charged_to", &payer).Error)
	return payer
}

func TestPlanPayerCannotBeEvictedWithoutAnAcceptedTransfer(t *testing.T) {
	e := newPlanEnv(t, "plan-evict@example.com")
	spaceID := provisionedID(t, e.ts, mintProvisionToken(t, e.ts, e.token, "plume"))
	coID, coToken := registered(t, e.ts, "plan-coowner@example.com")
	memberOf(t, e.ts, spaceID, coID, "owner")
	plainID, plainToken := registered(t, e.ts, "plan-plain@example.com")
	memberOf(t, e.ts, spaceID, plainID, "member")
	var payerMember schemas.SpaceMember
	require.NoError(t, e.ts.db.Where("space_id = ? AND user_id = ?", spaceID, e.userID).First(&payerMember).Error)

	assert.Equal(t, http.StatusConflict, doDelete(e.ts, fmt.Sprintf("/spaces/%d/members/%d", spaceID, payerMember.ID), coToken).StatusCode)
	assert.Equal(t, http.StatusConflict, doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, e.token).StatusCode)

	billing := fmt.Sprintf("/spaces/%d/billing", spaceID)
	accept := billing + "/accept"
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", billing, map[string]int64{"user_id": coID}, coToken).StatusCode)
	assert.Equal(t, http.StatusBadRequest, doJSON(e.ts, "POST", billing, map[string]int64{"user_id": plainID}, e.token).StatusCode)
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", accept, nil, coToken).StatusCode, "no offer yet")

	require.Equal(t, http.StatusNoContent, doJSON(e.ts, "POST", billing, map[string]int64{"user_id": coID}, e.token).StatusCode)
	assert.Equal(t, e.userID, chargedTo(t, e.ts, spaceID), "an offer alone moves nothing")
	assert.Equal(t, http.StatusConflict, doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, e.token).StatusCode)
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", accept, nil, plainToken).StatusCode, "only the offered owner accepts")
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", accept, nil, e.token).StatusCode)

	require.Equal(t, http.StatusNoContent, doJSON(e.ts, "POST", billing, map[string]int64{"user_id": coID}, e.token).StatusCode)
	require.Equal(t, http.StatusNoContent, doJSON(e.ts, "POST", accept, nil, coToken).StatusCode)
	assert.Equal(t, coID, chargedTo(t, e.ts, spaceID))
	assert.Equal(t, http.StatusOK, doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, e.token).StatusCode)
}

func TestPlanRevokedAndExpiredBillingOffersCannotBeAccepted(t *testing.T) {
	e := newPlanEnv(t, "plan-offer@example.com")
	spaceID := provisionedID(t, e.ts, mintProvisionToken(t, e.ts, e.token, "plume"))
	coID, coToken := registered(t, e.ts, "plan-offered@example.com")
	memberOf(t, e.ts, spaceID, coID, "owner")
	billing := fmt.Sprintf("/spaces/%d/billing", spaceID)
	accept := billing + "/accept"

	require.Equal(t, http.StatusNoContent, doJSON(e.ts, "POST", billing, map[string]int64{"user_id": coID}, e.token).StatusCode)
	require.Equal(t, http.StatusNoContent, doDelete(e.ts, billing, e.token).StatusCode)
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", accept, nil, coToken).StatusCode, "a revoked offer cannot be accepted")

	require.Equal(t, http.StatusNoContent, doJSON(e.ts, "POST", billing, map[string]int64{"user_id": coID}, e.token).StatusCode)
	require.NoError(t, e.ts.db.Exec("UPDATE spaces SET billing_offer_end = now() - interval '1 minute' WHERE id = ?", spaceID).Error)
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", accept, nil, coToken).StatusCode, "an expired offer cannot be accepted")
	assert.Equal(t, e.userID, chargedTo(t, e.ts, spaceID))
}

func TestPlanPayerCannotDumpBillingOnAnOwnerTheyAdded(t *testing.T) {
	e := newPlanEnv(t, "plan-dumper@example.com")
	facileID, spaceID := provisionSpace(t, e.ts, mintProvisionToken(t, e.ts, e.token, "ardoise"))
	victimID, victimToken := registered(t, e.ts, "plan-dumped@example.com")
	memberOf(t, e.ts, spaceID, victimID, "owner")

	require.Equal(t, http.StatusNoContent,
		doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/billing", spaceID), map[string]int64{"user_id": victimID}, e.token).StatusCode)
	assert.Equal(t, http.StatusConflict, doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/leave", spaceID), nil, e.token).StatusCode)
	assert.Equal(t, e.userID, chargedTo(t, e.ts, spaceID))

	victimApp := mintScopedToken(t, e.ts, victimToken, "ardoise", facileID)
	assert.Equal(t, http.StatusForbidden, doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/billing/accept", spaceID), nil, victimApp).StatusCode,
		"an app token cannot accept on the user's behalf")
	assert.Equal(t, e.userID, chargedTo(t, e.ts, spaceID))
}

func TestPlanDelegatedTokenCannotAcceptBilling(t *testing.T) {
	delegate := new(int64)
	ts := setupTestServerWith(t, delegatedEnv(), delegatedVerifier{userID: delegate})
	payerID, payerToken := registered(t, ts, "plan-delegating-payer@example.test")
	spaceID := provisionedID(t, ts, mintProvisionToken(t, ts, payerToken, "ardoise"))
	heirID, heirToken := registered(t, ts, "plan-delegating-heir@example.test")
	memberOf(t, ts, spaceID, heirID, "owner")
	*delegate = heirID
	billing := fmt.Sprintf("/spaces/%d/billing", spaceID)
	require.Equal(t, http.StatusNoContent, doJSON(ts, "POST", billing, map[string]int64{"user_id": heirID}, payerToken).StatusCode)

	assert.Equal(t, http.StatusForbidden, doJSON(ts, "POST", billing+"/accept", nil, delegatedWrite).StatusCode)
	assert.Equal(t, payerID, chargedTo(t, ts, spaceID))
	assert.Equal(t, http.StatusNoContent, doJSON(ts, "POST", billing+"/accept", nil, heirToken).StatusCode)
	assert.Equal(t, heirID, chargedTo(t, ts, spaceID))
}

func TestPlanBillingAcceptDoesNotDeadlockWithUploads(t *testing.T) {
	e := newPlanEnv(t, "plan-race@example.com")
	spaceID := provisionedID(t, e.ts, mintProvisionToken(t, e.ts, e.token, "ardoise"))
	heirID, heirToken := registered(t, e.ts, "plan-race-owner@example.com")
	memberOf(t, e.ts, spaceID, heirID, "owner")
	require.Equal(t, http.StatusNoContent,
		doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/billing", spaceID), map[string]int64{"user_id": heirID}, e.token).StatusCode)

	upload := e.ts.db.Begin()
	require.NoError(t, upload.Error)
	defer upload.Rollback()
	require.NoError(t, quota.LockPeople(upload, e.userID), "an upload holds the payer's plan lock")

	accepted := make(chan int, 1)
	go func() {
		accepted <- doJSON(e.ts, "POST", fmt.Sprintf("/spaces/%d/billing/accept", spaceID), nil, heirToken).StatusCode
	}()
	time.Sleep(300 * time.Millisecond)

	id := facile.NewID()
	insert := upload.Create(&schemas.File{FacileID: id, Name: id, BucketKey: id, UploadedBy: e.userID, SpaceID: &spaceID})
	require.NoError(t, insert.Error, "the upload's insert must not wait on accept's row lock (40P01 otherwise)")
	require.NoError(t, upload.Commit().Error)
	assert.Equal(t, http.StatusNoContent, <-accepted)
	assert.Equal(t, heirID, chargedTo(t, e.ts, spaceID))
}
