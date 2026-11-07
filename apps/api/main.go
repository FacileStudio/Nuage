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
	"strings"
	"syscall"
	"time"

	"github.com/FacileStudio/Nuage/apps/api/internal/activity"
	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/Nuage/apps/api/internal/database"
	"github.com/FacileStudio/Nuage/apps/api/internal/entitlements"
	"github.com/FacileStudio/Nuage/apps/api/internal/env"
	"github.com/FacileStudio/Nuage/apps/api/internal/erasure"
	"github.com/FacileStudio/Nuage/apps/api/internal/middleware"
	"github.com/FacileStudio/Nuage/apps/api/internal/nook"
	"github.com/FacileStudio/Nuage/apps/api/internal/storage"
	"github.com/FacileStudio/Nuage/apps/api/internal/tombstone"
	"github.com/FacileStudio/Nuage/apps/api/internal/version"
	activitymod "github.com/FacileStudio/Nuage/apps/api/modules/activity"
	"github.com/FacileStudio/Nuage/apps/api/modules/appfolders"
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
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpjson"
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

// buildAuth constructs porte: one session manager, shared by the OIDC kit,
// over the identity tables.
//
// One manager and not two: porte refuses a kit whose config disagrees with its
// manager's. Discovery runs here, so an unreachable or half-configured issuer
// fails at boot rather than on somebody's first login. Sign-in goes through
// Registre only: with an issuer configured porte mounts no password route.
//
// The notifier rides into the UserStore rather than the auth service, because
// creating an account is the thing that emits user.created and the UserStore
// is the only place an account is created.
func buildAuth(ctx context.Context, db *gorm.DB, notifier *nook.Notifier, appEnv env.Config, appLogger *slog.Logger) (*session.Manager, *oidc.Kit, *portepg.Store, error) {
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
	return sessions, kit, store, nil
}

// newDirectory builds the Registre space sync when OIDC is configured, using the app's own
// client credentials, and returns nils otherwise: without Registre there is nothing to sync
// and no nudge route to mount. Every snapshot updates the spaces first, then the plan
// entitlements; both run even when the other fails, and either failure retries the snapshot.
// The snapshot that first makes plans apply also logs who is now over their ceiling.
func newDirectory(db *gorm.DB, appEnv env.Config, plans *entitlements.Plans, quotaService *quota.Service, appLogger *slog.Logger) (*spaces.Registre, *directory.Syncer) {
	if appEnv.OIDC == nil {
		return nil, nil
	}
	registre := spaces.NewRegistre(db, appEnv.OIDC.Issuer, appLogger)
	syncer := &directory.Syncer{
		Client: directory.Client{
			Issuer:       appEnv.OIDC.Issuer,
			ClientID:     appEnv.OIDC.ClientID,
			ClientSecret: appEnv.OIDC.ClientSecret,
		},
		Apply: func(ctx context.Context, snapshot directory.Snapshot) error {
			was := plans.Enforcing()
			err := errors.Join(registre.Apply(ctx, snapshot), plans.Apply(ctx, snapshot))
			if !was && plans.Enforcing() {
				go quotaService.ReportOverPlan(context.Background(), appLogger)
			}
			return err
		},
		Logger: appLogger,
	}
	return registre, syncer
}

// newDeletions builds the GDPR deletion receiver when OIDC is configured, with
// the same client credentials as the snapshot sync, and returns nil otherwise.
// Nuage runs as one replica, so the one Run it starts is the only one.
func newDeletions(cfg erasure.Config, appEnv env.Config) *erasure.Deletions {
	if appEnv.OIDC == nil {
		return nil
	}
	cfg.Client = directory.Client{
		Issuer:       appEnv.OIDC.Issuer,
		ClientID:     appEnv.OIDC.ClientID,
		ClientSecret: appEnv.OIDC.ClientSecret,
	}
	cfg.StorageDir = appEnv.StorageDir
	return erasure.New(cfg)
}

// registreSpacesURL is Registre's spaces page, where shared spaces are managed once OIDC is
// configured, and empty when Nuage manages them itself.
func registreSpacesURL(appEnv env.Config) string {
	if appEnv.OIDC == nil {
		return ""
	}
	return strings.TrimRight(appEnv.OIDC.Issuer, "/") + "/spaces"
}

func main() {
	if healthcheck.Handle(os.Args) {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == appMigrationCommand {
		os.Exit(runAppMigration(os.Args[2:], os.Stdout))
	}

	os.Exit(run())
}

// startMaintenance runs the hourly housekeeping: it drops sync deletion markers
// older than the window clients are allowed to lag behind, and deletes for good
// the trashed items that outlived the trash retention.
func startMaintenance(ctx context.Context, db *gorm.DB, trashService *trash.Service, appLogger *slog.Logger) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			runMaintenance(ctx, db, trashService, appLogger)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func runMaintenance(ctx context.Context, db *gorm.DB, trashService *trash.Service, appLogger *slog.Logger) {
	if removed, err := tombstone.Prune(ctx, db); err != nil {
		appLogger.Error("failed to prune deletion markers", slog.Any("error", err))
	} else if removed > 0 {
		appLogger.Info("pruned deletion markers", slog.Int64("count", removed))
	}
	if purged, err := trashService.PurgeExpired(ctx, time.Now()); err != nil {
		appLogger.Error("failed to purge expired trash", slog.Any("error", err))
	} else if purged > 0 {
		appLogger.Info("purged expired trash", slog.Int("count", purged))
	}
}

// catalogOrigin is where muse's PricingPlans reads the plan catalogue from.
const catalogOrigin = "https://api-ardoise.facile.studio"

// securityConfig extends tronc's default security headers for the built client:
// the SvelteKit bootstrap and app.html each carry an inline script, and
// script-src 'self' would block both once the policy is enforced. The landing
// page's pricing section fetches the plan catalogue cross-origin, so its origin
// joins connect-src. The policy stays report-only until a release shows no
// violation in the console.
func securityConfig(clientDir string, appLogger *slog.Logger) troncmiddleware.SecurityConfig {
	sources := map[string][]string{"connect-src": {catalogOrigin}}
	if spa.Available(clientDir) {
		hashes, err := spa.InlineScriptHashes(clientDir)
		if err != nil {
			appLogger.Error("failed to hash the client's inline scripts", slog.Any("error", err))
		}
		sources["script-src"] = hashes
	}
	return troncmiddleware.SecurityConfig{CSPSources: sources}
}

// registerVersionRoute answers /version and /api/version with the version of
// the running binary.
//
// Both paths, for the reason health.Mount registers both: the public edge may
// forward only /api/*, and a probe that answers on one and 404s on the other is
// how a deploy ends up unverifiable.
func registerVersionRoute(router chi.Router) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"version": version.String()})
	}
	router.Get("/version", handler)
	router.Get(apiPrefix+"/version", handler)
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

	if err := appregistry.Sync(context.Background(), db, appEnv.Apps); err != nil {
		appLogger.Error("failed to register the configured apps", slog.Any("error", err))
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

	sessions, kit, porteStore, err := buildAuth(context.Background(), db, notifier, appEnv, appLogger)
	if err != nil {
		appLogger.Error("failed to build authentication", slog.Any("error", err))
		return 1
	}
	authService := auth.NewService(db, sessions, appLogger).WithFullAccessActors(appEnv.FullAccessActors)
	userService := users.NewService(db, appEnv.StorageDir, authService)
	plans := newPlans(db, appEnv, appLogger)
	quotaService := quota.NewService(db).WithPlans(plans)
	go reportStorage(quotaService, appLogger)
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
	spacesService := spaces.NewService(db, storageClient, registreSpacesURL(appEnv))
	registre, syncer := newDirectory(db, appEnv, plans, quotaService, appLogger)
	appFolderService := appfolders.NewService(db, storageClient, quotaService, appLogger)
	deletions := newDeletions(erasure.Config{
		ORM: db, Erase: porteStore.EraseIdentity, Files: trashService, Apps: appFolderService, Logger: appLogger,
	}, appEnv)
	if err := spacesService.MigrateBuckets(context.Background()); err != nil {
		appLogger.Error("failed to move space files to their own buckets; serving with the files still in the shared bucket", slog.Any("error", err))
	}
	activityService := activitymod.NewService(db)

	clientDir := spa.DirFromEnv()
	router := httpx.NewRouter(httpx.Config{
		TrustedProxies: appEnv.TrustedProxies,
		CDNProxies:     appEnv.CDNProxies,
		CDNHeader:      appEnv.CDNHeader,
		Logger:         appLogger,
		CORS: troncmiddleware.CORSConfig{
			AllowedOrigins: appEnv.CORSAllowedOrigins,
		},
		Security: securityConfig(clientDir, appLogger),
	})
	router.Use(middleware.RateLimitAPI(600, time.Minute, apiPrefix+"/", middleware.IsTransferPath))

	health.Mount(router, health.DB(sqlDB), func(ctx context.Context) error {
		return storageClient.EnsureBucket(ctx)
	})
	registerVersionRoute(router)
	if syncer != nil {
		router.Method(http.MethodPost, directory.SyncPath, syncer.Handler())
	}
	if deletions != nil {
		router.Method(http.MethodPost, directory.DeletionsPath, deletions.Handler())
	}

	avatarFS := http.StripPrefix(apiPrefix+avatarRoutePrefix, http.FileServer(http.Dir(filepath.Join(appEnv.StorageDir, "avatars"))))

	router.Route(apiPrefix, func(r chi.Router) {
		r.Get(avatarRoutePrefix+"*", func(w http.ResponseWriter, request *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
			avatarFS.ServeHTTP(w, request)
		})

		auth.RegisterRoutes(r, sessions, kit)
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
		appfolders.RegisterRoutes(r, appFolderService, authService)
	})

	nuagewebdav.RegisterRoutes(router, db, storageClient, authService, quotaService, appLogger)
	docs.Mount(router)

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
	startMaintenance(shutdownSignal, db, trashService, appLogger)
	if registre != nil {
		go registre.Run(shutdownSignal, syncer)
	}
	if deletions != nil {
		go deletions.Run(shutdownSignal)
	}

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
