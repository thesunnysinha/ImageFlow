// Package httpapi wires the Gin HTTP layer.
package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/thesunnysinha/imageflow/apps/api/internal/jobs"
)

type Validator func(ctx context.Context, rawURL string) error

type Deps struct {
	Store          jobs.Store
	APIKeys        []string
	MaxItemsPerJob int
	ValidateURL    Validator
	Logger         *slog.Logger
	Production     bool
}

const ownerKey = "owner"

func NewRouter(d Deps) *gin.Engine {
	if d.Production {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery(), requestLog(d.Logger), securityHeaders())
	r.MaxMultipartMemory = 8 << 20

	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := d.Store.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := r.Group("/v1", apiKeyAuth(d.APIKeys))
	h := &handlers{d: d}
	v1.POST("/jobs", h.createJob)
	v1.GET("/jobs/:id", h.getJob)
	return r
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store")
		c.Next()
	}
}

func requestLog(l *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if l != nil {
			l.Info("request", "method", c.Request.Method, "path", c.FullPath(),
				"status", c.Writer.Status(), "dur_ms", time.Since(start).Milliseconds())
		}
	}
}

// apiKeyAuth is the phase-1 authenticator. It is replaced by OIDC/JWT
// validation once accounts exist; the owner identity is derived from the key.
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
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}
