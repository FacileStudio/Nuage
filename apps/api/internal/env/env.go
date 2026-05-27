package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
	DatabaseURL    string
	Port           string
	LogLevel       string
	StorageDir     string
	OIDC           *OIDCConfig
	SSOOnly        bool
	MinIO          MinIOConfig
	AllowedOrigins []string
	PresignSecret  string
}

func Load() (Config, error) {
	env := Config{
		DatabaseURL: valueOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nuage?sslmode=disable"),
		Port:        valueOrDefault("PORT", "4000"),
		LogLevel:    valueOrDefault("LOG_LEVEL", "info"),
		StorageDir:  valueOrDefault("STORAGE_DIR", "./data"),
		MinIO: MinIOConfig{
			Endpoint:  valueOrDefault("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: valueOrDefault("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: valueOrDefault("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    valueOrDefault("MINIO_BUCKET", "nuage"),
			UseSSL:    strings.ToLower(os.Getenv("MINIO_USE_SSL")) == "true",
		},
	}

	port, err := strconv.Atoi(env.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}
	if err := validateLogLevel(env.LogLevel); err != nil {
		return Config{}, err
	}

	env.SSOOnly = strings.ToLower(os.Getenv("SSO_ONLY")) == "true"

	if issuer := os.Getenv("OIDC_ISSUER"); issuer != "" {
		clientID := os.Getenv("OIDC_CLIENT_ID")
		clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
		redirectURL := os.Getenv("OIDC_REDIRECT_URL")
		successURL := os.Getenv("OIDC_SUCCESS_URL")
		if clientID == "" || clientSecret == "" || redirectURL == "" || successURL == "" {
			return Config{}, fmt.Errorf("OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, OIDC_REDIRECT_URL, and OIDC_SUCCESS_URL are required when OIDC_ISSUER is set")
		}
		env.OIDC = &OIDCConfig{
			Issuer:       issuer,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			SuccessURL:   successURL,
		}
	}

	env.PresignSecret = os.Getenv("PRESIGN_SECRET")

	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		env.AllowedOrigins = strings.Split(origins, ",")
		for i := range env.AllowedOrigins {
			env.AllowedOrigins[i] = strings.TrimSpace(env.AllowedOrigins[i])
		}
	} else {
		env.AllowedOrigins = []string{}
	}

	return env, nil
}

func valueOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func validateLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}
