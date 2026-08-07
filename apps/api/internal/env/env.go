package env

import (
	"fmt"

	troncenv "github.com/FacileStudio/tronc/env"
)

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SuccessURL   string
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type Config struct {
	troncenv.Core
	StorageDir    string
	OIDC          *OIDCConfig
	SSOOnly       bool
	MinIO         MinIOConfig
	PresignSecret string
}

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
	}

	if cfg.MinIO, err = loadMinIO(); err != nil {
		return Config{}, err
	}

	if cfg.SSOOnly, err = troncenv.Bool("SSO_ONLY", false); err != nil {
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
		}
	}

	return cfg, nil
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
