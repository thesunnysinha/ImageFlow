// Package config reads settings from the environment.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Host, Port     string
	DatabaseURL    string
	APIKeys        []string
	MaxItemsPerJob int
	StorageDir     string
	WebhookSecret  string
	RunWorker      bool
	WorkerCount    int
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
	c := Config{Host: def("HOST", "0.0.0.0"), Port: port, MaxItemsPerJob: maxItems,
		StorageDir: def("STORAGE_DIR", "/data/images"), WebhookSecret: get("WEBHOOK_SECRET"),
		RunWorker: get("RUN_WORKER") != "false", WorkerCount: workers}
	for _, k := range strings.Split(get("API_KEYS"), ",") {
		if k = strings.TrimSpace(k); k != "" {
			c.APIKeys = append(c.APIKeys, k)
		}
	}
	if len(c.APIKeys) == 0 {
		return Config{}, errors.New("API_KEYS is required (comma-separated)")
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
