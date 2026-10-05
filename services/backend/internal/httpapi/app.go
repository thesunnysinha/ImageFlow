// Package httpapi builds the Gin router. It keeps the HTTP contract shared by every backend blueprint:
// /api/v1/health and /api/v1/ready, the response envelope and X-Request-ID.
package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"app/internal/envelope"
	"app/internal/jobs"
)

const (
	maxBodyBytes = 1 << 20
	traceKey     = "trace_id"
	ownerKey     = "owner"
)

// URLValidator rejects URLs the service must not fetch or call (see package safeurl).
type URLValidator func(ctx context.Context, rawURL string) error

type Dependencies struct {
	Store          jobs.Store
	APIKeys        []string
	MaxItemsPerJob int
	ValidateURL    URLValidator
	Logger         *slog.Logger
}

func New(d Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(requestID(), accessLog(d.Logger), recovery(d.Logger))

	api := r.Group("/api/v1")
	// Liveness never touches the database, and answers whatever Host header a probe sends.
	api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, envelope.OK(gin.H{"status": "ok"}, trace(c))) })
	api.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := d.Store.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, envelope.Failure("NOT_READY", "The database is not reachable.", trace(c), nil))
			return
		}
		c.JSON(http.StatusOK, envelope.OK(gin.H{"status": "ready"}, trace(c)))
	})

	// API-key auth is the interim authenticator; the owner is derived from the key.
	h := &handlers{d: d}
	protected := api.Group("", apiKeyAuth(d.APIKeys))
	protected.POST("/jobs", h.createJob)
	protected.GET("/jobs/:id", h.getJob)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, envelope.Failure("NOT_FOUND", "Not found.", trace(c), nil))
	})
	r.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, envelope.Failure("METHOD_NOT_ALLOWED", "Method not allowed.", trace(c), nil))
	})
	return r
}

type handlers struct{ d Dependencies }

type createJobRequest struct {
	WebhookURL string   `json:"webhook_url"`
	SourceURLs []string `json:"source_urls"`
}

func (h *handlers) createJob(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	var req createJobRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, invalid(c, "body", "The request body must be valid JSON."))
		return
	}
	if n := len(req.SourceURLs); n < 1 || n > h.d.MaxItemsPerJob {
		c.JSON(http.StatusUnprocessableEntity, invalid(c, "source_urls", "Must contain between 1 and the maximum number of URLs."))
		return
	}
	ctx := c.Request.Context()
	for i, u := range req.SourceURLs {
		if err := h.d.ValidateURL(ctx, u); err != nil {
			c.JSON(http.StatusUnprocessableEntity, invalid(c, "source_urls["+itoa(i)+"]", "The URL is not allowed."))
			return
		}
	}
	var webhook *string
	if req.WebhookURL != "" {
		if err := h.d.ValidateURL(ctx, req.WebhookURL); err != nil {
			c.JSON(http.StatusUnprocessableEntity, invalid(c, "webhook_url", "The URL is not allowed."))
			return
		}
		webhook = &req.WebhookURL
	}
	job, err := h.d.Store.Create(ctx, c.GetString(ownerKey), webhook, req.SourceURLs)
	if err != nil {
		h.d.Logger.Error("create job", "err", err, "trace_id", trace(c))
		c.JSON(http.StatusInternalServerError, envelope.Failure("INTERNAL_ERROR", "Internal server error", trace(c), nil))
		return
	}
	c.Header("Location", "/api/v1/jobs/"+job.ID)
	c.JSON(http.StatusAccepted, envelope.OK(job, trace(c)))
}

func (h *handlers) getJob(c *gin.Context) {
	job, err := h.d.Store.Get(c.Request.Context(), c.GetString(ownerKey), c.Param("id"))
	if errors.Is(err, jobs.ErrNotFound) {
		c.JSON(http.StatusNotFound, envelope.Failure("NOT_FOUND", "Not found.", trace(c), nil))
		return
	}
	if err != nil {
		h.d.Logger.Error("get job", "err", err, "trace_id", trace(c))
		c.JSON(http.StatusInternalServerError, envelope.Failure("INTERNAL_ERROR", "Internal server error", trace(c), nil))
		return
	}
	c.JSON(http.StatusOK, envelope.OK(job, trace(c)))
}

func apiKeyAuth(keys []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		for _, k := range keys {
			if tok != "" && subtle.ConstantTimeCompare([]byte(tok), []byte(k)) == 1 {
				sum := sha256.Sum256([]byte(k))
				c.Set(ownerKey, "key:"+hex.EncodeToString(sum[:8]))
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, envelope.Failure("UNAUTHORIZED", "Missing or invalid API key.", trace(c), nil))
	}
}

func invalid(c *gin.Context, field, message string) envelope.Envelope {
	return envelope.Failure("VALIDATION_ERROR", "The request is invalid.", trace(c),
		map[string]any{"details": []map[string]string{{"field": field, "message": message}}})
}

func trace(c *gin.Context) string { return c.GetString(traceKey) }

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" || len(id) > 128 {
			id = newID()
		}
		c.Set(traceKey, id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func accessLog(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		l.Info("request", "method", c.Request.Method, "path", c.FullPath(), "status", c.Writer.Status(),
			"dur_ms", time.Since(start).Milliseconds(), "trace_id", trace(c))
	}
}

func recovery(l *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		l.Error("unhandled error", "panic", recovered, "trace_id", trace(c))
		c.AbortWithStatusJSON(http.StatusInternalServerError, envelope.Failure("INTERNAL_ERROR", "Internal server error", trace(c), nil))
	})
}
