package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/facile"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte/directory"

	"github.com/stretchr/testify/require"
)

const gibibyte int64 = 1 << 30

// planEnv is one user with a Registre identity, ready to be put on a plan.
type planEnv struct {
	ts     *testServer
	userID int64
	token  string
}

// newPlanEnv registers a user and gives them the Registre subject "sub-1".
func newPlanEnv(t *testing.T, email string) planEnv {
	t.Helper()
	ts := setupSettledServer(t)
	rawID, token := registerUser(ts, email, "password12345")
	userID, err := strconv.ParseInt(rawID, 10, 64)
	require.NoError(t, err)
	require.NoError(t, ts.db.Exec(
		"INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)", userID, testIssuer, "sub-1").Error)
	return planEnv{ts: ts, userID: userID, token: token}
}

// enforce applies an authoritative snapshot giving the person and the space
// spc_1 the given storage_gb values. An empty value leaves the scope unset.
func (e planEnv) enforce(t *testing.T, revision int64, personGB, spaceGB string) {
	t.Helper()
	snap := directory.Snapshot{Version: revision, EntitlementsRevision: revision}
	if personGB != "" {
		snap.People = []directory.Person{{Subject: "sub-1", Entitlements: storageGB(personGB)}}
	}
	if spaceGB != "" {
		snap.Spaces = []directory.Space{{ID: "spc_1", Name: "Team", Entitlements: storageGB(spaceGB)}}
	}
	require.NoError(t, e.ts.plans.Apply(context.Background(), snap))
}

// storageGB is an entitlements map holding nuage.storage_gb.
func storageGB(value string) map[string]json.RawMessage {
	return map[string]json.RawMessage{"nuage.storage_gb": json.RawMessage(value)}
}

// linkedSpace creates a space and links it to the Registre space spc_1.
func (e planEnv) linkedSpace(t *testing.T) int64 {
	t.Helper()
	spaceID := createSpace(t, e.ts, e.token, "Team")
	require.NoError(t, e.ts.db.Exec("UPDATE spaces SET registre_space_id = 'spc_1' WHERE id = ?", spaceID).Error)
	return spaceID
}

// fill records a file row of size bytes with no stored object, so a test can
// bring a scope to a few bytes under a GB ceiling without uploading a GB.
func (e planEnv) fill(t *testing.T, spaceID *int64, size int64) {
	t.Helper()
	id := facile.NewID()
	require.NoError(t, e.ts.db.Create(&schemas.File{
		FacileID: id, Name: "filler-" + id, Size: size, BucketKey: fmt.Sprintf("%d/%s", e.userID, id),
		UploadedBy: e.userID, SpaceID: spaceID,
	}).Error)
}

// planError is the 402 body.
type planError struct {
	Error struct {
		Code       string `json:"code"`
		Message    string `json:"message"`
		Key        string `json:"key"`
		Limit      int64  `json:"limit"`
		UpgradeURL string `json:"upgrade_url"`
	} `json:"error"`
}
