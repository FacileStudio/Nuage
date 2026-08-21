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
	"github.com/FacileStudio/Nuage/apps/api/internal/antenne"
	"github.com/FacileStudio/Nuage/apps/api/internal/database"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
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
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	troncmiddleware "github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

const (
	apiPrefix = "/api"
	// avatarRoutePrefix is where uploaded avatars are served from. schemas.AvatarFilePrefix
	// is the same location spelled as a stored value, and main_test.go asserts the two
	// still agree — a derived avatar pointing at a route that moved is a silent 404.
	avatarRoutePrefix = "/avatars/"
)

// buildAuth constructs porte: one session manager, shared by the OIDC kit and
// the local login, over the identity tables.
//
// One manager and not two: they would each keep their own idea of the clock
// and of whether the cookie is Secure, and porte refuses a kit whose config
// disagrees with its manager's for exactly that reason. Discovery runs here,
// so an unreachable or half-configured issuer fails at boot rather than on
// somebody's first login.
//
// The notifier rides into the UserStore rather than the auth service, because
// creating an account is the thing that emits user.created and the UserStore
// is now the only place an account is created.
//
// Nuage's floor has always been eight characters. porte defaults to
// twelve, and raising it here would reject a password this app accepted
// yesterday — a product decision, not a migration.
func buildAuth(ctx context.Context, db *gorm.DB, notifier *antenne.Notifier, appEnv env.Config, appLogger *slog.Logger) (*session.Manager, *local.Kit, *oidc.Kit, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, nil, err
	}
	store := portepg.New(sqlDB)
	users := auth.NewUserStore(db, notifier)
	cfg := appEnv.Porte()

	sessions, err := session.New(cfg, session.Deps{Sessions: store.Sessions(), Logger: appLogger})
	if err != nil {
		return nil, nil, nil, err
	}
	kit, err := oidc.New(ctx, cfg, oidc.Deps{
		Users:      users,
		Identities: store.Identities(),
		Sessions:   sessions,
		Codes:      store.LoginCodes(),
		Logger:     appLogger,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	passwords, err := local.New(local.Config{AllowRegistration: !appEnv.SSOOnly, MinPasswordLength: 8}, local.Deps{
		Users:      users,
		Identities: store.Identities(),
		Sessions:   sessions,
		Logger:     appLogger,
		Count:      users.CountUsers,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return sessions, passwords, kit, nil
}

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

// run wires the config, database, storage, auth and every module's routes
// into a server and blocks until it exits, returning the process exit code.
//
// Behind Traefik and Cloudflare, RemoteAddr is only the visitor if both
// are trusted: Traefik replaces the forwarded chain rather than extending
// it, so the visitor survives in Cf-Connecting-Ip alone. TRUSTED_PROXIES=
// private,cloudflare fills all three.
//
// Whole-request read and write deadlines are deliberately absent on the
// HTTP server: a multi-gigabyte upload or download legitimately outlives
// any fixed budget, and exceeding it truncates the transfer after the
// response header has already promised a Content-Length. Slow-header and
// idle attacks are bounded below instead.
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

	if err := schemas.MigrateWithIssuer(db, appEnv.IssuerForMigration()); err != nil {
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

	notifier := antenne.NewNotifier(db)
	notifier.Start()
	defer notifier.Stop()
	actLogger := activity.NewLogger(db)

	sessions, passwords, kit, err := buildAuth(context.Background(), db, notifier, appEnv, appLogger)
	if err != nil {
		appLogger.Error("failed to build authentication", slog.Any("error", err))
		return 1
	}
	authService := auth.NewService(db, sessions, passwords, appLogger)
	userService := users.NewService(db, appEnv.StorageDir, authService)
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
		TrustedProxies: appEnv.TrustedProxies,
		CDNProxies:     appEnv.CDNProxies,
		CDNHeader:      appEnv.CDNHeader,
		Logger:         appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins: appEnv.CORSAllowedOrigins,
		},
	})
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.RateLimitExcept(100, time.Minute, "/api/files/upload", "/webdav"))
	router.Use(middleware.RateLimitPaths(10, time.Minute, "/api/auth/login", "/api/auth/register"))

	health.Mount(router, health.DB(sqlDB), func(ctx context.Context) error {
		return storageClient.EnsureBucket(ctx)
	})

	avatarFS := http.StripPrefix(apiPrefix+avatarRoutePrefix, http.FileServer(http.Dir(filepath.Join(appEnv.StorageDir, "avatars"))))

	router.Route(apiPrefix, func(r chi.Router) {
		docs.RegisterRoutes(r)

		r.Get(avatarRoutePrefix+"*", func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
			avatarFS.ServeHTTP(w, request)
		})

		sessions.Mount(r)
		kit.Mount(r)
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
		Addr:              addr,
		Handler:           router,
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
