package users

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/local"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// harness is the real users module over a real Postgres: porte's own stores,
// its session manager and its local kit, mounted on the router the app mounts.
//
// The password path cannot be tested any smaller than this. ChangePassword
// rotates the caller's session by writing a cookie through the response
// writer, and it reads the caller's session id out of the context porte's
// middleware puts there, so a service-level test holding a bare context would
// exercise neither and still pass.
type harness struct {
	db       *gorm.DB
	router   chi.Router
	sessions *session.Manager
	kit      *local.Kit
}

// newHarness follows schemas.openTestDatabase: a schema of its own per test, a
// skip for a developer with no Postgres and a hard failure for CI, where a
// green run must never mean "nothing was tested".
func newHarness(t *testing.T) *harness {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://nuage:nuage-internal-db@localhost:5432/nuage_test?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		unavailable(t, err)
	}

	schema := fmt.Sprintf("password_test_%d", time.Now().UnixNano())
	if err := db.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		unavailable(t, err)
	}
	t.Cleanup(func() { db.Exec("DROP SCHEMA " + schema + " CASCADE") })
	if err := db.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql handle: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := portepg.New(sqlDB)
	sessions, err := session.New(porte.Config{}, session.Deps{Sessions: store.Sessions(), Logger: logger})
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	accounts := auth.NewUserStore(db, nil)
	kit, err := local.New(local.Config{AllowRegistration: true, MinPasswordLength: 12}, local.Deps{
		Users:      accounts,
		Identities: store.Identities(),
		Sessions:   sessions,
		Logger:     logger,
		Count:      accounts.CountUsers,
	})
	if err != nil {
		t.Fatalf("local.New: %v", err)
	}
	authService := auth.NewService(db, sessions, kit, logger)

	router := chi.NewRouter()
	sessions.Mount(router)
	RegisterRoutes(router, NewService(db, t.TempDir(), authService), authService)
	return &harness{db: db, router: router, sessions: sessions, kit: kit}
}

func unavailable(t *testing.T, err error) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("integration test infrastructure required in CI: %v", err)
	}
	t.Skipf("skipping: database not available: %v", err)
}

// register creates an account with a password and returns its id and a bearer
// token for it.
func (h *harness) register(t *testing.T, email, password string) (int64, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/register", nil)
	userID, token, err := h.kit.Register(request.Context(), recorder, request, email, "", password)
	if err != nil {
		t.Fatalf("register %s: %v", email, err)
	}
	return userID, token
}

// seedFederated is an SSO-only account: a user row and no local identity, which
// is what SetPassword still exists for.
func (h *harness) seedFederated(t *testing.T, id int64, email string) string {
	t.Helper()
	if err := h.db.Exec(
		`INSERT INTO users (id, email, name, created_at) VALUES (?, ?, 'Noah', now())`, id, email,
	).Error; err != nil {
		t.Fatalf("seed a federated account: %v", err)
	}
	return h.login(t, id)
}

// login mints a second bearer token for an account, standing in for the other
// browser a password change is supposed to sign out.
func (h *harness) login(t *testing.T, userID int64) string {
	t.Helper()
	token, _, err := h.sessions.Issue(t.Context(), userID, "")
	if err != nil {
		t.Fatalf("issue a session: %v", err)
	}
	return token
}

// patchMe sends one PATCH /users/me as the holder of token.
func (h *harness) patchMe(t *testing.T, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPatch, "/users/me", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

func (h *harness) authenticates(t *testing.T, token string) bool {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder.Code == http.StatusOK
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the error body %q: %v", recorder.Body.String(), err)
	}
	return body.Error.Code
}

// TestChangingAPasswordRequiresTheCurrentOne is the regression this upgrade
// exists for. Until porte v0.3.0 one method served both "add a first password"
// and "replace an existing one", so anybody holding a session — a borrowed
// laptop, a stolen cookie — could replace the password without knowing it,
// which OWASP ASVS forbids at L1 (v4 2.1.6, v5 6.2.3).
//
// The old body is the interesting half: it must be refused, and refused in a
// way that names what is missing rather than 500ing or, worse, succeeding.
func TestChangingAPasswordRequiresTheCurrentOne(t *testing.T) {
	h := newHarness(t)
	_, token := h.register(t, "camille@facile.studio", "correct horse battery")

	recorder := h.patchMe(t, token, `{"password":"battery staple xyz"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("the old body was accepted: got %d %s", recorder.Code, recorder.Body.String())
	}
	if code := errorCode(t, recorder); code != "invalid_argument" {
		t.Fatalf("error code = %q, want invalid_argument", code)
	}
	if !strings.Contains(recorder.Body.String(), "current_password") {
		t.Fatalf("the refusal does not name the missing field: %s", recorder.Body.String())
	}

	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err != nil {
		t.Fatalf("the password was changed anyway: %v", err)
	}
	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "battery staple xyz"); err == nil {
		t.Fatal("the new password works, so the change went through without the current one")
	}
	if !h.authenticates(t, token) {
		t.Fatal("a refused change ended the caller's session")
	}
}

// A wrong current password is 401 and moves nothing. It is 401 and not 400
// because the request is well formed: the credential inside it is wrong.
func TestChangingAPasswordRejectsTheWrongCurrentOne(t *testing.T) {
	h := newHarness(t)
	_, token := h.register(t, "camille@facile.studio", "correct horse battery")

	recorder := h.patchMe(t, token, `{"password":"battery staple xyz","current_password":"not it at all"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", recorder.Code, recorder.Body.String())
	}
	if code := errorCode(t, recorder); code != "unauthenticated" {
		t.Fatalf("error code = %q, want unauthenticated", code)
	}
	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err != nil {
		t.Fatalf("the password moved on a wrong confirmation: %v", err)
	}
}

// The whole path once the current password is there: the new password works,
// the old one stops, the account's other logins end, named API tokens survive,
// and the caller's own session is rotated rather than dropped. porte writes the
// replacement cookie through the response writer, which is why this handler
// holds one; the same credential comes back in the body because this client
// keeps a bearer token in localStorage and the one it holds is now dead.
func TestChangingAPasswordWithTheCurrentOneRotatesTheSession(t *testing.T) {
	h := newHarness(t)
	userID, token := h.register(t, "camille@facile.studio", "correct horse battery")
	otherBrowser := h.login(t, userID)
	apiToken, _, err := h.sessions.Issue(t.Context(), userID, "CLI")
	if err != nil {
		t.Fatalf("issue an api token: %v", err)
	}

	recorder := h.patchMe(t, token, `{"password":"battery staple xyz","current_password":"correct horse battery"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "battery staple xyz"); err != nil {
		t.Fatalf("the new password does not work: %v", err)
	}
	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err == nil {
		t.Fatal("the old password still works")
	}

	var replacement string
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == porte.SessionCookieName && cookie.Value != "" {
			replacement = cookie.Value
		}
	}
	if replacement == "" {
		t.Fatal("no replacement cookie was written, so the browser that made the change holds a revoked token")
	}
	if !h.authenticates(t, replacement) {
		t.Fatal("the replacement session does not authenticate")
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode the response: %v", err)
	}
	if body.Token != replacement {
		t.Fatalf("the rotated token did not come back for the bearer client: %q", body.Token)
	}

	if h.authenticates(t, otherBrowser) {
		t.Fatal("another browser survived the password change")
	}
	if !h.authenticates(t, apiToken) {
		t.Fatal("a named API token was revoked by a password change")
	}
}

// An account with no password gets one without confirming anything, because
// there is nothing to confirm. This is the branch SetPassword still serves — a
// federated user adding a password — and the guard above must not have closed
// it.
func TestFirstPasswordNeedsNoConfirmation(t *testing.T) {
	h := newHarness(t)
	token := h.seedFederated(t, 91, "noah@facile.studio")

	recorder := h.patchMe(t, token, `{"password":"battery staple xyz"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if _, err := h.kit.Verify(t.Context(), "noah@facile.studio", "battery staple xyz"); err != nil {
		t.Fatalf("the first password does not work: %v", err)
	}
	if !h.authenticates(t, token) {
		t.Fatal("setting a first password ended the caller's session")
	}
}

// Confirming a password an account does not have is 400, not 401: there is no
// credential to be wrong about, and "invalid credentials" would send a
// federated user hunting for a password they never set.
func TestConfirmingAPasswordThatDoesNotExist(t *testing.T) {
	h := newHarness(t)
	token := h.seedFederated(t, 92, "noah@facile.studio")

	recorder := h.patchMe(t, token, `{"password":"battery staple xyz","current_password":"anything at all"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	if code := errorCode(t, recorder); code != "invalid_argument" {
		t.Fatalf("error code = %q, want invalid_argument", code)
	}
}

// A password below the floor is 400 on both writes. It is worth asserting on
// the change too: porte checks the length *after* confirming the current
// password, so an app that mapped its errors by position would answer 401.
func TestAShortPasswordIsRefusedOnBothWrites(t *testing.T) {
	h := newHarness(t)
	_, token := h.register(t, "camille@facile.studio", "correct horse battery")

	recorder := h.patchMe(t, token, `{"password":"short","current_password":"correct horse battery"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err != nil {
		t.Fatalf("a refused change moved the password: %v", err)
	}
}

// TestChangingAnEmailKeepsThePasswordWorking is why the hand-written
// `UPDATE porte_identities SET subject = ?` is gone rather than merely unused.
//
// porte v0.2 keyed a local identity on the address, so an app that let anybody
// edit their profile had to chase the key with raw SQL against porte's own
// table. v0.3 keys on the account id, which does not move, so an address change
// touches no credential at all — and this is the test that fails if address
// keying is reintroduced anywhere, including by the adoption INSERT.
func TestChangingAnEmailKeepsThePasswordWorking(t *testing.T) {
	h := newHarness(t)
	_, token := h.register(t, "camille@facile.studio", "correct horse battery")

	recorder := h.patchMe(t, token, `{"email":"camille@example.org","current_password":"correct horse battery"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}

	if _, err := h.kit.Verify(t.Context(), "camille@example.org", "correct horse battery"); err != nil {
		t.Fatalf("the password stopped working at the new address: %v", err)
	}
	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err == nil {
		t.Fatal("the old address still signs in, so the account did not move")
	}

	var subjects []string
	if err := h.db.Raw(`SELECT subject FROM porte_identities WHERE provider = 'local'`).Scan(&subjects).Error; err != nil {
		t.Fatalf("read the identities: %v", err)
	}
	if len(subjects) != 1 {
		t.Fatalf("expected exactly one local identity, got %d: %v", len(subjects), subjects)
	}
	if strings.Contains(subjects[0], "@") {
		t.Fatalf("the local identity is keyed on an address again: %q", subjects[0])
	}
}

// Moving the address the account signs in with is confirmed too. It is the
// other half of the same takeover: a borrowed session that cannot change the
// password could otherwise still walk off with the login handle.
func TestChangingAnEmailRequiresTheCurrentPassword(t *testing.T) {
	h := newHarness(t)
	_, token := h.register(t, "camille@facile.studio", "correct horse battery")

	recorder := h.patchMe(t, token, `{"email":"camille@example.org"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}

	recorder = h.patchMe(t, token, `{"email":"camille@example.org","current_password":"not it at all"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401: %s", recorder.Code, recorder.Body.String())
	}

	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err != nil {
		t.Fatalf("the account moved on a refused request: %v", err)
	}
}

// TestTheReKeyReachesAnAccountRegisteredAfterTheAdoption is the case only the
// UPDATE in porteSchema can fix, and the reason repairing the adoption INSERT
// alone is not enough.
//
// adoptExistingPasswords is filtered on users.password_hash <> ”, and this
// app's CreateFromPassword never writes that column — porte holds the
// credential. So every account that signed up after the porte adoption sits
// outside that filter with nothing in users to re-adopt from, while carrying
// an identity keyed on its address. Fixing only the INSERT would rescue the
// accounts that predate the adoption and lock out everyone who has joined
// since. The identity is put back on the address, which is where porte v0.2
// left it, and the account must be able to sign in again afterwards.
func TestTheReKeyReachesAnAccountRegisteredAfterTheAdoption(t *testing.T) {
	h := newHarness(t)
	userID, _ := h.register(t, "camille@facile.studio", "correct horse battery")

	var carried int64
	if err := h.db.Raw(`SELECT count(*) FROM users WHERE id = ? AND coalesce(password_hash, '') <> ''`, userID).
		Scan(&carried).Error; err != nil {
		t.Fatalf("read the account: %v", err)
	}
	if carried != 0 {
		t.Fatal("users.password_hash is written again, which puts this account back inside the adoption INSERT and hides the case this test is about")
	}

	if err := h.db.Exec(
		`UPDATE porte_identities SET subject = 'camille@facile.studio' WHERE provider = 'local' AND user_id = ?`, userID,
	).Error; err != nil {
		t.Fatalf("rebuild the v0.2 shape: %v", err)
	}
	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err == nil {
		t.Fatal("the address-keyed identity still signs in, so this test proves nothing")
	}

	if err := schemas.AdoptPorte(h.db, ""); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := h.kit.Verify(t.Context(), "camille@facile.studio", "correct horse battery"); err != nil {
		t.Fatalf("the account cannot sign in after the migration: %v", err)
	}
}
