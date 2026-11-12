package tests

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/erasure"
	"github.com/FacileStudio/Nuage/apps/api/modules/trash"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/oidc"
	jwtpkg "github.com/FacileStudio/porte/oidc/jwt"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const erasedSubject = "sub-gone@example.com"

type registreRoundTrip func(*http.Request) (*http.Response, error)

func (f registreRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// registreErasing is a Registre whose deletion list always names erasedSubject.
func registreErasing() http.RoundTripper {
	return registreRoundTrip(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		if r.URL.Path != "/api/directory/deletions" {
			recorder.WriteHeader(http.StatusNotFound)
			return recorder.Result(), nil
		}
		recorder.Header().Set("Content-Type", "application/json")
		body := `{"deleted":[{"sub":"` + erasedSubject + `","deleted_at":"` + time.Now().UTC().Format(time.RFC3339Nano) + `"}]}`
		if _, err := recorder.WriteString(body); err != nil {
			return nil, err
		}
		return recorder.Result(), nil
	})
}

// polledReceiver is the receiver as main.go builds it, over a Registre that
// lists erasedSubject, after one poll has erased it.
func polledReceiver(t *testing.T, f erasureFixture) *erasure.Deletions {
	t.Helper()
	sqlDB, err := f.ts.db.DB()
	require.NoError(t, err)
	store := portepg.New(sqlDB)
	client := directory.Client{Issuer: testIssuer, HTTP: &http.Client{Transport: registreErasing()}}
	deletions, err := erasure.New(context.Background(), erasure.Config{
		ORM: f.ts.db, Client: client, Erase: store.EraseIdentity, Holds: store.HoldsIdentity,
		Files: trash.NewService(f.ts.db, f.ts.store, nil, nil), Apps: f.ts.appFolders, StorageDir: t.TempDir(),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, err)
	ledger := erasure.Ledger(f.ts.db, client)
	pollUntil(t, deletions, func() bool {
		erased, err := ledger.Erased(context.Background(), testIssuer, erasedSubject)
		return err == nil && erased
	})
	return deletions
}

// pollUntil runs the receiver's poll loop, which polls at once, until done
// holds, then stops it.
func pollUntil(t *testing.T, deletions *erasure.Deletions, done func() bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan struct{})
	go func() {
		deletions.Run(ctx)
		close(finished)
	}()
	assert.Eventually(t, done, 30*time.Second, 50*time.Millisecond)
	cancel()
	<-finished
}

// A re-polled notice erases an account that came back for an erased subject,
// without a restart, and leaves everybody else alone.
func TestARecreatedAccountIsErasedAgainOnTheNextPoll(t *testing.T) {
	f := seedErasure(t)
	deletions := polledReceiver(t, f)
	again, _ := registerUser(f.ts, "gone-again@example.com", "")
	againID, err := strconv.ParseInt(again, 10, 64)
	require.NoError(t, err)
	link(t, f.ts, againID, erasedSubject)
	others := count(t, f.ts, `SELECT count(*) FROM users WHERE id <> ?`, againID)

	pollUntil(t, deletions, func() bool {
		var n int64
		return f.ts.db.Raw(`SELECT count(*) FROM users WHERE id = ?`, againID).Scan(&n).Error == nil && n == 0
	})

	assert.EqualValues(t, 0, count(t, f.ts, `SELECT count(*) FROM porte_identities WHERE subject = ?`, erasedSubject))
	assert.Equal(t, others, count(t, f.ts, `SELECT count(*) FROM users`))
	assert.EqualValues(t, 1, count(t, f.ts, `SELECT count(*) FROM users WHERE id = ?`, f.other))
}

// The space sync skips a member Nuage erased instead of provisioning them
// again from a snapshot that still lists them.
func TestTheSpaceSyncSkipsAnErasedMember(t *testing.T) {
	f := seedErasure(t)
	polledReceiver(t, f)
	registre := newTestRegistre(f.ts.db).WithErased(erasure.Ledger(f.ts.db, directory.Client{Issuer: registreIssuer}))
	stale := directory.Snapshot{Version: 1, Spaces: []directory.Space{
		{ID: "r-erased", Name: "Erased", Members: []directory.Member{
			{Subject: erasedSubject, Email: "gone@example.com", Role: directory.RoleMember},
		}},
	}}

	require.NoError(t, registre.Apply(context.Background(), stale))

	assert.EqualValues(t, 0, count(t, f.ts, `SELECT count(*) FROM users WHERE lower(email) = 'gone@example.com'`))
}

// discoveryOnly is a Registre that only answers discovery, enough for oidc.New.
func discoveryOnly(t *testing.T) string {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                 server.URL,
			"authorization_endpoint": server.URL + "/authorize",
			"token_endpoint":         server.URL + "/token",
			"jwks_uri":               server.URL + "/keys",
		})
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// The OIDC kit with the erasure ledger as Deps.Erased, as main.go builds it,
// refuses to provision an account for an erased subject from a bearer token
// (the MCP first call), and creates nothing.
func TestBearerProvisioningRefusesAnErasedSubject(t *testing.T) {
	f := seedErasure(t)
	polledReceiver(t, f)
	issuer := discoveryOnly(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sqlDB, err := f.ts.db.DB()
	require.NoError(t, err)
	store := portepg.New(sqlDB)
	cfg := porte.Config{
		Issuer: issuer, ClientID: "nuage", ClientSecret: "secret",
		RedirectURL: "https://nuage.test/api/auth/oidc/callback", SuccessURL: "https://nuage.test",
	}
	sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: logger})
	require.NoError(t, err)
	kit, err := oidc.New(context.Background(), cfg, oidc.Deps{
		Users: userStore(f.ts.db), Identities: store.Identities(), Sessions: sessions, Codes: store.LoginCodes(),
		Logger: logger, Erased: erasure.Ledger(f.ts.db, directory.Client{Issuer: issuer}),
	})
	require.NoError(t, err)
	before := count(t, f.ts, `SELECT count(*) FROM users`)
	verified := true

	_, err = kit.ProvisionBearer(context.Background(), jwtpkg.Claims{
		Subject: erasedSubject, Email: "gone@example.com", Profile: jwtpkg.Profile{EmailVerified: &verified},
	})

	assert.True(t, stderrors.Is(err, porte.ErrSubjectErased), "ProvisionBearer = %v", err)
	assert.Equal(t, before, count(t, f.ts, `SELECT count(*) FROM users`))
}
