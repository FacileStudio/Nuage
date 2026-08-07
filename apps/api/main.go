package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/database"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
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
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

func main() {
	if healthcheck.Handle(os.Args) {
		return
	}

	os.Exit(run())
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

func run() int {
	appEnv, err := env.Load()
	appLogger := logger.New(logger.Config{})
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		return 1
	}
	var journalClient *journal.Client
	appLogger = logger.New(logger.Config{
		Level: appEnv.LogLevel,
		Wrap: func(handler slog.Handler) slog.Handler {
			if appEnv.JournalURL == "" || appEnv.JournalToken == "" {
				return handler
			}
			journalClient = journal.New(journal.Config{URL: appEnv.JournalURL, Token: appEnv.JournalToken})
			return journal.NewHandler(journalClient, handler)
		},
	})
	if journalClient != nil {
		defer journalClient.Close()
	}
	slog.SetDefault(appLogger)

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		return 1
	}

	if err := schemas.Migrate(db); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return 1
	}

	if err := os.MkdirAll(filepath.Join(appEnv.StorageDir, "avatars"), 0o755); err != nil {
		appLogger.Error("failed to prepare storage", slog.Any("error", err))
		return 1
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
		return 1
	}

	if err := storageClient.EnsureBucket(context.Background()); err != nil {
		appLogger.Error("failed to ensure storage bucket", slog.Any("error", err))
		return 1
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		return 1
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
		appLogger.Error("failed to load config", slog.Any("error", errors.New("PRESIGN_SECRET is required: unauthenticated download links are signed with it")))
		return 1
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

	router := httpx.NewRouter(httpx.Config{
		Logger: appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins: appEnv.CORSAllowedOrigins,
		},
	})
	router.Use(middleware.RealIP)
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.RateLimitExcept(100, time.Minute, "/api/files/upload", "/webdav"))
	router.Use(middleware.RateLimitPaths(10, time.Minute, "/api/auth/login", "/api/auth/register"))

	health.Mount(router, health.DB(sqlDB), func(ctx context.Context) error {
		return storageClient.EnsureBucket(ctx)
	})

	avatarFS := http.StripPrefix("/api/avatars/", http.FileServer(http.Dir(filepath.Join(appEnv.StorageDir, "avatars"))))

	router.Route("/api", func(r chi.Router) {
		docs.RegisterRoutes(r)

		r.Get("/avatars/*", func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
			avatarFS.ServeHTTP(w, request)
		})

		auth.RegisterRoutes(r, authService, appEnv)
		users.RegisterRoutes(r, userService, authService)
		files.RegisterRoutes(r, fileService, authService)
		trash.RegisterRoutes(r, trashService, authService)
		sharing.RegisterRoutes(r, sharingService, authService, storageClient)
		settings.RegisterRoutes(r, settingsService, authService)
		sync.RegisterRoutes(r, syncService, authService)
		quota.RegisterRoutes(r, quotaService, authService)
		search.RegisterRoutes(r, searchService, authService)
		spaces.RegisterRoutes(r, spacesService, authService)
		activitymod.RegisterRoutes(r, activityService, authService)
	})

	nuagewebdav.RegisterRoutes(router, db, storageClient, authService, quotaService, appLogger)

	clientDir := spa.DirFromEnv()
	if spa.Available(clientDir) {
		router.Handle("/*", spa.Handler(spa.Config{Dir: clientDir}))
		appLogger.Info("serving client", slog.String("dir", clientDir))
	}

	addr := ":" + strconv.Itoa(appEnv.Port)
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
			return 1
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			return 1
		}
		appLogger.Info("server stopped")
	}

	return 0
}
