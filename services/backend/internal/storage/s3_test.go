package storage

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
)

// newFakeS3 starts an in-process S3-compatible server (real protocol, in memory).
func newFakeS3(t *testing.T, bucket string) S3Config {
	t.Helper()
	backend := s3mem.New()
	if err := backend.CreateBucket(bucket); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(decodeAWSChunked(gofakes3.New(backend).Server()))
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	return S3Config{Endpoint: u.Host, Bucket: bucket, Region: "us-east-1", AccessKey: "test", SecretKey: "test-secret", UseSSL: false}
}

// decodeAWSChunked unwraps the "aws-chunked" upload framing that minio-go uses for streaming PUTs over
// plain HTTP. Real S3 and MinIO do this themselves; gofakes3 would store the framing as object data.
func decodeAWSChunked(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Amz-Decoded-Content-Length") != "" {
			br := bufio.NewReader(r.Body)
			var plain bytes.Buffer
			for {
				line, err := br.ReadString('\n')
				if err != nil {
					break
				}
				size, err := strconv.ParseInt(strings.SplitN(strings.TrimSpace(line), ";", 2)[0], 16, 64)
				if err != nil || size == 0 {
					break
				}
				_, _ = io.CopyN(&plain, br, size)
				_, _ = br.Discard(2) // CRLF after each chunk
			}
			r.Body = io.NopCloser(&plain)
			r.ContentLength = int64(plain.Len())
			r.Header.Set("Content-Length", strconv.Itoa(plain.Len()))
			r.Header.Del("X-Amz-Decoded-Content-Length")
			r.Header.Del("Content-Encoding")
		}
		next.ServeHTTP(w, r)
	})
}

func TestS3PutOpenRoundTripWithPrefix(t *testing.T) {
	cfg := newFakeS3(t, "images")
	cfg.Prefix = "imageflow"
	s, err := NewS3(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	n, err := s.Put(context.Background(), "job/0.jpg", strings.NewReader("hello"))
	if err != nil || n != 5 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	rc, err := s.Open(context.Background(), "job/0.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	if b, _ := io.ReadAll(rc); string(b) != "hello" {
		t.Fatalf("got %q", b)
	}
	// The prefix is applied to the stored object name.
	other, _ := NewS3(context.Background(), S3Config{Endpoint: cfg.Endpoint, Bucket: cfg.Bucket, Region: cfg.Region, AccessKey: "test", SecretKey: "test-secret"})
	if rc, err := other.Open(context.Background(), "imageflow/job/0.jpg"); err != nil {
		t.Fatalf("object should live under the prefix: %v", err)
	} else {
		rc.Close()
	}
}

func TestS3MissingObjectsAndBadKeys(t *testing.T) {
	s, err := NewS3(context.Background(), newFakeS3(t, "images"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(context.Background(), "nope/0.jpg"); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := s.PresignGet(context.Background(), "nope/0.jpg", time.Minute); err != ErrNotFound {
		t.Fatalf("presigning a missing object must be ErrNotFound, got %v", err)
	}
	for _, bad := range []string{"", "/abs", "a/", "a/../b", "./a", "a//b", "a\x00b"} {
		if _, err := s.Put(context.Background(), bad, strings.NewReader("x")); err == nil {
			t.Errorf("Put(%q) should be rejected", bad)
		}
		if _, err := s.Open(context.Background(), bad); err != ErrNotFound {
			t.Errorf("Open(%q) should be ErrNotFound, got %v", bad, err)
		}
	}
}

func TestS3PresignedURLDownloadsWithoutCredentials(t *testing.T) {
	s, err := NewS3(context.Background(), newFakeS3(t, "images"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Put(context.Background(), "job/1.png", strings.NewReader("PNGDATA")); err != nil {
		t.Fatal(err)
	}
	signed, err := s.PresignGet(context.Background(), "job/1.png", 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(signed)
	if u.Query().Get("X-Amz-Expires") != "300" || u.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("not a signed URL with a 5 minute lifetime: %s", signed)
	}
	resp, err := http.Get(signed) // plain client: no Authorization header
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if b, _ := io.ReadAll(resp.Body); resp.StatusCode != 200 || string(b) != "PNGDATA" {
		t.Fatalf("status=%d body=%q", resp.StatusCode, b)
	}
}

func TestNewS3FailsFastOnBadConfiguration(t *testing.T) {
	cfg := newFakeS3(t, "images")
	cfg.Bucket = "missing-bucket"
	if _, err := NewS3(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("a missing bucket must fail at start, got %v", err)
	}
	if _, err := NewS3(context.Background(), S3Config{}); err == nil {
		t.Fatal("empty config must fail")
	}
}

func TestS3DeleteRemovesTheObjectAndIsIdempotent(t *testing.T) {
	s, err := NewS3(context.Background(), newFakeS3(t, "images"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := s.Put(ctx, "job/0.jpg", strings.NewReader("x")); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "job/0.jpg"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(ctx, "job/0.jpg"); err != ErrNotFound {
		t.Fatalf("object should be gone, got %v", err)
	}
	for _, key := range []string{"job/0.jpg", "never/existed", "../bad", ""} {
		if err := s.Delete(ctx, key); err != nil {
			t.Errorf("Delete(%q) must not fail: %v", key, err)
		}
	}
}

var _ Presigner = (*S3)(nil)
var _ Storage = (*S3)(nil)
