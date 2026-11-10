package server

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// In-memory login throttle: after too many failed attempts for an (email, IP)
// key, further attempts are rejected for a growing lockout window. This blunts
// online brute-force / credential-stuffing without an external dependency.
const (
	loginMaxFails        = 5  // per (email, source IP)
	loginAccountMaxFails = 50 // per account across all IPs (bounds distributed guessing)
	loginBaseLock        = time.Minute
	loginMaxLock         = 15 * time.Minute
	loginDecayAfter      = 30 * time.Minute
)

type attemptRecord struct {
	fails    int
	lockedAt time.Time
	seen     time.Time
}

var (
	loginMu       sync.Mutex
	loginAttempts = map[string]*attemptRecord{}
)

func loginKey(r *http.Request, email string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return strings.ToLower(email) + "|" + host
}

// loginLockRemaining returns how long the key is locked for (0 if not locked).
// maxFails is the threshold at which the key starts locking, so a per-(email,IP)
// key can lock quickly while a per-account key tolerates more before it does.
func loginLockRemaining(key string, maxFails int) time.Duration {
	loginMu.Lock()
	defer loginMu.Unlock()
	rec := loginAttempts[key]
	if rec == nil {
		return 0
	}
	if time.Since(rec.seen) > loginDecayAfter {
		delete(loginAttempts, key)
		return 0
	}
	if rec.fails < maxFails {
		return 0
	}
	lock := loginBaseLock << (rec.fails - maxFails)
	if lock > loginMaxLock || lock <= 0 {
		lock = loginMaxLock
	}
	rem := time.Until(rec.lockedAt.Add(lock))
	if rem < 0 {
		return 0
	}
	return rem
}

func loginFail(key string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	rec := loginAttempts[key]
	if rec == nil {
		rec = &attemptRecord{}
		loginAttempts[key] = rec
	}
	rec.fails++
	rec.lockedAt = time.Now()
	rec.seen = time.Now()
}

func loginReset(key string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginAttempts, key)
}
