package tests

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/mimetype"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/presign"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/files"
	"github.com/FacileStudio/Nuage/apps/api/modules/mcp"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/modules/sharing"
	"github.com/FacileStudio/Nuage/apps/api/modules/spaces"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/FacileStudio/tronc/mcp/mcptest"
	"github.com/go-chi/chi/v5"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpIdentities resolves each test subject to its user and nothing else.
type mcpIdentities struct {
	porte.IdentityStore
	users map[string]int64
}

// Find resolves a known subject.
func (s mcpIdentities) Find(_ context.Context, provider, sub string) (porte.StoredIdentity, error) {
	id, ok := s.users[sub]
	if !ok {
		return porte.StoredIdentity{}, porte.ErrNotFound
	}
	return porte.StoredIdentity{UserID: id, Provider: provider, Subject: sub}, nil
}

// mcpActors marks bulle as a full-access client and plume as a
// registered app.
type mcpActors struct{}

// ActorAccess mirrors NUAGE_APPS=plume and NUAGE_FULL_ACCESS_ACTORS=bulle.
func (mcpActors) ActorAccess(_ context.Context, clientID string) (*schemas.RegisteredApp, bool, error) {
	switch clientID {
	case "bulle":
		return nil, true, nil
	case "plume":
		return &schemas.RegisteredApp{}, false, nil
	}
	return nil, false, nil
}

// mcpFixture is the MCP endpoint over Nuage's real services, for an owner
// and a stranger.
type mcpFixture struct {
	ts       *testServer
	issuer   *mcptest.Issuer
	files    *files.Service
	resource string
	owner    int64
	stranger int64
	session  string
	front    func(http.Handler) http.Handler
}

// newMCPFixture starts the endpoint and the confirm page routes.
func newMCPFixture(t *testing.T) *mcpFixture {
	t.Helper()
	return newMCPFixtureOn(t, "")
}

// newMCPFixtureOn is newMCPFixture on addr, any free port when empty.
func newMCPFixtureOn(t *testing.T, addr string) *mcpFixture {
	t.Helper()
	ts := setupTestServer(t)
	f := &mcpFixture{ts: ts, issuer: mcptest.NewIssuer(t)}
	ownerID, session := registerUser(ts, "owner@example.test", "")
	strangerID, _ := registerUser(ts, "stranger@example.test", "")
	f.owner, _ = strconv.ParseInt(ownerID, 10, 64)
	f.stranger, _ = strconv.ParseInt(strangerID, 10, 64)
	f.session = session

	notifier := nook.NewNotifier(ts.db)
	actLogger := activity.NewLogger(ts.db)
	f.files = files.NewService(ts.db, ts.store, notifier, actLogger, quota.NewService(ts.db).WithPlans(ts.plans), presign.DeriveSecret("test-secret", "nuage-presign-v1"))

	var handler http.Handler
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.front != nil {
			f.front(handler).ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	if addr != "" {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		srv.Listener = listener
	}
	srv.Start()
	t.Cleanup(srv.Close)
	f.resource = srv.URL + "/mcp"
	verifier, err := mcpauth.Verifier(context.Background(), mcpauth.Config{
		Issuer: f.issuer.URL, Resource: f.resource, ScopePrefix: "nuage",
		Identities: mcpIdentities{users: map[string]int64{"owner": f.owner, "stranger": f.stranger}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := ts.db.DB()
	service, err := mcp.NewService(mcp.Deps{
		Files: f.files, Sharing: sharing.NewService(ts.db, notifier, actLogger),
		Spaces: spaces.NewService(ts.db, ts.store, ""), Actors: mcpActors{},
	}, mcp.Config{
		Resource: f.resource, Issuer: f.issuer.URL, Version: "test", StateKey: []byte("state-key"),
		Verifier: verifier, Pending: mcpkit.PostgresPending{DB: sqlDB}, PerMinute: 100_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mcp.RegisterRoutes(router, service)
	router.Route("/api", func(r chi.Router) {
		mcp.RegisterPendingRoutes(r, service, auth.NewService(ts.db, ts.sessions, slog.Default()))
	})
	handler = router
	return f
}

// browserCookie is the owner's session cookie as a browser sign-in sets
// it, the only credential the confirm page takes.
func (f *mcpFixture) browserCookie(t *testing.T) *http.Cookie {
	t.Helper()
	recorder := httptest.NewRecorder()
	if _, _, err := f.ts.sessions.IssueCookie(context.Background(), recorder, httptest.NewRequest(http.MethodGet, f.resource, nil), f.owner); err != nil {
		t.Fatal(err)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("a browser sign-in must set one cookie, got %d", len(cookies))
	}
	return cookies[0]
}

// token is a token for the owner with scopes, or both when none is given.
func (f *mcpFixture) token(t testing.TB, scopes ...string) string {
	if len(scopes) == 0 {
		scopes = []string{"nuage.read", "nuage.write"}
	}
	return f.issuer.Token(t, f.resource, "owner", scopes...)
}

// connect opens a session for subject, with an actor when set.
func (f *mcpFixture) connect(t *testing.T, subject, actor string) *sdk.ClientSession {
	t.Helper()
	claims := map[string]any{"aud": []string{f.resource}, "sub": subject, "scope": "nuage.read nuage.write", "exp": time.Now().Add(time.Hour).Unix()}
	if actor != "" {
		claims["act"] = map[string]any{"sub": actor}
		claims["client_id"] = actor
	}
	return mcptest.Connect(t, f.resource, f.issuer.Mint(t, claims), mcptest.ClientOptions{})
}

// upload stores a personal file of the owner, or one in spaceID.
func (f *mcpFixture) upload(t testing.TB, userID int64, name, content string, spaceID *int64) schemas.File {
	t.Helper()
	file, _, err := f.files.Upload(context.Background(), userID, name, mimetype.Resolve(name, ""), int64(len(content)), strings.NewReader(content), nil, spaceID)
	if err != nil {
		t.Fatal(err)
	}
	return *file
}

// call runs a tool and decodes its result, failing on a tool error.
func call(t *testing.T, session *sdk.ClientSession, tool string, args map[string]any, out any) {
	t.Helper()
	result := mcptest.Call(t, session, tool, args)
	if result.IsError {
		t.Fatalf("%s: %s", tool, mcptest.Text(result))
	}
	mcptest.Decode(t, result, out)
}

// refused runs a tool and returns its error text, failing when it worked.
func refused(t *testing.T, session *sdk.ClientSession, tool string, args map[string]any) string {
	t.Helper()
	result := mcptest.Call(t, session, tool, args)
	if !result.IsError {
		t.Fatalf("%s must be refused, got %s", tool, mcptest.Text(result))
	}
	return mcptest.Text(result)
}

// trashed reports 1 once file id is in the trash.
func (f *mcpFixture) trashed(id int64) int {
	var count int64
	f.ts.db.Model(&schemas.File{}).Where("id = ? AND deleted_at IS NOT NULL", id).Count(&count)
	return int(count)
}

// shares counts the links to file id.
func (f *mcpFixture) shares(id int64) int {
	var count int64
	f.ts.db.Model(&schemas.Share{}).Where("file_id = ?", id).Count(&count)
	return int(count)
}
