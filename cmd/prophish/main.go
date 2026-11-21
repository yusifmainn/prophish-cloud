package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	// Embed the IANA time zone database so named zones (e.g. Asia/Baku) resolve
	// even in the minimal distroless container, which ships no system tzdata.
	_ "time/tzdata"

	"github.com/alexedwards/scs/v2"

	"prophish/internal/config"
	"prophish/internal/database"
	"prophish/internal/engine"
	"prophish/internal/imappoll"
	"prophish/internal/netutil"
	"prophish/internal/publicsrv"
	"prophish/internal/server"
	"prophish/internal/store"
	"prophish/internal/tunnelmgr"
)

// refreshInfraPrefixes fetches the announced IPv4 prefixes of the configured
// scanner ASNs and installs them as the dynamic infrastructure list, then
// repeats once a day. Runs in the background: any fetch failure leaves the
// current list untouched, so the listener never depends on RIPEstat being up.
func refreshInfraPrefixes(ctx context.Context, asns []string, cachePath string) {
	refresh := func() {
		var all []*net.IPNet
		for _, asn := range asns {
			rctx, cancel := context.WithTimeout(ctx, 30*time.Second)
			nets, err := netutil.FetchAnnouncedPrefixes(rctx, asn)
			cancel()
			if err != nil {
				log.Printf("infra: fetch AS%s prefixes: %v (keeping previous list)", asn, err)
				return
			}
			all = append(all, nets...)
		}
		publicsrv.SetDynamicInfraCIDRs(all)
		if err := netutil.SavePrefixCache(cachePath, all); err != nil {
			log.Printf("infra: cache write: %v", err)
		}
		log.Printf("infra: refreshed %d prefixes from %d ASNs", len(all), len(asns))
	}
	refresh()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refresh()
		}
	}
}

func main() {

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// Apply the configured time zone process-wide so scheduling input and
	// timestamp display use the operator's local wall-clock, not UTC.
	time.Local = cfg.Location

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	// Apply any pending schema migrations (the app owns the schema, so upgrades
	// to an existing database pick up new migrations automatically).
	if n, err := database.Migrate(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	} else if n > 0 {
		log.Printf("applied %d database migration(s)", n)
	}

	st := store.New(pool)

	// Recover campaigns left 'running' by a crash/restart: reset them to
	// 'scheduled' (launch time cleared) so an operator must deliberately
	// relaunch; a relaunch only sends to targets not already sent.
	if n, err := st.RecoverStuckCampaigns(ctx); err != nil {
		log.Printf("recover stuck campaigns: %v", err)
	} else if n > 0 {
		log.Printf("recovered %d campaign(s) interrupted by a previous shutdown", n)
	}

	sessions := scs.New()
	sessions.Lifetime = 8 * time.Hour
	sessions.IdleTimeout = 30 * time.Minute
	sessions.Cookie.Name = "prophish_session"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	sessions.Cookie.Secure = cfg.SecureCookies

	// Public-reachability detection (listener-location recommendation + engine
	// link fallback). Prewarm asynchronously so the first campaign-form load
	// hits the cache instead of blocking on the lookup.
	detector := netutil.NewDetector()
	go func() { detector.Detect(context.Background()) }()

	launcher := engine.NewLauncher(st, cfg.EncryptionKey, cfg.PublicBaseURL, func() string { return detector.Detect(context.Background()).BaseURL() })

	// Per-campaign quick tunnels: restart tunnels of managed (non-archived)
	// campaigns; each restart yields a fresh trycloudflare URL (saved to the DB
	// by the manager when cloudflared prints it).
	tunnels := tunnelmgr.New(st)
	if ids, err := st.ListTunnelManagedCampaigns(ctx); err != nil {
		log.Printf("list tunnel-managed campaigns: %v", err)
	} else {
		for _, id := range ids {
			if err := tunnels.Start(id); err != nil {
				log.Printf("restart tunnel for campaign %s: %v", id, err)
			}
		}
		if len(ids) > 0 {
			log.Printf("restarted %d campaign tunnel(s)", len(ids))
		}
	}

	// Background workers: scheduled-campaign launcher and the IMAP reported-email monitor.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	go launcher.RunScheduler(bgCtx)
	go imappoll.New(st, cfg.EncryptionKey).Run(bgCtx)

	srv, err := server.NewServer(cfg, st, sessions, launcher, tunnels, detector)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	// Scanner-classification windows are deployment-dependent (fast internal
	// delivery puts real opens inside the long external defaults); apply the
	// operator's per-route overrides before the public listener starts serving.
	if cfg.ScannerWinInternal.Grace > 0 || cfg.ScannerWinInternal.Pair > 0 ||
		cfg.ScannerWinExternal.Grace > 0 || cfg.ScannerWinExternal.Pair > 0 {
		publicsrv.SetScannerWindows(
			publicsrv.ScannerWindows(cfg.ScannerWinInternal),
			publicsrv.ScannerWindows(cfg.ScannerWinExternal),
		)
	}
	if len(cfg.InfraCIDRs) > 0 {
		publicsrv.AddInfraCIDRs(cfg.InfraCIDRs)
	}

	// Trusted-mode infrastructure list: a fresh cache (24h) loads instantly at
	// startup; a background refresh then keeps it in sync with what the
	// configured scanner ASNs (default Microsoft) actually announce — cloud
	// ranges change constantly, so a fixed list goes stale within weeks. Any
	// failure keeps the previous list (or the built-in starter set).
	infraCachePath := filepath.Join(os.TempDir(), "prophish-infra-prefixes.json")
	if nets, ok := netutil.LoadPrefixCache(infraCachePath, 24*time.Hour); ok {
		publicsrv.SetDynamicInfraCIDRs(nets)
		log.Printf("infra: loaded %d cached prefixes", len(nets))
	}
	go refreshInfraPrefixes(bgCtx, cfg.InfraASNs, infraCachePath)

	adminServer := &http.Server{
		Addr:              cfg.AdminAddr,
		Handler:           srv.AdminRouter(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	publicServer := &http.Server{
		Addr:              cfg.PublicAddr,
		Handler:           publicsrv.NewServer(st, cfg.PublicBaseURL, cfg.TrustedProxies).Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       20 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("PROPhish admin panel listening on http://%s", cfg.AdminAddr)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("admin listen: %v", err)
		}
	}()

	go func() {
		log.Printf("PROPhish public server listening on http://%s", cfg.PublicAddr)
		if err := publicServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("public listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	tunnels.StopAll()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = adminServer.Shutdown(shutdownCtx)
	_ = publicServer.Shutdown(shutdownCtx)
}
