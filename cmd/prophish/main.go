package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	"prophish/internal/publicsrv"
	"prophish/internal/server"
	"prophish/internal/store"
)

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

	launcher := engine.NewLauncher(st, cfg.EncryptionKey, cfg.PublicBaseURL)

	// Background workers: scheduled-campaign launcher and the IMAP reported-email monitor.
	bgCtx, bgCancel := context.WithCancel(context.Background())
	defer bgCancel()
	go launcher.RunScheduler(bgCtx)
	go imappoll.New(st, cfg.EncryptionKey).Run(bgCtx)

	srv, err := server.NewServer(cfg, st, sessions, launcher)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = adminServer.Shutdown(shutdownCtx)
	_ = publicServer.Shutdown(shutdownCtx)
}
