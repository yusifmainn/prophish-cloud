package publicsrv

import (
	"net/http"
	"sync"
	"time"
)

// Per-IP token-bucket rate limiter for the public listener. Bounds how fast a
// single source can hit the token-lookup endpoints and reach the database.
const (
	rlRate    = 10.0 // tokens per second refilled
	rlBurst   = 30.0 // max tokens
	rlIdleTTL = 10 * time.Minute
)

type bucket struct {
	tokens float64
	last   time.Time
}

var (
	rlMu      sync.Mutex
	rlBuckets = map[string]*bucket{}
	rlLastGC  time.Time
)

func (s *Server) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Key on the real client identity (spoof-resistant: XFF only honored from
		// trusted proxies) so a single client behind the proxy can't share — or
		// monopolize — one global bucket.
		if !allow(s.clientIP(r)) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allow(key string) bool {
	now := time.Now()
	rlMu.Lock()
	defer rlMu.Unlock()

	if now.Sub(rlLastGC) > rlIdleTTL {
		for k, b := range rlBuckets {
			if now.Sub(b.last) > rlIdleTTL {
				delete(rlBuckets, k)
			}
		}
		rlLastGC = now
	}

	b := rlBuckets[key]
	if b == nil {
		b = &bucket{tokens: rlBurst, last: now}
		rlBuckets[key] = b
	}
	// Refill.
	b.tokens += now.Sub(b.last).Seconds() * rlRate
	if b.tokens > rlBurst {
		b.tokens = rlBurst
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
