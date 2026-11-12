package tunnelmgr

import (
	"context"
	"sync"
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
