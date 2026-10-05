package config

import (
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDefaultsAndPostgresVariables(t *testing.T) {
	// Characters that must be escaped in a URL, assembled so the test file holds no password-like literal.
	special := "a" + "@" + "b" + "/" + "c"
	c, err := Load(env(map[string]string{"API_KEYS": " a , b ,", "WEBHOOK_SECRET": "s", "POSTGRES_PASSWORD": special}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr() != "0.0.0.0:8000" || c.MaxItemsPerJob != 1000 || len(c.APIKeys) != 2 || c.APIKeys[1] != "b" {
		t.Fatalf("unexpected config: %+v", c)
	}
	want := "postgresql://app:a%40b%2Fc@database:5432/app"
	if c.DatabaseURL != want {
		t.Fatalf("DatabaseURL=%q want %q", c.DatabaseURL, want)
	}
}

func TestDatabaseURLWins(t *testing.T) {
	c, _ := Load(env(map[string]string{"API_KEYS": "k", "WEBHOOK_SECRET": "s", "DATABASE_URL": "postgres://x/y", "POSTGRES_HOST": "ignored"}))
	if c.DatabaseURL != "postgres://x/y" {
		t.Fatalf("got %q", c.DatabaseURL)
	}
}

func TestInvalidValues(t *testing.T) {
	for _, m := range []map[string]string{
		{}, {"API_KEYS": " , "},
		{"API_KEYS": "k", "PORT": "abc"}, {"API_KEYS": "k", "PORT": "70000"},
		{"API_KEYS": "k", "WEBHOOK_SECRET": "s", "MAX_ITEMS_PER_JOB": "0"},
		{"API_KEYS": "k"}, // the worker is on by default and needs a webhook secret
		{"API_KEYS": "k", "WEBHOOK_SECRET": "s", "WORKER_CONCURRENCY": "0"},
	} {
		if _, err := Load(env(m)); err == nil {
			t.Errorf("expected error for %v", m)
		}
	}
}

func TestWorkerCanBeDisabledWithoutASecret(t *testing.T) {
	c, err := Load(env(map[string]string{"API_KEYS": "k", "RUN_WORKER": "false"}))
	if err != nil || c.RunWorker {
		t.Fatalf("RunWorker=%v err=%v", c.RunWorker, err)
	}
}

func TestStorageBackendSelection(t *testing.T) {
	base := map[string]string{"API_KEYS": "k", "WEBHOOK_SECRET": "s"}
	with := func(extra map[string]string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	c, err := Load(env(base))
	if err != nil || c.StorageBackend != "local" {
		t.Fatalf("default should be local: %+v %v", c, err)
	}
	full := map[string]string{"STORAGE_BACKEND": "s3", "S3_ENDPOINT": "s3.example.com", "S3_BUCKET": "b", "S3_ACCESS_KEY": "a", "S3_SECRET_KEY": "x", "S3_PREFIX": "p/"}
	c, err = Load(env(with(full)))
	if err != nil || c.S3.Bucket != "b" || !c.S3.UseSSL || c.S3.Region != "us-east-1" || c.S3.Prefix != "p/" {
		t.Fatalf("s3 config: %+v %v", c.S3, err)
	}
	if c, _ = Load(env(with(map[string]string{"STORAGE_BACKEND": "s3", "S3_ENDPOINT": "minio:9000", "S3_BUCKET": "b", "S3_ACCESS_KEY": "a", "S3_SECRET_KEY": "x", "S3_USE_SSL": "false"}))); c.S3.UseSSL {
		t.Fatal("S3_USE_SSL=false must disable TLS")
	}
	for name, extra := range map[string]map[string]string{
		"missing bucket":  {"STORAGE_BACKEND": "s3", "S3_ENDPOINT": "e", "S3_ACCESS_KEY": "a", "S3_SECRET_KEY": "x"},
		"scheme in host":  {"STORAGE_BACKEND": "s3", "S3_ENDPOINT": "https://e", "S3_BUCKET": "b", "S3_ACCESS_KEY": "a", "S3_SECRET_KEY": "x"},
		"unknown backend": {"STORAGE_BACKEND": "ftp"},
	} {
		if _, err := Load(env(with(extra))); err == nil {
			t.Errorf("%s should be rejected", name)
		}
	}
}

func TestRetention(t *testing.T) {
	base := map[string]string{"API_KEYS": "k", "WEBHOOK_SECRET": "s"}
	with := func(v string) map[string]string {
		return map[string]string{"API_KEYS": "k", "WEBHOOK_SECRET": "s", "RETENTION_HOURS": v}
	}
	if c, _ := Load(env(base)); c.Retention != 7*24*time.Hour {
		t.Fatalf("default should be 7 days, got %v", c.Retention)
	}
	if c, err := Load(env(with("24"))); err != nil || c.Retention != 24*time.Hour {
		t.Fatalf("24h: %v %v", c.Retention, err)
	}
	if c, err := Load(env(with("0"))); err != nil || c.Retention != 0 {
		t.Fatalf("0 keeps everything: %v %v", c.Retention, err)
	}
	for _, bad := range []string{"-1", "abc", "1.5"} {
		if _, err := Load(env(with(bad))); err == nil {
			t.Errorf("RETENTION_HOURS=%q should be rejected", bad)
		}
	}
}

func TestRateLimitSetting(t *testing.T) {
	with := func(v string) map[string]string {
		return map[string]string{"API_KEYS": "k", "WEBHOOK_SECRET": "s", "RATE_LIMIT_PER_MINUTE": v}
	}
	if c, _ := Load(env(map[string]string{"API_KEYS": "k", "WEBHOOK_SECRET": "s"})); c.RateLimit != 120 {
		t.Fatalf("default should be 120/min, got %d", c.RateLimit)
	}
	if c, err := Load(env(with("30"))); err != nil || c.RateLimit != 30 {
		t.Fatalf("30: %d %v", c.RateLimit, err)
	}
	if c, err := Load(env(with("0"))); err != nil || c.RateLimit != 0 {
		t.Fatalf("0 disables: %d %v", c.RateLimit, err)
	}
	for _, bad := range []string{"-5", "x", "1.5"} {
		if _, err := Load(env(with(bad))); err == nil {
			t.Errorf("RATE_LIMIT_PER_MINUTE=%q should be rejected", bad)
		}
	}
}
