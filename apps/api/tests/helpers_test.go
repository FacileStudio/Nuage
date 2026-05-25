package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	activitymod "github.com/FacileStudio/Nuage/apps/api/modules/activity"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/files"
	"github.com/FacileStudio/Nuage/apps/api/modules/quota"
	"github.com/FacileStudio/Nuage/apps/api/modules/settings"
	"github.com/FacileStudio/Nuage/apps/api/modules/sharing"
	"github.com/FacileStudio/Nuage/apps/api/modules/sync"
	"github.com/FacileStudio/Nuage/apps/api/modules/trash"
	"github.com/FacileStudio/Nuage/apps/api/modules/users"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/go-chi/chi/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type testServer struct {
	router *chi.Mux
	db     *gorm.DB
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
		t.Skipf("skipping integration test: database not available: %v", err)
	}

	sqlDB, _ := db.DB()
	if err := sqlDB.Ping(); err != nil {
		t.Skipf("skipping integration test: database not reachable: %v", err)
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
		t.Skipf("skipping integration test: minio not available: %v", err)
	}
	_ = storageClient.EnsureBucket(context.Background())

	notifier := nook.NewNotifier(db)
	actLogger := activity.NewLogger(db)
	authService := auth.NewService(db)
	userService := users.NewService(db, t.TempDir())
	quotaService := quota.NewService(db)
	fileService := files.NewService(db, storageClient, notifier, actLogger, quotaService)
	trashService := trash.NewService(db, storageClient, actLogger, quotaService)
	syncService := sync.NewService(db)
	sharingService := sharing.NewService(db, notifier, actLogger)
	settingsService := settings.NewService(db)
	activityService := activitymod.NewService(db)

	appEnv := env.Config{SSOOnly: false}
	router := chi.NewRouter()
	router.Use(middleware.CORS([]string{"*"}))

	auth.RegisterRoutes(router, authService, appEnv)
	users.RegisterRoutes(router, userService, authService)
	files.RegisterRoutes(router, fileService, authService)
	trash.RegisterRoutes(router, trashService, authService)
	sharing.RegisterRoutes(router, sharingService, authService, storageClient)
	settings.RegisterRoutes(router, settingsService, authService)
	sync.RegisterRoutes(router, syncService, authService)
	quota.RegisterRoutes(router, quotaService, authService)
	activitymod.RegisterRoutes(router, activityService, authService)

	t.Cleanup(func() {
		cleanDB(db)
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	return &testServer{router: router, db: db}
}

func cleanDB(db *gorm.DB) {
	tables := []string{
		"activity_logs", "upload_chunks", "upload_sessions",
		"file_versions", "user_quotas", "shares",
		"files", "folders", "api_tokens", "sessions",
		"settings", "users",
	}
	for _, t := range tables {
		db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", t))
	}
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
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))

	if folderID != nil {
		writer.WriteField("folder_id", fmt.Sprintf("%d", *folderID))
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
