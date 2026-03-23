package httpserver

import (
	"net/http"
	"sync"
	"time"
)

// SimpleTokenBucket — простой rate limiter на основе token bucket
type SimpleTokenBucket struct {
	mu       sync.Mutex
	clients  map[string]*bucket
	capacity float64
	refill   time.Duration
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

func NewSimpleTokenBucket(capacity float64, refill time.Duration) *SimpleTokenBucket {
	return &SimpleTokenBucket{
		clients:  make(map[string]*bucket),
		capacity: capacity,
		refill:   refill,
	}
}

func (rl *SimpleTokenBucket) Allow(clientID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.clients[clientID]
	if !ok {
		b = &bucket{
			tokens:     float64(rl.capacity),
			lastRefill: time.Now(),
		}
		rl.clients[clientID] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastRefill)
	tokensToAdd := float64(elapsed) / float64(rl.refill)
	b.tokens = min(b.tokens+tokensToAdd, float64(rl.capacity))
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func RateLimitMiddleware(rl *SimpleTokenBucket) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientID := r.RemoteAddr
			if cookie, err := r.Cookie("session_token"); err == nil {
				clientID = cookie.Value
			}

			if !rl.Allow(clientID) {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limit exceeded, try again later"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
