package jobs_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"app/internal/database"
	"app/internal/jobs"
	"app/migrations"
)

// Needs a real database; skipped unless TEST_DATABASE_URL is set (the generated CI starts Postgres for it).
func newStore(t *testing.T) (*jobs.PGStore, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP TABLE IF EXISTS job_items, jobs, schema_migrations; DROP TYPE IF EXISTS item_status, job_status`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return jobs.NewPGStore(pool), pool
}

func TestMigrateIsIdempotent(t *testing.T) {
	_, pool := newStore(t)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("schema_migrations rows=%d err=%v", n, err)
	}
}

func TestCreateAndGetKeepOrderAndOwnership(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	hook := "https://example.com/hook"
	created, err := s.Create(ctx, "owner-a", &hook, []string{"https://x/1.jpg", "https://x/2.jpg", "https://x/3.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "queued" || len(created.Items) != 3 || created.WebhookURL == nil {
		t.Fatalf("unexpected job: %+v", created)
	}
	for i, it := range created.Items {
		if it.Position != i || it.Status != "queued" {
			t.Fatalf("item %d: %+v", i, it)
		}
	}
	if created.Items[2].SourceURL != "https://x/3.jpg" {
		t.Fatalf("order lost: %+v", created.Items)
	}
	if _, err := s.Get(ctx, "owner-b", created.ID); err != jobs.ErrNotFound {
		t.Fatalf("another owner must get ErrNotFound, got %v", err)
	}
	if _, err := s.Get(ctx, "owner-a", "not-a-uuid"); err != jobs.ErrNotFound {
		t.Fatalf("malformed id must be ErrNotFound, got %v", err)
	}
	if err := s.Ping(ctx); err != nil {
		t.Fatal(err)
	}
}
