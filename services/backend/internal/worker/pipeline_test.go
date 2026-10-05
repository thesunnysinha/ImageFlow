package worker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"app/internal/httpapi"
)

// TestClientJourneyOverHTTP drives the real router, store, worker and storage together, the way a client does:
// create a job, watch it finish, download the compressed image, and check that nobody else can see it.
func TestClientJourneyOverHTTP(t *testing.T) {
	h := start(t, 2)
	jpg := jpegData(t)
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/photo.jpg" {
			_, _ = w.Write(jpg)
			return
		}
		w.WriteHeader(404)
	}))
	defer src.Close()

	api := httptest.NewServer(httpapi.New(httpapi.Dependencies{
		Store: h.store, Storage: h.files, APIKeys: []string{"alice-key", "bob-key"}, MaxItemsPerJob: 10,
		// The test image server is on loopback, which the production validator rightly refuses.
		ValidateURL: func(context.Context, string) error { return nil },
		Logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}))
	defer api.Close()

	do := func(method, path, key, body string) (int, http.Header, []byte) {
		req, _ := http.NewRequest(method, api.URL+path, strings.NewReader(body))
		if key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, resp.Header, b
	}
	type envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	decode := func(b []byte, into any) {
		var e envelope
		if err := json.Unmarshal(b, &e); err != nil || !e.Success {
			t.Fatalf("not a success envelope: %s", b)
		}
		if err := json.Unmarshal(e.Data, into); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Create the job with one good and one missing image.
	code, loc, body := do("POST", "/api/v1/jobs", "alice-key", `{"source_urls":["`+src.URL+`/photo.jpg","`+src.URL+`/missing.jpg"]}`)
	if code != 202 {
		t.Fatalf("create: %d %s", code, body)
	}
	var created struct{ ID, Status string }
	decode(body, &created)
	if created.Status != "queued" || loc.Get("Location") != "/api/v1/jobs/"+created.ID {
		t.Fatalf("created job: %+v location=%q", created, loc.Get("Location"))
	}

	// 2. Poll until the worker has finished it.
	type item struct {
		Position int
		Status   string
		BytesIn  int64 `json:"bytes_in"`
		BytesOut int64 `json:"bytes_out"`
		Error    string
	}
	var job struct {
		Status string
		Items  []item
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		_, _, b := do("GET", "/api/v1/jobs/"+created.ID, "alice-key", "")
		job.Items = nil
		decode(b, &job)
		if job.Status == "partial" || job.Status == "completed" || job.Status == "failed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job did not finish: %s", b)
		}
		time.Sleep(25 * time.Millisecond)
	}
	if job.Status != "partial" || job.Items[0].Status != "completed" || job.Items[1].Status != "failed" || !strings.Contains(job.Items[1].Error, "404") {
		t.Fatalf("unexpected result: %+v", job)
	}
	if job.Items[0].BytesOut == 0 || job.Items[0].BytesOut >= job.Items[0].BytesIn {
		t.Fatalf("the image should be smaller: %+v", job.Items[0])
	}

	// 3. Download the result and check it is a real, smaller image.
	code, hdr, img := do("GET", "/api/v1/jobs/"+created.ID+"/items/0/output", "alice-key", "")
	if code != 200 || hdr.Get("Content-Type") != "image/jpeg" || int64(len(img)) != job.Items[0].BytesOut {
		t.Fatalf("download: %d type=%q len=%d want %d", code, hdr.Get("Content-Type"), len(img), job.Items[0].BytesOut)
	}
	if cfg, format, err := image.DecodeConfig(bytes.NewReader(img)); err != nil || format != "jpeg" || cfg.Width != 300 || cfg.Height != 200 {
		t.Fatalf("not the expected JPEG: %v %s %+v", err, format, cfg)
	}
	if code, _, _ := do("GET", "/api/v1/jobs/"+created.ID+"/items/1/output", "alice-key", ""); code != 404 {
		t.Fatalf("a failed item has no output, got %d", code)
	}

	// 4. The job shows up in the list with counts; local storage cannot presign.
	_, _, b := do("GET", "/api/v1/jobs", "alice-key", "")
	var list []struct {
		ID     string
		Status string
		Counts struct{ Total, Completed, Failed int }
	}
	decode(b, &list)
	if len(list) != 1 || list[0].ID != created.ID || list[0].Counts.Total != 2 || list[0].Counts.Completed != 1 || list[0].Counts.Failed != 1 {
		t.Fatalf("list: %+v", list)
	}
	if code, _, _ := do("GET", "/api/v1/jobs/"+created.ID+"/items/0/output-url", "alice-key", ""); code != 501 {
		t.Fatalf("local storage cannot presign, got %d", code)
	}

	// 5. Another key sees nothing of it, and an unauthenticated caller gets nothing.
	for _, p := range []string{"/api/v1/jobs/" + created.ID, "/api/v1/jobs/" + created.ID + "/items/0/output"} {
		if code, _, _ := do("GET", p, "bob-key", ""); code != 404 {
			t.Errorf("bob must not see %s: %d", p, code)
		}
		if code, _, _ := do("GET", p, "", ""); code != 401 {
			t.Errorf("anonymous must be refused %s: %d", p, code)
		}
	}
	_, _, b = do("GET", "/api/v1/jobs", "bob-key", "")
	var bobs []json.RawMessage
	decode(b, &bobs)
	if len(bobs) != 0 {
		t.Fatalf("bob's list must be empty: %s", b)
	}
}
