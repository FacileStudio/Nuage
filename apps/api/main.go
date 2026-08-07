package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/database"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
	"github.com/FacileStudio/Nuage/apps/api/internal/logger"
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/internal/tombstone"
	activitymod "github.com/FacileStudio/Nuage/apps/api/modules/activity"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/docs"
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

	"github.com/FacileStudio/Journal/sdk/journal"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// startTombstonePruner keeps the sync deletion-marker table bounded by dropping
// markers older than the retention window clients are allowed to lag behind.
func startTombstonePruner(ctx context.Context, db *gorm.DB, appLogger *slog.Logger) {
	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			if removed, err := tombstone.Prune(ctx, db); err != nil {
				appLogger.Error("failed to prune deletion markers", slog.Any("error", err))
			} else if removed > 0 {
				appLogger.Info("pruned deletion markers", slog.Int64("count", removed))
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func run() error {
	appEnv, err := env.Load()
	appLogger := logger.New("info")
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		return err
	}
	appLogger = logger.New(appEnv.LogLevel)
	slog.SetDefault(appLogger)

	if appEnv.JournalURL != "" && appEnv.JournalToken != "" {
		journalClient := journal.New(journal.Config{URL: appEnv.JournalURL, Token: appEnv.JournalToken})
		defer journalClient.Close()
		appLogger = slog.New(journal.NewHandler(journalClient, appLogger.Handler()))
		slog.SetDefault(appLogger)
	}

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		return err
	}

	if err := schemas.Migrate(db); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return err
	}

	if err := os.MkdirAll(filepath.Join(appEnv.StorageDir, "avatars"), 0o755); err != nil {
		appLogger.Error("failed to prepare storage", slog.Any("error", err))
		return err
	}

	storageClient, err := storage.NewClient(storage.MinIOConfig{
		Endpoint:  appEnv.MinIO.Endpoint,
		AccessKey: appEnv.MinIO.AccessKey,
		SecretKey: appEnv.MinIO.SecretKey,
		Bucket:    appEnv.MinIO.Bucket,
		UseSSL:    appEnv.MinIO.UseSSL,
	})
	if err != nil {
		appLogger.Error("failed to create storage client", slog.Any("error", err))
		return err
	}

	if err := storageClient.EnsureBucket(context.Background()); err != nil {
		appLogger.Error("failed to ensure storage bucket", slog.Any("error", err))
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		return err
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	notifier := nook.NewNotifier(db)
	notifier.Start()
	defer notifier.Stop()
	actLogger := activity.NewLogger(db)

	authService := auth.NewService(db, notifier, appEnv.StorageDir, appLogger)
	userService := users.NewService(db, appEnv.StorageDir)
	quotaService := quota.NewService(db)
	if appEnv.PresignSecret == "" {
		err := errors.New("PRESIGN_SECRET is required: unauthenticated download links are signed with it")
		appLogger.Error("failed to load config", slog.Any("error", err))
		return err
	}
	presignSecret := []byte(appEnv.PresignSecret)
	fileService := files.NewService(db, storageClient, notifier, actLogger, quotaService, presignSecret)
	trashService := trash.NewService(db, storageClient, actLogger, quotaService)
	syncService := sync.NewService(db)
	sharingService := sharing.NewService(db, notifier, actLogger)
	settingsService := settings.NewService(db, notifier)
	searchService := search.NewService(db)
	spacesService := spaces.NewService(db)
	activityService := activitymod.NewService(db)

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.CORS(appEnv.AllowedOrigins))
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.RequestLogger(appLogger))
	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.RateLimitExcept(100, time.Minute, "/files/upload", "/webdav"))
	router.Use(middleware.RateLimitPaths(10, time.Minute, "/auth/login", "/auth/register"))

	router.Get("/health", func(w http.ResponseWriter, request *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/ready", func(w http.ResponseWriter, request *http.Request) {
		readinessContext, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(readinessContext); err != nil {
			httpjson.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": "database"})
			return
		}
		if err := storageClient.EnsureBucket(readinessContext); err != nil {
			httpjson.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "reason": "storage"})
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	docs.RegisterRoutes(router)

	avatarFS := http.StripPrefix("/avatars/", http.FileServer(http.Dir(filepath.Join(appEnv.StorageDir, "avatars"))))
	router.Get("/avatars/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		avatarFS.ServeHTTP(w, r)
	})

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
	nuagewebdav.RegisterRoutes(router, db, storageClient, authService, quotaService, appLogger)

	addr := ":" + appEnv.Port
	server := &http.Server{
		Addr:    addr,
		Handler: router,
		// Whole-request read and write deadlines are deliberately absent: a
		// multi-gigabyte upload or download legitimately outlives any fixed
		// budget, and exceeding it truncates the transfer after the response
		// header has already promised a Content-Length. Slow-header and idle
		// attacks are bounded below instead.
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fileService.StartSessionSweeper(shutdownSignal)
	startTombstonePruner(shutdownSignal, db, appLogger)

	appLogger.Info("server starting", slog.String("addr", addr))
	select {
	case err := <-serverErrCh:
		if !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server stopped", slog.Any("error", err))
			return err
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			return err
		}
		appLogger.Info("server stopped")
	}

	return nil
}
