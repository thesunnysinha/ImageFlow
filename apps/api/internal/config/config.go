// Package config loads service configuration from environment variables.
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	APIKeys         []string
	AllowedOrigins  []string
	MaxItemsPerJob  int
	ShutdownTimeout time.Duration
	Production      bool
}

// Load reads configuration and fails fast on missing required values.
func Load() (Config, error) {
	c := Config{
		Addr:            getenv("ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		APIKeys:         splitCSV(os.Getenv("API_KEYS")),
		AllowedOrigins:  splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		MaxItemsPerJob:  getint("MAX_ITEMS_PER_JOB", 1000),
		ShutdownTimeout: 20 * time.Second,
		Production:      os.Getenv("APP_ENV") == "production",
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	if len(c.APIKeys) == 0 {
		return c, errors.New("API_KEYS is required (comma-separated)")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getint(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v > 0 {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
