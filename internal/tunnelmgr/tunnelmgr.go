// Package tunnelmgr manages per-campaign Cloudflare quick tunnels: each managed
// campaign gets its own `cloudflared tunnel --url http://localhost:8080`
// process, its trycloudflare URL is scraped from the process output and written
// to campaigns.public_base_url, and the process is supervised (restarted with
// backoff on exit — every restart yields a NEW URL, which is re-saved).
//
// Quick tunnels are a best-effort free service: no documented hard limit, but
// we cap concurrency anyway (default 25). Existing links die when a tunnel
// restarts — that is inherent to trycloudflare and surfaced in the UI.
package tunnelmgr

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os/exec"
	"regexp"
	"sync"
	"sync/atomic"
	"time"

	"prophish/internal/store"
)

// ErrCap is returned when the concurrent-tunnel cap is reached.
var ErrCap = errors.New("tunnel cap reached")

// Restart backoff. A run that provisioned a URL (the tunnel worked and then
// dropped) reconnects quickly. A run that never got a URL means provisioning
// failed — a missing/incompatible binary, no egress, or (most commonly)
// trycloudflare rate-limiting us with HTTP 429. Retrying such a failure quickly
// only deepens the rate limit, so we back off hard and grow from there.
const (
	minBackoff     = time.Second
	maxBackoff     = time.Minute
	failBackoff    = 10 * time.Minute
	maxFailBackoff = 30 * time.Minute
)

// nextBackoff returns how long to wait before the next restart, given the
// current backoff and whether the just-finished run obtained a tunnel URL.
func nextBackoff(current time.Duration, gotURL bool) time.Duration {
	if gotURL {
		return minBackoff
	}
	if current < failBackoff {
		return failBackoff
	}
	if next := current * 2; next < maxFailBackoff {
		return next
	}
	return maxFailBackoff
}

var urlRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// ExtractURL returns the first trycloudflare quick-tunnel URL in a log line, or
// "". cloudflared prints https://api.trycloudflare.com in its control-plane and
// in failure messages ("failed to request quick Tunnel: Post ...api.trycloudflare.com");
// that is the API endpoint, not an assigned tunnel hostname, so it must never be
// saved as a campaign's public URL.
func ExtractURL(line string) string {
	u := urlRe.FindString(line)
	if u == "https://api.trycloudflare.com" {
		return ""
	}
	return u
}

type proc struct {
	cmd    *exec.Cmd
	stopCh chan struct{}
}

// Manager owns campaignID → cloudflared process.
type Manager struct {
	Bin  string   // cloudflared binary (overridable in tests)
	Args []string // tunnel args (overridable in tests)
	Max  int      // concurrent tunnel cap
	// Save persists a freshly-observed tunnel URL for the campaign.
	Save func(ctx context.Context, campaignID, url string) error

	mu    sync.Mutex
	procs map[string]*proc
}

// New builds a Manager bound to the store and the in-container public server.
func New(st *store.Store) *Manager {
	return &Manager{
		Bin:   "cloudflared",
		Args:  []string{"tunnel", "--no-autoupdate", "--url", "http://localhost:8080"},
		Max:   25,
		Save:  func(c context.Context, id, url string) error { return st.SetCampaignTunnel(c, id, url) },
		procs: map[string]*proc{},
	}
}

// NewTest builds a Manager with a custom binary/save for unit tests.
func NewTest(bin string, args []string, max int, save func(context.Context, string, string) error) *Manager {
	return &Manager{Bin: bin, Args: args, Max: max, Save: save, procs: map[string]*proc{}}
}

// Start spawns (idempotently) and supervises a tunnel for the campaign.
func (m *Manager) Start(campaignID string) error {
	m.mu.Lock()
	if _, ok := m.procs[campaignID]; ok {
		m.mu.Unlock()
		return nil
	}
	if len(m.procs) >= m.Max {
		m.mu.Unlock()
		return ErrCap
	}
	pr := &proc{stopCh: make(chan struct{})}
	m.procs[campaignID] = pr
	m.mu.Unlock()
	go m.supervise(campaignID, pr)
	return nil
}

func stopped(pr *proc) bool {
	select {
	case <-pr.stopCh:
		return true
	default:
		return false
	}
}

func (m *Manager) supervise(id string, pr *proc) {
	backoff := minBackoff
	for {
		if stopped(pr) {
			return
		}
		var gotURL atomic.Bool
		cmd := exec.Command(m.Bin, m.Args...)
		out, err := cmd.StdoutPipe()
		if err == nil {
			cmd.Stderr = cmd.Stdout
			err = cmd.Start()
		}
		if err != nil {
			log.Printf("tunnelmgr: campaign %s: spawn failed: %v", id, err)
		} else {
			m.mu.Lock()
			pr.cmd = cmd
			m.mu.Unlock()
			if stopped(pr) {
				// Stop landed while the process was spawning — kill it now or it leaks.
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
				return
			}
			done := make(chan struct{})
			go func() { m.scan(id, out, &gotURL); close(done) }()
			if werr := cmd.Wait(); werr != nil {
				log.Printf("tunnelmgr: campaign %s: tunnel exited: %v", id, werr)
			}
			<-done // let scan drain the pipe so gotURL reflects this run
		}
		if stopped(pr) {
			return
		}
		backoff = nextBackoff(backoff, gotURL.Load())
		if gotURL.Load() {
			log.Printf("tunnelmgr: campaign %s: restarting tunnel in %s", id, backoff)
		} else {
			// Most often trycloudflare rate-limiting (HTTP 429); hammering makes it
			// worse, so wait a long time before trying again.
			log.Printf("tunnelmgr: campaign %s: tunnel did not provision (rate limit or error) — backing off %s", id, backoff)
		}
		select {
		case <-pr.stopCh:
			return
		case <-time.After(backoff):
		}
	}
}

// scan reads cloudflared output and saves every trycloudflare URL observed
// (a supervised restart prints a new one, which replaces the old in the DB). It
// sets gotURL when a tunnel hostname is seen, so the supervisor can tell a
// working-then-dropped tunnel from one that never provisioned.
func (m *Manager) scan(id string, r io.Reader, gotURL *atomic.Bool) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		if u := ExtractURL(sc.Text()); u != "" {
			gotURL.Store(true)
			if err := m.Save(context.Background(), id, u); err != nil {
				log.Printf("tunnelmgr: campaign %s: save url: %v", id, err)
			} else {
				log.Printf("tunnelmgr: campaign %s tunnel ready: %s", id, u)
			}
		}
	}
}

// Stop kills the campaign's tunnel and stops supervising it.
func (m *Manager) Stop(campaignID string) {
	m.mu.Lock()
	pr, ok := m.procs[campaignID]
	if ok {
		delete(m.procs, campaignID)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	close(pr.stopCh)
	m.mu.Lock()
	cmd := pr.cmd
	m.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// StopAll kills every tunnel (called on app shutdown).
func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.procs))
	for id := range m.procs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.Stop(id)
	}
}
