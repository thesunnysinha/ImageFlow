package config

import "testing"

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
