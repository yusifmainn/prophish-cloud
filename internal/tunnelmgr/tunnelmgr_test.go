package tunnelmgr

import (
	"context"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestExtractURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{`2026-11-11 INF +++ https://unit-abc-123.trycloudflare.com +++`, "https://unit-abc-123.trycloudflare.com"},
		{"no url here", ""},
		{`https://not-tunnel.example.com`, ""},
	}
	for _, c := range cases {
		if got := ExtractURL(c.in); got != c.want {
			t.Errorf("ExtractURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestRunNamed verifies the named tunnel is (re)spawned and stops on cancel.
func TestRunNamed(t *testing.T) {
	var starts atomic.Int32
	m := New(nil)
	m.NamedCmd = func(token string) *exec.Cmd {
		starts.Add(1)
		// A short-lived process so RunNamed loops and restarts quickly.
		return exec.Command("sh", "-c", "sleep 0.2")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { m.RunNamed(ctx, "dummy-token"); close(done) }()

	// It should spawn at least twice (restart after the first short run).
	deadline := time.Now().Add(3 * time.Second)
	for starts.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if starts.Load() < 2 {
		t.Fatalf("expected the named tunnel to restart; starts=%d", starts.Load())
	}

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("RunNamed did not return after context cancel")
	}
}

func TestNextBackoff(t *testing.T) {
	// A provisioned tunnel that dropped reconnects quickly.
	if got := nextBackoff(maxFailBackoff, true); got != minBackoff {
		t.Errorf("gotURL reconnect = %s, want %s", got, minBackoff)
	}
	// A run that never provisioned (429 / spawn error) jumps straight to the long
	// fail backoff rather than retrying within seconds.
	if got := nextBackoff(minBackoff, false); got != failBackoff {
		t.Errorf("first failure = %s, want %s", got, failBackoff)
	}
	// Repeated failures grow...
	if got := nextBackoff(failBackoff, false); got <= failBackoff {
		t.Errorf("escalation = %s, want > %s", got, failBackoff)
	}
	// ...but are capped.
	if got := nextBackoff(maxFailBackoff, false); got != maxFailBackoff {
		t.Errorf("cap = %s, want %s", got, maxFailBackoff)
	}
}

// TestStartSaveStop uses a stub binary (sh) that prints a URL and sleeps; it
// verifies spawn → URL scrape → Save callback, the concurrency cap, and Stop.
func TestStartSaveStop(t *testing.T) {
	var mu sync.Mutex
	saved := map[string]string{}
	m := NewTest("sh", []string{"-c", `echo boot; echo "https://unit-xyz.trycloudflare.com/"; sleep 30`}, 2,
		func(_ context.Context, id, url string) error {
			mu.Lock()
			saved[id] = url
			mu.Unlock()
			return nil
		})

	if err := m.Start("c1"); err != nil {
		t.Fatalf("start c1: %v", err)
	}
	if err := m.Start("c2"); err != nil {
		t.Fatalf("start c2: %v", err)
	}
	if err := m.Start("c3"); err != ErrCap {
		t.Fatalf("cap: got %v, want ErrCap", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(saved)
		mu.Unlock()
		if n == 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	mu.Lock()
	if saved["c1"] != "https://unit-xyz.trycloudflare.com" || saved["c2"] != "https://unit-xyz.trycloudflare.com" {
		t.Fatalf("saved=%v", saved)
	}
	mu.Unlock()

	m.Stop("c1")
	// After Stop the slot frees up.
	if err := m.Start("c3"); err != nil {
		t.Fatalf("start c3 after stop: %v", err)
	}
	m.StopAll()
}
