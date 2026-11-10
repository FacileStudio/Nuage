package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/ideas"
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/presign"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	activitymod "github.com/FacileStudio/Nuage/apps/api/modules/activity"
	"github.com/FacileStudio/Nuage/apps/api/modules/appfolders"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/files"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/modules/search"
	"github.com/FacileStudio/Nuage/apps/api/modules/settings"
	"github.com/FacileStudio/Nuage/apps/api/modules/sharing"
	"github.com/FacileStudio/Nuage/apps/api/modules/spaces"
	"github.com/FacileStudio/Nuage/apps/api/modules/sync"
	"github.com/FacileStudio/Nuage/apps/api/modules/trash"
	"github.com/FacileStudio/Nuage/apps/api/modules/users"
	nuagewebdav "github.com/FacileStudio/Nuage/apps/api/modules/webdav"
	"github.com/FacileStudio/Nuage/apps/api/schemas"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	troncbulle "github.com/FacileStudio/tronc/bulle"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type testServer struct {
	router     *chi.Mux
	db         *gorm.DB
	sessions   *session.Manager
	plans      *entitlements.Plans
	store      *storage.Client
	appFolders *appfolders.Service
	bulle      *fakeBulle
}

// testIssuer is the OIDC issuer the plan entitlements are built for in tests.
const testIssuer = "https://sso.test"

// skipOrFail lets a developer without Postgres and MinIO run the suite, while
// making the same missing infrastructure a hard failure in CI so a green run
// can never mean "nothing was tested".
func skipOrFail(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("CI") != "" {
		t.Fatalf("integration test infrastructure required in CI: "+format, args...)
	}
	t.Skipf("skipping integration test: "+format, args...)
}

func setupTestServer(t *testing.T) *testServer {
	t.Helper()
	return setupTestServerWith(t, env.Config{}, nil)
}

// setupTestServerWith builds the server from appEnv. A non-nil verifier stands
// in for the provider's JWKS on the bearer JWT path, and the OIDC kit is then
// left unmounted because its discovery would need a live issuer.
func setupTestServerWith(t *testing.T, appEnv env.Config, verifier session.JWTVerifier) *testServer {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://nuage:nuage-internal-db@localhost:5432/nuage_test?sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Discard,
	})
	if err != nil {
		skipOrFail(t, "database not available: %v", err)
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Ping(); err != nil {
		skipOrFail(t, "database not reachable: %v", err)
	}

	cleanDB(db)
	if err := schemas.Migrate(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	if err := appregistry.Sync(context.Background(), db, appEnv.Apps); err != nil {
		t.Fatalf("register apps: %v", err)
	}

	minioEndpoint := os.Getenv("TEST_MINIO_ENDPOINT")
	if minioEndpoint == "" {
		minioEndpoint = "localhost:9000"
	}
	minioAccess := os.Getenv("TEST_MINIO_ACCESS_KEY")
	if minioAccess == "" {
		minioAccess = "nuage-minio"
	}
	minioSecret := os.Getenv("TEST_MINIO_SECRET_KEY")
	if minioSecret == "" {
		minioSecret = "nuage-internal-storage"
	}

	storageClient, err := storage.NewClient(storage.MinIOConfig{
		Endpoint:  minioEndpoint,
		AccessKey: minioAccess,
		SecretKey: minioSecret,
		Bucket:    "nuage-test",
		UseSSL:    false,
	})
	if err != nil {
		skipOrFail(t, "minio not available: %v", err)
	}
	_ = storageClient.EnsureBucket(context.Background())

	notifier := nook.NewNotifier(db)
	actLogger := activity.NewLogger(db)
	sessions, kit, err := buildTestAuth(db, notifier, appEnv, verifier)
	if err != nil {
		t.Fatalf("build auth: %v", err)
	}
	authService := auth.NewService(db, sessions, slog.Default()).WithFullAccessActors(appEnv.FullAccessActors)
	userService := users.NewService(db, t.TempDir(), authService)
	plans := entitlements.New(db, testIssuer, slog.Default())
	quotaService := quota.NewService(db).WithPlans(plans)
	presignSecret := presign.DeriveSecret("test-secret", "nuage-presign-v1")
	fileService := files.NewService(db, storageClient, notifier, actLogger, quotaService, presignSecret)
	trashService := trash.NewService(db, storageClient, actLogger, quotaService)
	syncService := sync.NewService(db)
	sharingService := sharing.NewService(db, notifier, actLogger)
	settingsService := settings.NewService(db, notifier)
	searchService := search.NewService(db)
	registreURL := ""
	if appEnv.OIDC != nil {
		registreURL = appEnv.OIDC.Issuer + "/spaces"
	}
	spacesService := spaces.NewService(db, storageClient, registreURL)
	activityService := activitymod.NewService(db)
	appFolderService := appfolders.NewService(db, storageClient, quotaService, slog.Default())

	router := chi.NewRouter()
	router.Use(troncmiddleware.CORS(troncmiddleware.CORSConfig{AllowedOrigins: []string{"*"}}))

	if err := auth.RegisterRoutes(router, sessions, kit); err != nil {
		t.Fatalf("mount auth: %v", err)
	}
	users.RegisterRoutes(router, userService, authService)
	files.RegisterRoutes(router, fileService, authService)
	trash.RegisterRoutes(router, trashService, authService)
	sharing.RegisterRoutes(router, sharingService, authService, storageClient)
	settings.RegisterRoutes(router, settingsService, authService)
	sync.RegisterRoutes(router, syncService, authService)
	quota.RegisterRoutes(router, quotaService, authService)
	search.RegisterRoutes(router, searchService, authService)
	spaces.RegisterRoutes(router, spacesService, authService)
	activitymod.RegisterRoutes(router, activityService, authService)
	appfolders.RegisterRoutes(router, appFolderService, authService)
	bulle := newFakeBulle(t)
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		err := ideas.New(db, plans, quotaService).Mount(r, troncbulle.Options{
			URL:         bulle.server.URL,
			TokenSource: func(context.Context) (string, error) { return "tok", nil },
			UpgradeURL:  testIssuer + "/spaces",
		})
		if err != nil {
			t.Fatalf("mount ideas: %v", err)
		}
	})
	nuagewebdav.RegisterRoutes(router, db, storageClient, authService, quotaService, slog.Default())

	t.Cleanup(func() {
		cleanDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return &testServer{router: router, db: db, sessions: sessions, plans: plans, store: storageClient, appFolders: appFolderService, bulle: bulle}
}

func cleanDB(db *gorm.DB) {
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")
}

// userStore is the app's UserStore over db, as buildAuth makes it.
func userStore(db *gorm.DB) *auth.UserStore {
	store, err := auth.NewUserStore(context.Background(), db, nil)
	if err != nil {
		panic(err)
	}
	return store
}

// registerUser creates an account through the app's UserStore, as a Registre
// sign-in does, gives it the local identity the retired password sign-up wrote,
// and issues a session. The password is ignored: no password sign-in exists
// any more. The first account created is the instance admin, as in production.
func registerUser(ts *testServer, email, _ string) (string, string) {
	ctx := context.Background()
	claims := porte.Claims{Provider: testIssuer, Subject: "sub-" + email, Email: email, EmailVerified: true}
	id, err := userStore(ts.db).UpsertFromOIDC(ctx, claims)
	if err != nil {
		panic(err)
	}
	err = ts.db.Exec(`INSERT INTO porte_identities (user_id, provider, subject) VALUES (?, 'local', ?)
		ON CONFLICT DO NOTHING`, id, strconv.FormatInt(id, 10)).Error
	if err != nil {
		panic(err)
	}
	token, _, err := ts.sessions.Issue(ctx, id, "")
	if err != nil {
		panic(err)
	}
	return strconv.FormatInt(id, 10), token
}

func doJSON(ts *testServer, method, path string, body any, token string) *http.Response {
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w.Result()
}

func doGet(ts *testServer, path, token string) *http.Response {
	return doJSON(ts, "GET", path, nil, token)
}

func doDelete(ts *testServer, path, token string) *http.Response {
	return doJSON(ts, "DELETE", path, nil, token)
}

func uploadFile(ts *testServer, token, filename, content string, folderID *int64) *http.Response {
	fields := map[string]string{}
	if folderID != nil {
		fields["folder_id"] = fmt.Sprintf("%d", *folderID)
	}
	return uploadFileWithFields(ts, token, filename, content, fields)
}

func uploadFileWithFields(ts *testServer, token, filename, content string, fields map[string]string) *http.Response {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))

	for key, value := range fields {
		writer.WriteField(key, value)
	}

	writer.Close()

	req := httptest.NewRequest("POST", "/files", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w.Result()
}

func reuploadFile(ts *testServer, token string, fileID int64, content string) *http.Response {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, _ := writer.CreateFormFile("file", "updated.txt")
	part.Write([]byte(content))
	writer.Close()

	req := httptest.NewRequest("POST", fmt.Sprintf("/files/%d/reupload", fileID), &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w.Result()
}

func parseJSON(resp *http.Response, dest any) {
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(dest)
}

// buildTestAuth mirrors main.go's buildAuth over the test database. OIDC is
// unconfigured here, so oidc.New returns a kit that serves /auth/config and
// authenticates sessions and nothing else — which is what these tests exercise.
// With a verifier the manager also takes bearer JWTs and no kit is built.
func buildTestAuth(
	db *gorm.DB, notifier *nook.Notifier, appEnv env.Config, verifier session.JWTVerifier,
) (*session.Manager, *oidc.Kit, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	store := portepg.New(sqlDB)
	users, err := auth.NewUserStore(context.Background(), db, notifier)
	if err != nil {
		return nil, nil, err
	}
	cfg := appEnv.Porte()

	sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: slog.Default()})
	if err != nil {
		return nil, nil, err
	}
	if verifier != nil {
		return sessions.WithJWT(verifier), nil, nil
	}
	kit, err := oidc.New(context.Background(), cfg, oidc.Deps{
		Users: users, Identities: store.Identities(), Sessions: sessions,
		Codes: store.LoginCodes(), Logger: slog.Default(),
	})
	if err != nil {
		return nil, nil, err
	}
	return sessions, kit, nil
}
