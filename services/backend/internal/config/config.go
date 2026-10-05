// Package config reads settings from the environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// S3Settings configures any S3-compatible object store (AWS S3, Cloudflare R2, MinIO, ...).
type S3Settings struct {
	Endpoint, Bucket, Region, AccessKey, SecretKey, Prefix string
	UseSSL                                                 bool
}

type Config struct {
	Host, Port     string
	DatabaseURL    string
	APIKeys        []string
	MaxItemsPerJob int
	StorageBackend string // "local" (default) or "s3"
	StorageDir     string
	S3             S3Settings
	WebhookSecret  string
	RunWorker      bool
	WorkerCount    int
	Retention      time.Duration // finished jobs and their files older than this are deleted; 0 keeps everything
}

// Load reads the environment. Launchpad supplies DATABASE_URL and the POSTGRES_* variables;
// local Compose supplies POSTGRES_* only. API_KEYS (comma-separated) is required.
func Load(get func(string) string) (Config, error) {
	if get == nil {
		get = os.Getenv
	}
	def := func(key, fallback string) string {
		if v := get(key); v != "" {
			return v
		}
		return fallback
	}
	port := def("PORT", "8000")
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("PORT %q is not a valid port", port)
	}
	maxItems := 1000
	if raw := get("MAX_ITEMS_PER_JOB"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("MAX_ITEMS_PER_JOB %q must be a positive integer", raw)
		}
		maxItems = n
	}
	workers := 4
	if raw := get("WORKER_CONCURRENCY"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 64 {
			return Config{}, fmt.Errorf("WORKER_CONCURRENCY %q must be between 1 and 64", raw)
		}
		workers = n
	}
	retention := 7 * 24 * time.Hour
	if raw := get("RETENTION_HOURS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("RETENTION_HOURS %q must be 0 (keep forever) or a positive number of hours", raw)
		}
		retention = time.Duration(n) * time.Hour
	}
	c := Config{Retention: retention, Host: def("HOST", "0.0.0.0"), Port: port, MaxItemsPerJob: maxItems,
		StorageBackend: def("STORAGE_BACKEND", "local"), StorageDir: def("STORAGE_DIR", "/data/images"), WebhookSecret: get("WEBHOOK_SECRET"),
		RunWorker: get("RUN_WORKER") != "false", WorkerCount: workers}
	for _, k := range strings.Split(get("API_KEYS"), ",") {
		if k = strings.TrimSpace(k); k != "" {
			c.APIKeys = append(c.APIKeys, k)
		}
	}
	if len(c.APIKeys) == 0 {
		return Config{}, errors.New("API_KEYS is required (comma-separated)")
	}
	switch c.StorageBackend {
	case "local":
	case "s3":
		c.S3 = S3Settings{
			Endpoint: get("S3_ENDPOINT"), Bucket: get("S3_BUCKET"), Region: def("S3_REGION", "us-east-1"),
			AccessKey: get("S3_ACCESS_KEY"), SecretKey: get("S3_SECRET_KEY"), Prefix: get("S3_PREFIX"),
			UseSSL: get("S3_USE_SSL") != "false",
		}
		if c.S3.Endpoint == "" || c.S3.Bucket == "" || c.S3.AccessKey == "" || c.S3.SecretKey == "" {
			return Config{}, errors.New("STORAGE_BACKEND=s3 requires S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY and S3_SECRET_KEY")
		}
		if strings.Contains(c.S3.Endpoint, "://") {
			return Config{}, errors.New("S3_ENDPOINT must be host[:port] without a scheme (use S3_USE_SSL)")
		}
	default:
		return Config{}, fmt.Errorf("STORAGE_BACKEND %q must be local or s3", c.StorageBackend)
	}
	if c.RunWorker && c.WebhookSecret == "" {
		return Config{}, errors.New("WEBHOOK_SECRET is required when the worker runs (set RUN_WORKER=false to disable it)")
	}
	if c.DatabaseURL = get("DATABASE_URL"); c.DatabaseURL == "" {
		u := url.URL{
			Scheme: "postgresql",
			User:   url.UserPassword(def("POSTGRES_USER", "app"), get("POSTGRES_PASSWORD")),
			Host:   def("POSTGRES_HOST", "database") + ":" + def("POSTGRES_PORT", "5432"),
			Path:   "/" + def("POSTGRES_DB", "app"),
		}
		c.DatabaseURL = u.String()
	}
	return c, nil
}

func (c Config) Addr() string { return c.Host + ":" + c.Port }
