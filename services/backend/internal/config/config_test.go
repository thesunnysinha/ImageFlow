package config

import "testing"

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestDefaultsAndPostgresVariables(t *testing.T) {
	c, err := Load(env(map[string]string{"API_KEYS": " a , b ,", "POSTGRES_PASSWORD": "p@ss/word"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr() != "0.0.0.0:8000" || c.MaxItemsPerJob != 1000 || len(c.APIKeys) != 2 || c.APIKeys[1] != "b" {
		t.Fatalf("unexpected config: %+v", c)
	}
	want := "postgresql://app:p%40ss%2Fword@database:5432/app"
	if c.DatabaseURL != want {
		t.Fatalf("DatabaseURL=%q want %q", c.DatabaseURL, want)
	}
}

func TestDatabaseURLWins(t *testing.T) {
	c, _ := Load(env(map[string]string{"API_KEYS": "k", "DATABASE_URL": "postgres://x/y", "POSTGRES_HOST": "ignored"}))
	if c.DatabaseURL != "postgres://x/y" {
		t.Fatalf("got %q", c.DatabaseURL)
	}
}

func TestInvalidValues(t *testing.T) {
	for _, m := range []map[string]string{
		{}, {"API_KEYS": " , "},
		{"API_KEYS": "k", "PORT": "abc"}, {"API_KEYS": "k", "PORT": "70000"},
		{"API_KEYS": "k", "MAX_ITEMS_PER_JOB": "0"},
	} {
		if _, err := Load(env(m)); err == nil {
			t.Errorf("expected error for %v", m)
		}
	}
}
