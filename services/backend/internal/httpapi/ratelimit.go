package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"app/internal/envelope"
)

// limiter is a token bucket per API key: it refills perMinute tokens each minute up to burst, and every request spends
// one. State is per process, so with several replicas the effective limit is the sum across them.
type limiter struct {
	mu      sync.Mutex
	perSec  float64
	burst   float64
	now     func() time.Time
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newLimiter(perMinute int, now func() time.Time) *limiter {
	if perMinute <= 0 {
		return nil
	}
	if now == nil {
		now = time.Now
	}
	burst := math.Max(5, float64(perMinute)/4)
	return &limiter{perSec: float64(perMinute) / 60, burst: burst, now: now, buckets: map[string]*bucket{}}
}

// allow spends a token for key. When none is left it reports how long until one is.
func (l *limiter) allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b := l.buckets[key]
	if b == nil {
		if len(l.buckets) > 10_000 { // keep memory bounded: forget keys that have been idle long enough to be full again
			l.evict(now)
		}
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.perSec)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / l.perSec * float64(time.Second))
}

func (l *limiter) evict(now time.Time) {
	idle := time.Duration(l.burst/l.perSec*float64(time.Second)) + time.Minute
	for k, b := range l.buckets {
		if now.Sub(b.last) > idle {
			delete(l.buckets, k)
		}
	}
}

// rateLimit rejects callers who exceed their key's rate with 429 and a Retry-After header. It must run after auth.
func rateLimit(l *limiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil {
			c.Next()
			return
		}
		ok, wait := l.allow(c.GetString(ownerKey))
		if ok {
			c.Next()
			return
		}
		secs := int(math.Ceil(wait.Seconds()))
		c.Header("Retry-After", strconv.Itoa(secs))
		c.AbortWithStatusJSON(http.StatusTooManyRequests, envelope.Failure("RATE_LIMITED",
			"Too many requests. Retry in "+strconv.Itoa(secs)+" seconds.", trace(c), map[string]any{"retry_after_seconds": secs}))
	}
}
