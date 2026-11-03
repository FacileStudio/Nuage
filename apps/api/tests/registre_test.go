package tests

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/modules/spaces"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const registreIssuer = "https://sso.example.test"

// registreSeed is the local state before Registre answers: one user-made space "team" and
// one space Plume provisioned under the same name, plus the people below.
type registreSeed struct {
	team, provisioned int64
	carolToken        string
}

func newTestRegistre(db *gorm.DB) *spaces.Registre {
	return spaces.NewRegistre(db, registreIssuer, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// seedRegistre builds "team" holding alice (Registre identity, owner), carol (local password
// account, admin) and dave (Registre identity, admin, removed in Registre), and Plume's
// "Team" holding dave alone. bob exists with no identity and no membership.
func seedRegistre(t *testing.T, ts *testServer) registreSeed {
	t.Helper()
	carolID, carolToken := registerUser(ts, "carol@x.test", "password12345")
	ids := map[string]int64{"carol": mustID(t, carolID)}
	for _, name := range []string{"alice", "bob", "dave"} {
		user := schemas.User{Email: name + "@x.test", Color: "blue"}
		require.NoError(t, ts.db.Create(&user).Error)
		ids[name] = user.ID
	}
	require.NoError(t, ts.db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, 'sub-alice'), (?, ?, 'sub-dave')`,
		ids["alice"], registreIssuer, ids["dave"], registreIssuer).Error)
	seed := registreSeed{carolToken: carolToken}
	seed.team = seedSpace(t, ts.db, "team", "", map[int64]string{ids["alice"]: "owner", ids["carol"]: "admin", ids["dave"]: "admin"})
	seed.provisioned = seedSpace(t, ts.db, "Team", "plume", map[int64]string{ids["dave"]: "owner"})
	return seed
}

func mustID(t *testing.T, raw string) int64 {
	t.Helper()
	id, err := strconv.ParseInt(raw, 10, 64)
	require.NoError(t, err)
	return id
}

func seedSpace(t *testing.T, db *gorm.DB, name, provisionedBy string, members map[int64]string) int64 {
	t.Helper()
	space := schemas.Space{FacileID: facile.NewID(), Name: name, ProvisionedBy: provisionedBy}
	require.NoError(t, db.Create(&space).Error)
	for userID, role := range members {
		require.NoError(t, db.Omit("Space", "User").Create(&schemas.SpaceMember{SpaceID: space.ID, UserID: userID, Role: role}).Error)
	}
	return space.ID
}

func teamSnapshot() directory.Snapshot {
	return directory.Snapshot{Version: 7, Spaces: []directory.Space{
		{ID: "r-team", Name: "Team", Members: []directory.Member{
			{Subject: "sub-alice", Email: "alice@x.test", Role: directory.RoleOwner},
			{Subject: "sub-bob", Email: "BOB@x.test", Role: directory.RoleAdmin},
			{Subject: "sub-erin", Email: "erin@x.test", Name: "Erin", Picture: registreIssuer + "/avatars/sub-erin", Role: "viewer"},
		}},
		{ID: "r-fresh", Name: "Fresh", Members: []directory.Member{
			{Subject: "sub-alice", Email: "alice@x.test", Role: directory.RoleMember},
		}},
	}}
}

// membershipState is every row Apply can touch, keyed "<registre id>:<name>/<email>" with
// the role and row id, so two applies can be compared row for row.
func membershipState(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()
	var rows []struct {
		ID         int64
		Email, Key string
		Role       string
	}
	require.NoError(t, db.Raw(`SELECT m.id, u.email, m.role, COALESCE(s.registre_space_id, s.provisioned_by) || ':' || s.name AS key
		FROM space_members m JOIN users u ON u.id = m.user_id JOIN spaces s ON s.id = m.space_id`).Scan(&rows).Error)
	state := map[string]string{}
	for _, row := range rows {
		state[row.Key+"/"+row.Email] = fmt.Sprintf("%s#%d", row.Role, row.ID)
	}
	return state
}

func roleIn(state map[string]string, key string) string {
	role, _, _ := strings.Cut(state[key], "#")
	return role
}

func TestRegistreApplyMirrorsTheSnapshot(t *testing.T) {
	ts := setupTestServer(t)
	seed := seedRegistre(t, ts)
	registre := newTestRegistre(ts.db)
	require.NoError(t, registre.Apply(context.Background(), teamSnapshot()))

	var team, plume schemas.Space
	require.NoError(t, ts.db.First(&team, seed.team).Error)
	require.NoError(t, ts.db.First(&plume, seed.provisioned).Error)
	require.NotNil(t, team.RegistreSpaceID, "the user-made space links by name")
	assert.Equal(t, "r-team", *team.RegistreSpaceID)
	assert.Nil(t, plume.RegistreSpaceID, "a provisioned space never links")

	state := membershipState(t, ts.db)
	want := map[string]string{
		"r-team:Team/alice@x.test":   "owner",
		"r-team:Team/bob@x.test":     "admin",
		"r-team:Team/erin@x.test":    "member",
		"r-team:Team/carol@x.test":   "admin",
		"r-fresh:Fresh/alice@x.test": "member",
		"plume:Team/dave@x.test":     "owner",
	}
	require.Len(t, state, len(want), "dave leaves team; carol, a local account, stays: %v", state)
	for key, role := range want {
		assert.Equal(t, role, roleIn(state, key), key)
	}
	assertRegistreProvisioned(t, ts.db)

	require.NoError(t, registre.Apply(context.Background(), teamSnapshot()))
	assert.Equal(t, state, membershipState(t, ts.db), "a second apply changes nothing")
	var users int64
	ts.db.Model(&schemas.User{}).Count(&users)
	assert.EqualValues(t, 5, users, "erin is provisioned once")

	assertLinkedSpaceRefusesEdits(t, ts, seed)
}

func assertRegistreProvisioned(t *testing.T, db *gorm.DB) {
	t.Helper()
	var erin schemas.User
	require.NoError(t, db.First(&erin, "email = ?", "erin@x.test").Error)
	assert.Equal(t, "Erin", erin.Name)
	assert.Equal(t, registreIssuer+"/avatars/sub-erin", erin.OIDCPictureURL)
	assert.NotEmpty(t, erin.Color)
	assert.False(t, erin.IsAdmin)
	for subject, email := range map[string]string{"sub-erin": "erin@x.test", "sub-bob": "bob@x.test"} {
		var owner []string
		db.Raw(`SELECT u.email FROM porte_identities i JOIN users u ON u.id = i.user_id
			WHERE i.provider = ? AND i.subject = ?`, registreIssuer, subject).Scan(&owner)
		assert.Equal(t, []string{email}, owner, subject)
	}
}

// assertLinkedSpaceRefusesEdits checks that carol, an admin of the linked space, can change
// its description but not its name, members or her own membership.
func assertLinkedSpaceRefusesEdits(t *testing.T, ts *testServer, seed registreSeed) {
	t.Helper()
	base := fmt.Sprintf("/spaces/%d", seed.team)
	refused := map[string]*http.Response{
		"rename": doJSON(ts, "PUT", base, map[string]string{"name": "Renamed"}, seed.carolToken),
		"add":    doJSON(ts, "POST", base+"/members", map[string]any{"user_id": 1, "role": "member"}, seed.carolToken),
		"leave":  doJSON(ts, "POST", base+"/leave", nil, seed.carolToken),
	}
	for action, resp := range refused {
		assert.Equal(t, http.StatusConflict, resp.StatusCode, action)
	}
	described := doJSON(ts, "PUT", base, map[string]string{"description": "Shared drive"}, seed.carolToken)
	assert.Equal(t, http.StatusOK, described.StatusCode)
}
