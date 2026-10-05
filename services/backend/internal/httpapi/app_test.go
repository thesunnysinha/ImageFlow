package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/internal/jobs"
)

type fakeStore struct {
	byOwner map[string]jobs.Job
	pingErr error
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }
func (f *fakeStore) Create(_ context.Context, owner string, w *string, urls []string) (jobs.Job, error) {
	j := jobs.Job{ID: "11111111-1111-1111-1111-111111111111", Status: "queued", WebhookURL: w, Items: []jobs.Item{}}
	for i, u := range urls {
		j.Items = append(j.Items, jobs.Item{Position: i, SourceURL: u, Status: "queued"})
	}
	f.byOwner[owner] = j
	return j, nil
}
func (f *fakeStore) Get(_ context.Context, owner, id string) (jobs.Job, error) {
	if j, ok := f.byOwner[owner]; ok && j.ID == id {
		return j, nil
	}
	return jobs.Job{}, jobs.ErrNotFound
}

const jobID = "11111111-1111-1111-1111-111111111111"

func newApp(s *fakeStore, max int) http.Handler {
	return New(Dependencies{
		Store: s, APIKeys: []string{"k1", "k2"}, MaxItemsPerJob: max,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		ValidateURL: func(_ context.Context, u string) error {
			if strings.Contains(u, "bad") {
				return errors.New("blocked")
			}
			return nil
		},
	})
}

func call(h http.Handler, method, path, key, body string) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func fresh() *fakeStore { return &fakeStore{byOwner: map[string]jobs.Job{}} }

func TestHealthAnswersAnyHostAndEchoesRequestID(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	req.Host = "10.1.2.3:8000"
	req.Header.Set("X-Request-ID", "abc")
	w := httptest.NewRecorder()
	newApp(fresh(), 10).ServeHTTP(w, req)
	var env map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if w.Code != 200 || w.Header().Get("X-Request-ID") != "abc" || env["success"] != true || env["trace_id"] != "abc" {
		t.Fatalf("code=%d id=%q env=%v", w.Code, w.Header().Get("X-Request-ID"), env)
	}
}

func TestReadyReportsDatabase(t *testing.T) {
	if w, _ := call(newApp(fresh(), 10), "GET", "/api/v1/ready", "", ""); w.Code != 200 {
		t.Fatalf("ready=%d", w.Code)
	}
	s := fresh()
	s.pingErr = errors.New("down")
	if w, env := call(newApp(s, 10), "GET", "/api/v1/ready", "", ""); w.Code != 503 || env["code"] != "NOT_READY" {
		t.Fatalf("code=%d env=%v", w.Code, env)
	}
}

func TestAuthRequired(t *testing.T) {
	for _, key := range []string{"", "wrong"} {
		w, env := call(newApp(fresh(), 10), "POST", "/api/v1/jobs", key, `{}`)
		if w.Code != 401 || env["code"] != "UNAUTHORIZED" {
			t.Errorf("key %q: code=%d env=%v", key, w.Code, env)
		}
	}
}

func TestCreateAndGetAreScopedToTheKey(t *testing.T) {
	h := newApp(fresh(), 10)
	w, env := call(h, "POST", "/api/v1/jobs", "k1", `{"source_urls":["https://a.com/1.jpg"]}`)
	if w.Code != 202 || w.Header().Get("Location") != "/api/v1/jobs/"+jobID || env["success"] != true {
		t.Fatalf("code=%d loc=%q env=%v", w.Code, w.Header().Get("Location"), env)
	}
	if w, _ := call(h, "GET", "/api/v1/jobs/"+jobID, "k1", ""); w.Code != 200 {
		t.Fatalf("own job: %d", w.Code)
	}
	if w, env := call(h, "GET", "/api/v1/jobs/"+jobID, "k2", ""); w.Code != 404 || env["code"] != "NOT_FOUND" {
		t.Fatalf("another key must not see it: %d %v", w.Code, env)
	}
}

func TestCreateValidation(t *testing.T) {
	h := newApp(fresh(), 2)
	for name, body := range map[string]string{
		"empty list": `{"source_urls":[]}`, "not json": `not json`, "too many": `{"source_urls":["a","b","c"]}`,
		"blocked source":  `{"source_urls":["https://bad.com/x"]}`,
		"blocked webhook": `{"source_urls":["https://ok.com/x"],"webhook_url":"https://bad.com/h"}`,
	} {
		w, env := call(h, "POST", "/api/v1/jobs", "k1", body)
		if w.Code != 422 || env["code"] != "VALIDATION_ERROR" {
			t.Errorf("%s: code=%d env=%v", name, w.Code, env)
		}
	}
}

func TestUnknownRouteAndMethodUseTheEnvelope(t *testing.T) {
	h := newApp(fresh(), 10)
	if w, env := call(h, "GET", "/nope", "", ""); w.Code != 404 || env["code"] != "NOT_FOUND" {
		t.Fatalf("%d %v", w.Code, env)
	}
	if w, env := call(h, "DELETE", "/api/v1/health", "", ""); w.Code != 405 || env["success"] != false {
		t.Fatalf("%d %v", w.Code, env)
	}
}
