package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestLimiterAllowsABurstThenRefills(t *testing.T) {
	c := &clock{t: time.Unix(1_000_000, 0)}
	l := newLimiter(60, c.now) // 1 token per second, burst max(5, 15) = 15
	for i := 0; i < 15; i++ {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("request %d of the burst was refused", i+1)
		}
	}
	ok, wait := l.allow("a")
	if ok || wait < 900*time.Millisecond || wait > 1100*time.Millisecond {
		t.Fatalf("16th request: ok=%v wait=%v, want refused with about 1s", ok, wait)
	}
	c.advance(3 * time.Second)
	for i := 0; i < 3; i++ {
		if ok, _ := l.allow("a"); !ok {
			t.Fatalf("after 3s, request %d should pass", i+1)
		}
	}
	if ok, _ := l.allow("a"); ok {
		t.Fatal("only 3 tokens refilled")
	}
	c.advance(time.Hour) // refill is capped at the burst
	n := 0
	for ok, _ := l.allow("a"); ok; ok, _ = l.allow("a") {
		n++
	}
	if n != 15 {
		t.Fatalf("a long idle must not bank more than the burst: got %d", n)
	}
}

func TestLimiterKeysAreIndependentAndZeroDisables(t *testing.T) {
	c := &clock{t: time.Unix(1_000_000, 0)}
	l := newLimiter(4, c.now) // burst 5
	for i := 0; i < 5; i++ {
		l.allow("a")
	}
	if ok, _ := l.allow("a"); ok {
		t.Fatal("a is exhausted")
	}
	if ok, _ := l.allow("b"); !ok {
		t.Fatal("b has its own bucket")
	}
	if newLimiter(0, nil) != nil {
		t.Fatal("0 must disable limiting")
	}
}

func TestLimiterForgetsIdleKeys(t *testing.T) {
	c := &clock{t: time.Unix(1_000_000, 0)}
	l := newLimiter(60, c.now)
	for i := 0; i < 10_001; i++ {
		l.allow(strings.Repeat("k", 1) + string(rune(i)))
	}
	c.advance(time.Hour)
	l.allow("fresh") // a new key triggers the sweep once the map is large
	if len(l.buckets) > 5 {
		t.Fatalf("idle keys should have been dropped, %d remain", len(l.buckets))
	}
}

func TestHTTPRateLimitIsPerKeyAndSparesHealthChecks(t *testing.T) {
	c := &clock{t: time.Unix(1_000_000, 0)}
	h := New(Dependencies{
		Store: fresh(), Storage: memStorage{}, APIKeys: []string{"k1", "k2"}, MaxItemsPerJob: 10,
		RateLimitPerMinute: 4, Now: c.now, // burst 5
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		ValidateURL: func(context.Context, string) error { return nil },
	})
	get := func(path, key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", path, nil)
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	for i := 0; i < 5; i++ {
		if w := get("/api/v1/jobs", "k1"); w.Code != http.StatusOK {
			t.Fatalf("request %d: %d", i+1, w.Code)
		}
	}
	w := get("/api/v1/jobs", "k1")
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" || !strings.Contains(w.Body.String(), `"code":"RATE_LIMITED"`) {
		t.Fatalf("6th request: %d retry-after=%q body=%s", w.Code, w.Header().Get("Retry-After"), w.Body.String())
	}
	if w := get("/api/v1/jobs", "k2"); w.Code != http.StatusOK {
		t.Fatalf("another key is unaffected: %d", w.Code)
	}
	if w := get("/api/v1/health", ""); w.Code != http.StatusOK {
		t.Fatalf("health checks are never limited: %d", w.Code)
	}
	if w := get("/api/v1/jobs", "wrong"); w.Code != http.StatusUnauthorized {
		t.Fatalf("a bad key is a 401, not a 429: %d", w.Code)
	}
	c.advance(20 * time.Second) // 4/min refills about 1.3 tokens in 20s
	if w := get("/api/v1/jobs", "k1"); w.Code != http.StatusOK {
		t.Fatalf("after waiting, the key works again: %d", w.Code)
	}
}
