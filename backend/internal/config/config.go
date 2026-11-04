package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is everything the server reads from the environment.
type Config struct {
	Env             string
	Addr            string
	DatabaseURL     string
	JWTSecret       []byte
	AccessTTL       time.Duration
	RefreshTTL      time.Duration
	StorageDriver   string // local | s3
	StorageDir      string
	S3Bucket        string
	S3Region        string
	S3Endpoint      string
	MaxUploadBytes  int64
	AllowedOrigins  []string
	CookieSecure    bool
	BootstrapEmail  string
	BootstrapPass   string
	TrustedProxyHdr bool
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// Load validates the environment, failing fast on unsafe production settings.
func Load() (*Config, error) {
	c := &Config{
		Env:            env("APP_ENV", "development"),
		Addr:           env("ADDR", ":8080"),
		DatabaseURL:    env("DATABASE_URL", ""),
		JWTSecret:      []byte(env("JWT_SECRET", "")),
		AccessTTL:      time.Duration(atoi(env("ACCESS_TTL_MINUTES", "15"), 15)) * time.Minute,
		RefreshTTL:     time.Duration(atoi(env("REFRESH_TTL_HOURS", "168"), 168)) * time.Hour,
		StorageDriver:  env("STORAGE_DRIVER", "local"),
		StorageDir:     env("STORAGE_DIR", "/data/scripts"),
		S3Bucket:       env("S3_BUCKET", ""),
		S3Region:       env("S3_REGION", "eu-west-1"),
		S3Endpoint:     env("S3_ENDPOINT", ""),
		MaxUploadBytes: int64(atoi(env("MAX_UPLOAD_MB", "25"), 25)) << 20,
		CookieSecure:   env("COOKIE_SECURE", "true") == "true",
		BootstrapEmail: env("BOOTSTRAP_ADMIN_EMAIL", ""),
		BootstrapPass:  env("BOOTSTRAP_ADMIN_PASSWORD", ""),
	}
	for _, o := range strings.Split(env("ALLOWED_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, o)
		}
	}
	if c.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	if len(c.JWTSecret) < 32 {
		if c.Env == "production" {
			return nil, errors.New("JWT_SECRET must be at least 32 bytes in production")
		}
		c.JWTSecret = []byte("dev-only-insecure-secret-key-change-me-32b")
	}
	if c.StorageDriver == "s3" && c.S3Bucket == "" {
		return nil, errors.New("S3_BUCKET is required when STORAGE_DRIVER=s3")
	}
	return c, nil
}

func atoi(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
