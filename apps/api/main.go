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

	"github.com/FacileStudio/Nuage/apps/api/internal/database"
	documentation "github.com/FacileStudio/Nuage/apps/api/internal/documentation"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/httpjson"
	"github.com/FacileStudio/Nuage/apps/api/internal/logger"
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/modules/auth"
	"github.com/FacileStudio/Nuage/apps/api/modules/files"
	"github.com/FacileStudio/Nuage/apps/api/modules/users"
	"github.com/FacileStudio/Nuage/apps/api/schemas"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	appEnv, err := env.Load()
	appLogger := logger.New("info")
	if err != nil {
		appLogger.Error("failed to load config", slog.Any("error", err))
		return
	}
	appLogger = logger.New(appEnv.LogLevel)

	db, err := database.Open(appEnv.DatabaseURL)
	if err != nil {
		appLogger.Error("failed to open database", slog.Any("error", err))
		return
	}

	if err := schemas.Migrate(db); err != nil {
		appLogger.Error("failed to run migrations", slog.Any("error", err))
		return
	}

	if err := os.MkdirAll(filepath.Join(appEnv.StorageDir, "avatars"), 0o755); err != nil {
		appLogger.Error("failed to prepare storage", slog.Any("error", err))
		return
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
		return
	}

	if err := storageClient.EnsureBucket(context.Background()); err != nil {
		appLogger.Error("failed to ensure storage bucket", slog.Any("error", err))
		return
	}

	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("failed to access database handle", slog.Any("error", err))
		return
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			appLogger.Error("failed to close database", slog.Any("error", err))
		}
	}()

	authService := auth.NewService(db)
	userService := users.NewService(db, appEnv.StorageDir)
	fileService := files.NewService(db, storageClient)
	docs := documentation.Response{
		Modules: []documentation.Module{
			auth.Documentation,
			users.Documentation,
			files.Documentation,
		},
	}

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(middleware.CORS(appEnv.CORSAllowedOrigins))
	router.Use(middleware.RequestLogger(appLogger))
	router.Use(chimiddleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, request *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/ready", func(w http.ResponseWriter, request *http.Request) {
		readinessContext, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(readinessContext); err != nil {
			httpjson.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	router.Get("/docs", func(w http.ResponseWriter, request *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, docs)
	})

	auth.RegisterRoutes(router, authService, appEnv)
	users.RegisterRoutes(router, userService, authService)
	files.RegisterRoutes(router, fileService, authService)

	addr := ":" + appEnv.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	appLogger.Info("server starting", slog.String("addr", addr))
	select {
	case err := <-serverErrCh:
		if !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server stopped", slog.Any("error", err))
		}
	case <-shutdownSignal.Done():
		appLogger.Info("server shutting down")
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			appLogger.Error("server shutdown failed", slog.Any("error", err))
			return
		}
		appLogger.Info("server stopped")
	}
}
