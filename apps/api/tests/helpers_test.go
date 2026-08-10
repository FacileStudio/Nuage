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
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/presign"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	activitymod "github.com/FacileStudio/Nuage/apps/api/modules/activity"
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
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type testServer struct {
	router *chi.Mux
	db     *gorm.DB
}

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
	appEnv := env.Config{SSOOnly: false}
	sessions, passwords, kit, err := buildTestAuth(db, notifier, appEnv)
	if err != nil {
		t.Fatalf("build auth: %v", err)
	}
	authService := auth.NewService(db, sessions, passwords, slog.Default())
	userService := users.NewService(db, t.TempDir(), authService)
	quotaService := quota.NewService(db)
	presignSecret := presign.DeriveSecret("test-secret", "nuage-presign-v1")
	fileService := files.NewService(db, storageClient, notifier, actLogger, quotaService, presignSecret)
	trashService := trash.NewService(db, storageClient, actLogger, quotaService)
	syncService := sync.NewService(db)
	sharingService := sharing.NewService(db, notifier, actLogger)
	settingsService := settings.NewService(db, notifier)
	searchService := search.NewService(db)
	spacesService := spaces.NewService(db)
	activityService := activitymod.NewService(db)

	router := chi.NewRouter()
	router.Use(troncmiddleware.CORS(troncmiddleware.CORSConfig{AllowedOrigins: []string{"*"}}))

	sessions.Mount(router)
	kit.Mount(router)
	auth.RegisterRoutes(router, authService, appEnv)
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
	nuagewebdav.RegisterRoutes(router, db, storageClient, authService, quotaService, slog.Default())

	t.Cleanup(func() {
		cleanDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return &testServer{router: router, db: db}
}

func cleanDB(db *gorm.DB) {
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")
}

func registerUser(ts *testServer, email, password string) (string, string) {
	body := map[string]string{"email": email, "password": password}
	resp := doJSON(ts, "POST", "/auth/register", body, "")
	var result struct {
		UserID string `json:"user_id"`
		Token  string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.UserID, result.Token
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
func buildTestAuth(db *gorm.DB, notifier *nook.Notifier, appEnv env.Config) (*session.Manager, *local.Kit, *oidc.Kit, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, nil, err
	}
	store := portepg.New(sqlDB)
	users := auth.NewUserStore(db, notifier)
	cfg := appEnv.Porte()

	sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: slog.Default()})
	if err != nil {
		return nil, nil, nil, err
	}
	kit, err := oidc.New(context.Background(), cfg, oidc.Deps{
		Users: users, Identities: store.Identities(), Sessions: sessions,
		Codes: store.LoginCodes(), Logger: slog.Default(),
	})
	if err != nil {
		return nil, nil, nil, err
	}
	passwords, err := local.New(local.Config{AllowRegistration: true, MinPasswordLength: 8}, local.Deps{
		Users: users, Identities: store.Identities(), Sessions: sessions,
		Logger: slog.Default(), Count: users.CountUsers,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return sessions, passwords, kit, nil
}
