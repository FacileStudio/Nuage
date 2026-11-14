package env

import (
	"github.com/FacileStudio/Nuage/apps/api/internal/appregistry"
	"github.com/FacileStudio/porte"

	"fmt"

	troncenv "github.com/FacileStudio/tronc/env"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// OIDCConfig holds the settings used to federate with the identity provider.
type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SuccessURL   string

	// CLIAudience is the client id the suite's CLI holds its device-grant
	// token under, and setting it is what mounts porte's device exchange so
	// one `facile login` reaches Nuage too. It is deliberately not
	// OIDC_CLIENT_ID: the token being traded was minted for the CLI, not for
	// Nuage. It is not the machine audience either, which names this app and
	// arms the bearer path for service accounts.
	CLIAudience string

	// MachineAudience is Nuage's own Registre client id. Setting it mounts
	// offline verification of the delegated bearer JWTs other suite services
	// mint for Nuage on a signed-in user's behalf.
	MachineAudience string

	// ScopePrefix gates those JWTs by method: GET, HEAD and OPTIONS need
	// nuage.read or nuage.write, anything else needs nuage.write. It
	// requires MachineAudience.
	ScopePrefix string
}

// MinIOConfig holds the S3-compatible storage connection settings.
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// Config is the application configuration loaded from environment variables.
type Config struct {
	troncenv.Core
	StorageDir    string
	OIDC          *OIDCConfig
	MinIO         MinIOConfig
	PresignSecret string

	// Apps are the suite apps that get a private app folder per user, from
	// NUAGE_APPS. An app acts only through a delegated Registre JWT whose
	// act.sub is its client id, so they need OIDC_MACHINE_AUDIENCE.
	Apps []appregistry.App

	// FullAccessActors are the Registre client ids whose delegated JWTs act
	// with the user's full access, from NUAGE_FULL_ACCESS_ACTORS (Bulle in
	// production). Any other actor that is not a registered app is refused.
	FullAccessActors []string

	// BulleURL is where the Ideas routes reach Bulle, from BULLE_URL. Empty
	// leaves them unmounted, as does a Nuage without OIDC.
	BulleURL string

	// MCP turns on the remote MCP server.
	MCP MCP
}

// MCP configures the remote MCP server. Resource is the canonical URI
// clients connect to, https://nuage.facile.studio/mcp in production, and the
// RFC 8707 resource Registre binds its tokens to, from MCP_RESOURCE. It needs
// OIDC. StateKey signs the confirmation state handed to clients; it is
// derived from PRESIGN_SECRET, so no new secret is needed.
type MCP struct {
	Resource string
	StateKey []byte
}

// Load reads the configuration from the environment, failing when a required
// value (MinIO endpoint and credentials, PRESIGN_SECRET) is absent or invalid.
func Load() (Config, error) {
	core, err := troncenv.LoadCore()
	if err != nil {
		return Config{}, err
	}
	if core.Port < 1 || core.Port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}

	cfg := Config{
		Core:          core,
		StorageDir:    troncenv.String("STORAGE_DIR", "./data"),
		PresignSecret: troncenv.String("PRESIGN_SECRET", ""),
		BulleURL:      troncenv.String("BULLE_URL", ""),
	}
	if cfg.PresignSecret != "" {
		cfg.MCP = MCP{Resource: troncenv.String("MCP_RESOURCE", ""), StateKey: mcpkit.StateKey("nuage", cfg.PresignSecret)}
	}

	if cfg.MinIO, err = loadMinIO(); err != nil {
		return Config{}, err
	}

	if issuer := troncenv.String("OIDC_ISSUER", ""); issuer != "" {
		clientID := troncenv.String("OIDC_CLIENT_ID", "")
		clientSecret := troncenv.String("OIDC_CLIENT_SECRET", "")
		redirectURL := troncenv.String("OIDC_REDIRECT_URL", "")
		successURL := troncenv.String("OIDC_SUCCESS_URL", "")
		if clientID == "" || clientSecret == "" || redirectURL == "" || successURL == "" {
			return Config{}, fmt.Errorf("OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, OIDC_REDIRECT_URL, and OIDC_SUCCESS_URL are required when OIDC_ISSUER is set")
		}
		cfg.OIDC = &OIDCConfig{
			Issuer:       issuer,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			SuccessURL:   successURL,
			CLIAudience:  troncenv.String("OIDC_CLI_AUDIENCE", ""),

			MachineAudience: troncenv.String("OIDC_MACHINE_AUDIENCE", ""),
			ScopePrefix:     troncenv.String("OIDC_SCOPE_PREFIX", ""),
		}
	}

	if cfg.Apps, cfg.FullAccessActors, err = loadApps(cfg.OIDC); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// loadApps reads NUAGE_APPS and NUAGE_FULL_ACCESS_ACTORS. Registered apps
// prove who they are with a delegated JWT, so declaring one without the
// machine audience that verifies those tokens is refused rather than left
// unable to act.
func loadApps(oidc *OIDCConfig) ([]appregistry.App, []string, error) {
	ownClientID := ""
	if oidc != nil {
		ownClientID = oidc.ClientID
	}
	apps, err := appregistry.Parse(troncenv.String("NUAGE_APPS", ""), ownClientID)
	if err != nil {
		return nil, nil, err
	}
	if len(apps) > 0 && (oidc == nil || oidc.MachineAudience == "") {
		return nil, nil, fmt.Errorf("NUAGE_APPS requires OIDC_ISSUER and OIDC_MACHINE_AUDIENCE: apps act through delegated Registre tokens")
	}
	actors, err := appregistry.ParseActors(troncenv.String("NUAGE_FULL_ACCESS_ACTORS", ""), apps)
	return apps, actors, err
}

// loadMinIO requires the endpoint and the credentials. Every file the app
// stores goes through MinIO, and the old defaults were localhost:9000 with
// minioadmin/minioadmin: a deployment that forgot them started, answered
// health checks, and failed on the first upload.
func loadMinIO() (MinIOConfig, error) {
	endpoint, err := troncenv.Required("MINIO_ENDPOINT")
	if err != nil {
		return MinIOConfig{}, err
	}
	accessKey, err := troncenv.Required("MINIO_ACCESS_KEY")
	if err != nil {
		return MinIOConfig{}, err
	}
	secretKey, err := troncenv.Required("MINIO_SECRET_KEY")
	if err != nil {
		return MinIOConfig{}, err
	}
	useSSL, err := troncenv.Bool("MINIO_USE_SSL", false)
	if err != nil {
		return MinIOConfig{}, err
	}

	return MinIOConfig{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    troncenv.String("MINIO_BUCKET", "nuage"),
		UseSSL:    useSSL,
	}, nil
}

// Porte is the one configuration porte's session manager and OIDC kit are both
// built from. They share it because porte refuses at boot a kit whose config
// disagrees with its manager's — a mismatch would otherwise change silently
// whether the session cookie is Secure.
func (c Config) Porte() porte.Config {
	cfg := porte.Config{AcceptLegacyCookie: true}
	if c.OIDC == nil {
		return cfg
	}
	cfg.Issuer = c.OIDC.Issuer
	cfg.ClientID = c.OIDC.ClientID
	cfg.ClientSecret = c.OIDC.ClientSecret
	cfg.RedirectURL = c.OIDC.RedirectURL
	cfg.SuccessURL = c.OIDC.SuccessURL
	cfg.CLIAudience = c.OIDC.CLIAudience
	cfg.MachineAudience = c.OIDC.MachineAudience
	cfg.ScopePrefix = c.OIDC.ScopePrefix
	return cfg
}

// IssuerForMigration is the issuer the identity backfill keys on, or empty
// when SSO is not configured. It exists so the migration cannot be handed a
// placeholder: an identity row written under the wrong provider matches
// nothing and degrades to the email fallback in silence.
func (c Config) IssuerForMigration() string {
	if c.OIDC == nil {
		return ""
	}
	return c.OIDC.Issuer
}
