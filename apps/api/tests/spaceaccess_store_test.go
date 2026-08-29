package tests

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/internal/spaceaccess"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/spaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedMembership writes a user, a space and the row joining them, and returns
// the two real ids. The space is recreated until its id differs from the user's
// so that a store confusing the two columns produces a visible mismatch.
func seedMembership(t *testing.T, ts *testServer, role string) (int64, int64) {
	t.Helper()

	user := &schemas.User{Email: fmt.Sprintf("store-%s@example.com", facile.NewID()), Name: "store probe"}
	require.NoError(t, ts.db.Create(user).Error)

	space := &schemas.Space{FacileID: facile.NewID(), Name: "Store Probe"}
	for {
		require.NoError(t, ts.db.Create(space).Error)
		if space.ID != user.ID {
			break
		}
		space = &schemas.Space{FacileID: facile.NewID(), Name: "Store Probe"}
	}

	row := &schemas.SpaceMember{SpaceID: space.ID, UserID: user.ID, Role: role}
	require.NoError(t, ts.db.Create(row).Error)

	return space.ID, user.ID
}

// TestStoreMembershipComesFromTheRow pins the property the conformance suite
// structurally cannot see. That suite translates fixture names to row ids and
// back through maps that are inverses, so a store echoing the arguments it was
// handed still lands on the right fixture name and passes every invariant. Here
// the arguments are spelled non-canonically and the expectations are the real
// numeric ids, so echoing an argument is a mismatch rather than a round trip.
func TestStoreMembershipComesFromTheRow(t *testing.T) {
	ts := setupTestServer(t)
	store := spaceaccess.NewStore(ts.db)

	spaceID, userID := seedMembership(t, ts, "admin")
	space := strconv.FormatInt(spaceID, 10)
	user := strconv.FormatInt(userID, 10)

	got, err := store.Membership(context.Background(), "0"+space, "0"+user)
	require.NoError(t, err)

	assert.Equal(t, space, got.SpaceID, "SpaceID must be the row's space_id, not the argument")
	assert.Equal(t, user, got.UserID, "UserID must be the row's user_id, not the argument")
	assert.Equal(t, spaces.Role("admin"), got.Role)
}

// TestStoreMembershipsComeFromTheRows is the list counterpart: every field of
// every returned Membership has to equal what its row holds.
func TestStoreMembershipsComeFromTheRows(t *testing.T) {
	ts := setupTestServer(t)
	store := spaceaccess.NewStore(ts.db)

	spaceID, userID := seedMembership(t, ts, "owner")
	space := strconv.FormatInt(spaceID, 10)
	user := strconv.FormatInt(userID, 10)

	second := &schemas.Space{FacileID: facile.NewID(), Name: "Second Probe"}
	require.NoError(t, ts.db.Create(second).Error)
	require.NoError(t, ts.db.Create(&schemas.SpaceMember{SpaceID: second.ID, UserID: userID, Role: "member"}).Error)

	rows, err := store.Memberships(context.Background(), "0"+user)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	bySpace := map[string]spaces.Membership{}
	for _, row := range rows {
		bySpace[row.SpaceID] = row
	}

	require.Contains(t, bySpace, space)
	assert.Equal(t, user, bySpace[space].UserID, "UserID must be the row's user_id, not the argument")
	assert.Equal(t, spaces.Role("owner"), bySpace[space].Role)

	other := strconv.FormatInt(second.ID, 10)
	require.Contains(t, bySpace, other)
	assert.Equal(t, user, bySpace[other].UserID, "UserID must be the row's user_id, not the argument")
	assert.Equal(t, spaces.Role("member"), bySpace[other].Role)

	count, err := store.CountRole(context.Background(), "0"+space, spaces.RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

// TestStoreMembershipRejectsUnparseableIDs keeps the string-to-int64 boundary
// answering ErrNotMember instead of leaking a driver error.
func TestStoreMembershipRejectsUnparseableIDs(t *testing.T) {
	ts := setupTestServer(t)
	store := spaceaccess.NewStore(ts.db)

	_, err := store.Membership(context.Background(), "not-a-number", "1")
	assert.ErrorIs(t, err, spaces.ErrNotMember)

	_, err = store.Membership(context.Background(), "1", "not-a-number")
	assert.ErrorIs(t, err, spaces.ErrNotMember)
}
