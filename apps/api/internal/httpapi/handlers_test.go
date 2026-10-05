package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thesunnysinha/imageflow/apps/api/internal/jobs"
)

type fakeStore struct{ byOwner map[string]jobs.Job }

func (f *fakeStore) Ping(context.Context) error { return nil }
func (f *fakeStore) Create(_ context.Context, owner string, w *string, urls []string) (jobs.Job, error) {
	j := jobs.Job{ID: "11111111-1111-1111-1111-111111111111", Status: "queued", WebhookURL: w}
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

func newTestRouter(max int) http.Handler {
	return NewRouter(Deps{
		Store:          &fakeStore{byOwner: map[string]jobs.Job{}},
		APIKeys:        []string{"k1", "k2"},
		MaxItemsPerJob: max,
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		ValidateURL: func(_ context.Context, u string) error {
			if strings.Contains(u, "bad") {
				return errors.New("blocked")
			}
			return nil
		},
	})
}

func do(h http.Handler, method, path, key, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHealth(t *testing.T) {
	assert.Equal(t, 200, do(newTestRouter(10), "GET", "/healthz", "", "").Code)
	assert.Equal(t, 200, do(newTestRouter(10), "GET", "/readyz", "", "").Code)
}

func TestAuthRequired(t *testing.T) {
	assert.Equal(t, 401, do(newTestRouter(10), "POST", "/v1/jobs", "", `{}`).Code)
	assert.Equal(t, 401, do(newTestRouter(10), "POST", "/v1/jobs", "wrong", `{}`).Code)
}

func TestCreateAndGet(t *testing.T) {
	h := newTestRouter(10)
	w := do(h, "POST", "/v1/jobs", "k1", `{"source_urls":["https://a.com/1.jpg"]}`)
	require.Equal(t, 202, w.Code)
	assert.Equal(t, "/v1/jobs/11111111-1111-1111-1111-111111111111", w.Header().Get("Location"))

	assert.Equal(t, 200, do(h, "GET", "/v1/jobs/11111111-1111-1111-1111-111111111111", "k1", "").Code)
	// Another API key must not see this owner's job.
	assert.Equal(t, 404, do(h, "GET", "/v1/jobs/11111111-1111-1111-1111-111111111111", "k2", "").Code)
}

func TestCreateValidation(t *testing.T) {
	h := newTestRouter(2)
	assert.Equal(t, 400, do(h, "POST", "/v1/jobs", "k1", `{"source_urls":[]}`).Code)
	assert.Equal(t, 400, do(h, "POST", "/v1/jobs", "k1", `not json`).Code)
	assert.Equal(t, 422, do(h, "POST", "/v1/jobs", "k1", `{"source_urls":["a","b","c"]}`).Code)
	assert.Equal(t, 422, do(h, "POST", "/v1/jobs", "k1", `{"source_urls":["https://bad.com/x"]}`).Code)
	assert.Equal(t, 422, do(h, "POST", "/v1/jobs", "k1", `{"source_urls":["https://ok.com/x"],"webhook_url":"https://bad.com/h"}`).Code)
}
