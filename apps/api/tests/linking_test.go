package tests

import (
	"context"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func signIn(subject, email string, verified bool) porte.Claims {
	return porte.Claims{Provider: testIssuer, Subject: subject, Email: email, EmailVerified: verified, Locale: "de"}
}

func link(t *testing.T, ts *testServer, userID int64, subject string) {
	t.Helper()
	require.NoError(t, ts.db.Exec(
		`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, ?, ?)`, userID, testIssuer, subject).Error)
}

func TestFirstSignInLinksAnExistingAccountByVerifiedEmail(t *testing.T) {
	ts := setupTestServer(t)
	store := auth.NewUserStore(ts.db, nil)
	ctx := context.Background()
	existing := schemas.User{Email: "camille@facile.studio", Name: "Camille"}
	require.NoError(t, ts.db.Create(&existing).Error)

	id, err := store.UpsertFromOIDC(ctx, signIn("sub-1", "Camille@Facile.Studio", true))
	require.NoError(t, err)
	assert.Equal(t, existing.ID, id)
	link(t, ts, id, "sub-1")

	again, err := store.UpsertFromOIDC(ctx, signIn("sub-1", "camille@facile.studio", true))
	require.NoError(t, err)
	assert.Equal(t, existing.ID, again)

	var user schemas.User
	require.NoError(t, ts.db.First(&user, id).Error)
	assert.Equal(t, "de", user.Locale)
}

func TestAnUnverifiedAddressNeverLinksToAnExistingAccount(t *testing.T) {
	ts := setupTestServer(t)
	require.NoError(t, ts.db.Create(&schemas.User{Email: "noah@facile.studio", Name: "Noah"}).Error)

	_, err := auth.NewUserStore(ts.db, nil).UpsertFromOIDC(context.Background(), signIn("sub-2", "noah@facile.studio", false))
	assert.Error(t, err)
}

func TestARecycledAddressIsNotHandedToANewSubject(t *testing.T) {
	ts := setupTestServer(t)
	owner := schemas.User{Email: "lea@facile.studio", Name: "Lea"}
	require.NoError(t, ts.db.Create(&owner).Error)
	link(t, ts, owner.ID, "sub-owner")

	_, err := auth.NewUserStore(ts.db, nil).UpsertFromOIDC(context.Background(), signIn("sub-newcomer", "lea@facile.studio", true))
	assert.Error(t, err)
}

func TestANewSignInCreatesAnAccountWithItsLocale(t *testing.T) {
	ts := setupTestServer(t)
	id, err := auth.NewUserStore(ts.db, nil).UpsertFromOIDC(context.Background(), signIn("sub-3", "new@facile.studio", true))
	require.NoError(t, err)

	var user schemas.User
	require.NoError(t, ts.db.First(&user, id).Error)
	assert.Equal(t, "de", user.Locale)
}
