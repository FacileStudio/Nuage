package tests

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

// TestDemotionDuringRemovalIsSeenByThePeerRule holds the membership lock,
// demotes the caller from owner to admin while their removal request waits on
// it, and then requires the peer rule to refuse. It fails whenever the caller's
// role is resolved before the transaction takes the lock, because the request
// then decides on a role the database no longer holds.
func TestDemotionDuringRemovalIsSeenByThePeerRule(t *testing.T) {
	ts := setupTestServer(t)

	_, founder := registerUser(ts, "race-founder@example.com", "password12345")
	actorID, actor := registerUser(ts, "race-actor@example.com", "password12345")
	victimID, _ := registerUser(ts, "race-victim@example.com", "password12345")

	spaceID := createSpace(t, ts, founder, "Contested Space")
	addSpaceMember(t, ts, founder, spaceID, actorID)
	promoteToOwner(t, ts, founder, spaceID, spaceMemberID(t, ts, founder, spaceID, actorID))
	addSpaceMember(t, ts, founder, spaceID, victimID)
	promoteToAdmin(t, ts, founder, spaceID, spaceMemberID(t, ts, founder, spaceID, victimID))

	victimMemberID := spaceMemberID(t, ts, founder, spaceID, victimID)
	actorRow, err := strconv.ParseInt(actorID, 10, 64)
	require.NoError(t, err)
	victimRow, err := strconv.ParseInt(victimID, 10, 64)
	require.NoError(t, err)

	tx := ts.db.Begin()
	require.NoError(t, tx.Error)
	var locked []schemas.SpaceMember
	require.NoError(t, tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("space_id = ?", spaceID).Find(&locked).Error)

	done := make(chan *http.Response, 1)
	go func() {
		done <- doDelete(ts, fmt.Sprintf("/spaces/%d/members/%d", spaceID, victimMemberID), actor)
	}()

	time.Sleep(500 * time.Millisecond)

	require.NoError(t, tx.Model(&schemas.SpaceMember{}).
		Where("space_id = ? AND user_id = ?", spaceID, actorRow).
		Update("role", "admin").Error)
	require.NoError(t, tx.Commit().Error)

	resp := <-done
	assert.Equal(t, http.StatusForbidden, resp.StatusCode, "a demoted caller must not remove a peer admin")

	var remaining int64
	require.NoError(t, ts.db.Model(&schemas.SpaceMember{}).
		Where("space_id = ? AND user_id = ?", spaceID, victimRow).Count(&remaining).Error)
	assert.Equal(t, int64(1), remaining, "the peer admin must still be a member")
}
