package janitor_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"app/internal/database"
	"app/internal/janitor"
	"app/internal/jobs"
	"app/internal/storage"
	"app/migrations"
)

// Needs a real database; skipped unless TEST_DATABASE_URL is set.
func setup(t *testing.T) (*jobs.PGStore, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	boot, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()
	if _, err := boot.Exec(ctx, `DROP SCHEMA IF EXISTS test_janitor CASCADE; CREATE SCHEMA test_janitor`); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "test_janitor"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return jobs.NewPGStore(pool), pool
}

// finishedJob creates a job with one completed item whose output exists in storage, then backdates it.
func finishedJob(t *testing.T, s *jobs.PGStore, pool *pgxpool.Pool, files storage.Storage, owner, status string, age time.Duration) (id, key string) {
	t.Helper()
	ctx := context.Background()
	job, err := s.Create(ctx, owner, nil, []string{"https://x/1.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	key = job.ID + "/0.jpg"
	if _, err := files.Put(ctx, key, strings.NewReader("image")); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE job_items SET status='completed', output_key=$2 WHERE job_id=$1`, job.ID, key); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jobs SET status=$2::job_status, created_at=now()-make_interval(secs => $3) WHERE id=$1`,
		job.ID, status, age.Seconds()); err != nil {
		t.Fatal(err)
	}
	return job.ID, key
}

func newJanitor(s *jobs.PGStore, files storage.Storage, retention time.Duration) *janitor.Janitor {
	return &janitor.Janitor{Store: s, Storage: files, Retention: retention, BatchSize: 2, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
}

func exists(files storage.Storage, key string) bool {
	rc, err := files.Open(context.Background(), key)
	if err != nil {
		return false
	}
	rc.Close()
	return true
}

func TestSweepDeletesExpiredFinishedJobsAndTheirFiles(t *testing.T) {
	s, pool := setup(t)
	files, _ := storage.NewLocal(t.TempDir())
	ctx := context.Background()

	var oldIDs, oldKeys []string
	for _, status := range []string{"completed", "partial", "failed", "completed", "completed"} { // 5 > the batch size of 2
		id, key := finishedJob(t, s, pool, files, "o", status, 48*time.Hour)
		oldIDs, oldKeys = append(oldIDs, id), append(oldKeys, key)
	}
	recentID, recentKey := finishedJob(t, s, pool, files, "o", "completed", time.Hour)
	runningID, runningKey := finishedJob(t, s, pool, files, "o", "processing", 72*time.Hour) // old but not finished
	queuedID, _ := finishedJob(t, s, pool, files, "o", "queued", 72*time.Hour)

	res, err := newJanitor(s, files, 24*time.Hour).RunOnce(ctx, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.Jobs != 5 || res.Objects != 5 {
		t.Fatalf("want 5 jobs and 5 files removed across batches, got %+v", res)
	}
	for i, id := range oldIDs {
		if _, err := s.Get(ctx, "o", id); err != jobs.ErrNotFound {
			t.Errorf("expired job %s should be gone, got %v", id, err)
		}
		if exists(files, oldKeys[i]) {
			t.Errorf("file %s should be deleted", oldKeys[i])
		}
	}
	for _, id := range []string{recentID, runningID, queuedID} {
		if _, err := s.Get(ctx, "o", id); err != nil {
			t.Errorf("job %s must be kept: %v", id, err)
		}
	}
	if !exists(files, recentKey) || !exists(files, runningKey) {
		t.Error("files of kept jobs must be kept")
	}
	// Items go with their job (cascade).
	var items int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM job_items WHERE job_id = ANY($1::uuid[])`, oldIDs).Scan(&items)
	if items != 0 {
		t.Errorf("%d items of deleted jobs remain", items)
	}
}

func TestSweepIsSafeToRepeatAndToResumeAfterAnInterruption(t *testing.T) {
	s, pool := setup(t)
	files, _ := storage.NewLocal(t.TempDir())
	ctx := context.Background()
	id, key := finishedJob(t, s, pool, files, "o", "completed", 48*time.Hour)

	// Simulate a crash after the file was deleted but before the row was: the next sweep must still succeed.
	if err := files.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	j := newJanitor(s, files, 24*time.Hour)
	res, err := j.RunOnce(ctx, time.Now())
	if err != nil || res.Jobs != 1 {
		t.Fatalf("resume: %+v %v", res, err)
	}
	if _, err := s.Get(ctx, "o", id); err != jobs.ErrNotFound {
		t.Fatalf("job should be gone, got %v", err)
	}
	if res, err = j.RunOnce(ctx, time.Now()); err != nil || res.Jobs != 0 {
		t.Fatalf("a second sweep finds nothing: %+v %v", res, err)
	}
}
