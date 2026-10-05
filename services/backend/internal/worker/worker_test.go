package worker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"app/internal/database"
	"app/internal/jobs"
	"app/internal/safeurl"
	"app/internal/storage"
	"app/internal/worker"
	"app/migrations"
)

// These tests need a real database; they skip unless TEST_DATABASE_URL is set.
func newPool(t *testing.T, schema string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	// Each test package owns a schema, so packages running in parallel never touch each other's tables.
	boot, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer boot.Close()
	if _, err := boot.Exec(ctx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE; CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	return pool
}

func setup(t *testing.T) (*jobs.PGStore, *pgxpool.Pool) {
	t.Helper()
	pool := newPool(t, "test_worker")
	return jobs.NewPGStore(pool), pool
}

func testImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{uint8(x * 7), uint8(y * 13), uint8((x ^ y) * 5), 255})
		}
	}
	return img
}

func jpegData(t *testing.T) []byte {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, testImage(300, 200), &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func pngData(t *testing.T) []byte {
	var b bytes.Buffer
	if err := png.Encode(&b, testImage(64, 64)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

type harness struct {
	store  *jobs.PGStore
	pool   *pgxpool.Pool
	files  *storage.Local
	worker *worker.Worker
	hooks  chan hookCall
	cancel context.CancelFunc
	done   chan struct{}
}

type hookCall struct {
	body      []byte
	timestamp string
	signature string
}

func start(t *testing.T, concurrency int) *harness {
	store, pool := setup(t)
	files, err := storage.NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg := worker.DefaultConfig()
	cfg.Concurrency = concurrency
	cfg.Poll = 20 * time.Millisecond
	cfg.WebhookSecret = "test-secret"
	h := &harness{store: store, pool: pool, files: files, hooks: make(chan hookCall, 10), done: make(chan struct{})}
	h.worker = &worker.Worker{
		Queue: store, Storage: files, Config: cfg, Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		// The test servers listen on loopback, so private addresses are allowed here only.
		Fetch: safeurl.NewClient(10*time.Second, true, true),
		Hooks: safeurl.NewClient(5*time.Second, true, false),
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	go func() { h.worker.Run(ctx); close(h.done) }()
	t.Cleanup(func() { cancel(); <-h.done })
	return h
}

func (h *harness) waitFor(t *testing.T, id string, want ...string) jobs.Job {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		j, err := h.store.Get(context.Background(), "o", id)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range want {
			if j.Status == w {
				return j
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	j, _ := h.store.Get(context.Background(), "o", id)
	t.Fatalf("job %s did not reach %v; now %q items=%+v", id, want, j.Status, j.Items)
	return j
}

func hookServer(h *harness, status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		h.hooks <- hookCall{b, r.Header.Get("X-ImageFlow-Timestamp"), r.Header.Get("X-ImageFlow-Signature")}
		w.WriteHeader(status)
	}))
}

func TestPartialJobStoresOutputsAndSendsASignedWebhook(t *testing.T) {
	h := start(t, 3)
	jpg, pn := jpegData(t), pngData(t)
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a.jpg":
			_, _ = w.Write(jpg)
		case "/b.png":
			_, _ = w.Write(pn)
		default:
			w.WriteHeader(404)
		}
	}))
	defer src.Close()
	hook := hookServer(h, 200)
	defer hook.Close()

	hookURL := hook.URL
	job, err := h.store.Create(context.Background(), "o", &hookURL, []string{src.URL + "/a.jpg", src.URL + "/missing.jpg", src.URL + "/b.png"})
	if err != nil {
		t.Fatal(err)
	}
	got := h.waitFor(t, job.ID, "partial")
	if got.Items[0].Status != "completed" || got.Items[1].Status != "failed" || got.Items[2].Status != "completed" {
		t.Fatalf("unexpected items: %+v", got.Items)
	}
	if got.Items[1].Error == nil || !strings.Contains(*got.Items[1].Error, "404") {
		t.Fatalf("failed item should say why: %+v", got.Items[1])
	}
	if *got.Items[0].BytesOut >= *got.Items[0].BytesIn {
		t.Fatalf("JPEG should shrink: in=%d out=%d", *got.Items[0].BytesIn, *got.Items[0].BytesOut)
	}
	for i, ext := range map[int]string{0: ".jpg", 2: ".png"} {
		key, err := h.store.OutputKey(context.Background(), "o", job.ID, i)
		if err != nil || !strings.HasSuffix(key, ext) {
			t.Fatalf("item %d key=%q err=%v", i, key, err)
		}
		rc, err := h.files.Open(context.Background(), key)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := image.Decode(rc); err != nil {
			t.Fatalf("stored output %d is not a valid image: %v", i, err)
		}
		rc.Close()
	}
	if _, err := h.store.OutputKey(context.Background(), "o", job.ID, 1); err != jobs.ErrNotFound {
		t.Fatalf("a failed item has no output, got %v", err)
	}
	if _, err := h.store.OutputKey(context.Background(), "someone-else", job.ID, 0); err != jobs.ErrNotFound {
		t.Fatalf("another owner must not get the output, got %v", err)
	}

	select {
	case call := <-h.hooks:
		if call.signature != worker.Sign("test-secret", call.timestamp, call.body) {
			t.Fatalf("bad signature %q", call.signature)
		}
		var p struct {
			JobID  string      `json:"job_id"`
			Status string      `json:"status"`
			Counts jobs.Counts `json:"counts"`
		}
		_ = json.Unmarshal(call.body, &p)
		if p.JobID != job.ID || p.Status != "partial" || p.Counts != (jobs.Counts{Completed: 2, Failed: 1, Total: 3}) {
			t.Fatalf("unexpected payload: %s", call.body)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no webhook delivered")
	}
	// Delivered exactly once.
	select {
	case <-h.hooks:
		t.Fatal("webhook delivered twice")
	case <-time.After(500 * time.Millisecond):
	}
}

func TestTransientFailuresRetryAndThenSucceed(t *testing.T) {
	h := start(t, 1)
	jpg := jpegData(t)
	var calls atomic.Int32
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		_, _ = w.Write(jpg)
	}))
	defer src.Close()
	job, _ := h.store.Create(context.Background(), "o", nil, []string{src.URL + "/x.jpg"})

	// After the first failure the item is re-queued with a backoff; skip the wait.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && calls.Load() < 1 {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	if _, err := h.pool.Exec(context.Background(), `UPDATE job_items SET next_attempt_at = now() WHERE job_id = $1`, job.ID); err != nil {
		t.Fatal(err)
	}
	got := h.waitFor(t, job.ID, "completed")
	if calls.Load() != 2 || got.Items[0].Status != "completed" {
		t.Fatalf("calls=%d item=%+v", calls.Load(), got.Items[0])
	}
}

func TestTransientFailuresGiveUpAfterMaxAttempts(t *testing.T) {
	h := start(t, 1)
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	defer src.Close()
	job, _ := h.store.Create(context.Background(), "o", nil, []string{src.URL + "/x.jpg"})
	for i := 0; i < 3; i++ {
		time.Sleep(150 * time.Millisecond)
		_, _ = h.pool.Exec(context.Background(), `UPDATE job_items SET next_attempt_at = now() WHERE job_id = $1 AND status = 'queued'`, job.ID)
	}
	got := h.waitFor(t, job.ID, "failed")
	var attempts int
	_ = h.pool.QueryRow(context.Background(), `SELECT attempts FROM job_items WHERE job_id = $1`, job.ID).Scan(&attempts)
	if attempts != 3 || got.Items[0].Error == nil {
		t.Fatalf("attempts=%d item=%+v", attempts, got.Items[0])
	}
}

func TestManyWorkersProcessEachItemExactlyOnce(t *testing.T) {
	h := start(t, 6)
	jpg := jpegData(t)
	var mu sync.Mutex
	seen := map[string]int{}
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.Path]++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		_, _ = w.Write(jpg)
	}))
	defer src.Close()
	var urls []string
	for i := 0; i < 30; i++ {
		urls = append(urls, src.URL+"/img"+string(rune('a'+i%26))+string(rune('a'+i/26))+".jpg")
	}
	job, _ := h.store.Create(context.Background(), "o", nil, urls)
	got := h.waitFor(t, job.ID, "completed")
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 30 {
		t.Fatalf("expected 30 distinct fetches, got %d", len(seen))
	}
	for path, n := range seen {
		if n != 1 {
			t.Errorf("%s fetched %d times", path, n)
		}
	}
	for _, it := range got.Items {
		if it.Status != "completed" {
			t.Errorf("item %d is %s", it.Position, it.Status)
		}
	}
}

func TestAbandonedItemsAreReclaimedAfterTheLease(t *testing.T) {
	store, pool := setup(t)
	ctx := context.Background()
	job, _ := store.Create(ctx, "o", nil, []string{"https://example.com/a.jpg"})
	first, err := store.ClaimItem(ctx, time.Minute)
	if err != nil || first == nil || first.Attempts != 1 {
		t.Fatalf("claim: %+v %v", first, err)
	}
	if again, _ := store.ClaimItem(ctx, time.Minute); again != nil {
		t.Fatal("a leased item must not be claimed twice")
	}
	_, _ = pool.Exec(ctx, `UPDATE job_items SET locked_at = now() - interval '10 minutes' WHERE job_id = $1`, job.ID)
	second, err := store.ClaimItem(ctx, time.Minute)
	if err != nil || second == nil || second.ItemID != first.ItemID || second.Attempts != 2 {
		t.Fatalf("expired lease should be reclaimed: %+v %v", second, err)
	}
}

func TestFailedWebhookIsRetried(t *testing.T) {
	h := start(t, 1)
	jpg := jpegData(t)
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(jpg) }))
	defer src.Close()
	var status atomic.Int32
	status.Store(500)
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.hooks <- hookCall{}
		w.WriteHeader(int(status.Load()))
	}))
	defer hook.Close()
	hookURL := hook.URL
	job, _ := h.store.Create(context.Background(), "o", &hookURL, []string{src.URL + "/a.jpg"})
	h.waitFor(t, job.ID, "completed")
	<-h.hooks // first attempt, answered with 500
	status.Store(200)
	time.Sleep(200 * time.Millisecond)
	_, _ = h.pool.Exec(context.Background(), `UPDATE jobs SET webhook_next_at = now() WHERE id = $1`, job.ID)
	select {
	case <-h.hooks:
	case <-time.After(10 * time.Second):
		t.Fatal("webhook was not retried")
	}
	time.Sleep(300 * time.Millisecond)
	var pending bool
	_ = h.pool.QueryRow(context.Background(), `SELECT webhook_pending FROM jobs WHERE id = $1`, job.ID).Scan(&pending)
	if pending {
		t.Fatal("a delivered webhook must stop being pending")
	}
}
