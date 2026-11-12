package server

import (
	"html/template"
	"net/http"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"prophish/internal/config"
	"prophish/internal/engine"
	"prophish/internal/netutil"
	"prophish/internal/store"
	"prophish/internal/tunnelmgr"
)

// Server holds shared dependencies and implements the HTTP handlers.
type Server struct {
	cfg       *config.Config
	store     *store.Store
	sessions  *scs.SessionManager
	launcher  *engine.Launcher
	tunnels   *tunnelmgr.Manager
	detector  *netutil.Detector // public-reachability check (listener recommendation)
	templates *template.Template
	previews  *previewCache
}

func NewServer(cfg *config.Config, st *store.Store, sessions *scs.SessionManager, launcher *engine.Launcher, tunnels *tunnelmgr.Manager, detector *netutil.Detector) (*Server, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, store: st, sessions: sessions, launcher: launcher, tunnels: tunnels, detector: detector, templates: tmpl, previews: &previewCache{}}, nil
}

// AdminRouter builds the authenticated admin panel router.
func (s *Server) AdminRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// NOTE: middleware.RealIP is intentionally NOT used here. The admin panel is
	// reached directly (over a VPN), so r.RemoteAddr is the true peer; trusting a
	// client-supplied X-Forwarded-For would let an attacker spoof the IP used for
	// the login brute-force lockout.
	r.Use(middleware.Recoverer)
	r.Use(secureHeaders)
	r.Use(s.sessions.LoadAndSave)

	// Public (unauthenticated) routes. Login/setup get CSRF origin checks too.
	r.Get("/healthz", s.handleHealth)
	// Language switch: sets the UI-language cookie and redirects back. Public so
	// it also works from the login/setup screens; safe-redirect enforced in the
	// handler (same-origin relative path only).
	r.Get("/lang/{code}", s.handleSetLang)
	r.Get("/login", s.handleLoginForm)
	r.With(s.checkOrigin).Post("/login", s.handleLoginSubmit)
	r.Get("/setup", s.handleSetupForm)
	r.With(s.checkOrigin).Post("/setup", s.handleSetupSubmit)
	r.Get("/static/app.js", s.handleStaticJS)
	r.Get("/static/editor.js", s.handleStaticEditorJS)
	r.Get("/static/builder.js", s.handleStaticBuilderJS)
	r.Get("/static/shahdag.json", s.handleStaticShahdag)
	r.Get("/static/auth.js", s.handleStaticAuthJS)
	r.Get("/static/hero/{name}", s.handleStaticHero)
	r.Get("/static/logo-mark.svg", s.handleStaticLogo)
	r.Get("/static/logo-mark-dark-bg.svg", s.handleStaticLogo)
	r.Get("/favicon.ico", s.handleBrandIcon)
	r.Get("/favicon.svg", s.handleBrandIcon)
	r.Get("/apple-touch-icon.png", s.handleBrandIcon)
	r.Get("/static/fonts/{name}", s.handleStaticFont)
	r.Get("/static/docs/{lang}/{name}", s.handleStaticDoc)

	// Authenticated routes. checkOrigin adds CSRF defense-in-depth on top of the
	// SameSite=Lax session cookie; adminOnly gates sensitive configuration.
	adminOnly := s.requireAdmin
	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Use(s.checkOrigin)
		r.Get("/", s.handleDashboard)
		r.Get("/documentation", s.handleDocumentation)
		r.Post("/logout", s.handleLogout)

		r.Get("/clients", s.handleClientsList)
		r.Post("/clients", s.handleClientsCreate)
		r.Get("/clients/{id}", s.handleClientDetail)
		r.Post("/clients/{id}/edit", s.handleClientUpdate)
		// Deleting a client cascades to all its data → admin-only + confirmed.
		r.With(adminOnly).Post("/clients/{id}/delete", s.handleClientDelete)
		// Authorization/RoE creation is admin-only (separation of duties: an
		// operator must not be able to self-authorize a campaign).
		r.With(adminOnly).Post("/clients/{id}/authorizations", s.handleAuthCreate)
		r.With(adminOnly).Post("/authorizations/{aid}/edit", s.handleAuthUpdate)
		r.With(adminOnly).Post("/authorizations/{aid}/delete", s.handleAuthDelete)
		r.Post("/clients/{id}/groups", s.handleGroupCreate)

		r.Get("/groups/{id}", s.handleGroupDetail)
		r.Post("/groups/{id}/targets", s.handleTargetAdd)
		r.Post("/groups/{id}/import", s.handleTargetImport)
		r.Post("/groups/{id}/rename", s.handleGroupUpdate)
		r.With(adminOnly).Post("/groups/{id}/delete", s.handleGroupDelete)
		r.Post("/targets/{tid}/edit", s.handleTargetUpdate)
		r.Post("/targets/{tid}/delete", s.handleTargetDelete)

		r.Get("/templates", s.handleTemplatesList)
		r.Post("/templates", s.handleTemplateCreate)
		r.Post("/templates/import", s.handleTemplateImport)
		r.Get("/templates/{id}", s.handleTemplateDetail)
		r.Get("/templates/{id}/preview", s.handleTemplatePreview)
		r.Post("/templates/{id}/edit", s.handleTemplateUpdate)
		r.With(adminOnly).Post("/templates/{id}/delete", s.handleTemplateDelete)
		r.Post("/templates/{id}/attachments", s.handleAttachmentUpload)
		r.Post("/templates/{id}/attachments/{aid}/delete", s.handleAttachmentDelete)

		r.Get("/pages", s.handlePagesList)
		r.Post("/pages", s.handlePageCreate)
		r.Post("/pages/import", s.handlePageImport)
		r.Get("/pages/builder", s.handlePageBuilderNew)
		r.Post("/pages/builder/preview", s.handlePageBuilderPreview)
		r.Get("/pages/builder/preview/{token}", s.handlePageBuilderPreviewShell)
		r.Get("/pages/builder/preview/{token}/raw", s.handlePageBuilderPreviewRaw)
		r.Get("/pages/{id}", s.handlePageDetail)
		r.Get("/pages/{id}/builder", s.handlePageBuilderEdit)
		r.Get("/pages/{id}/preview", s.handlePagePreview)
		r.Get("/pages/{id}/preview/devices", s.handlePageDevices)
		r.Post("/pages/{id}/edit", s.handlePageUpdate)
		r.With(adminOnly).Post("/pages/{id}/delete", s.handlePageDelete)

		// Built-in library of ready-made brand-faithful simulation kits.
		r.Get("/library", s.handleLibraryList)
		r.Get("/library/{id}/preview/page", s.handleLibraryPagePreview)
		r.Get("/library/{id}/preview/email", s.handleLibraryEmailPreview)
		r.Post("/library/{id}/use", s.handleLibraryUse)

		// Sending profiles hold SMTP credentials and reach external hosts → admin-only.
		r.Get("/profiles", s.handleProfilesList)
		r.With(adminOnly).Post("/profiles", s.handleProfileCreate)
		r.With(adminOnly).Post("/profiles/{id}/edit", s.handleProfileUpdate)
		r.With(adminOnly).Post("/profiles/{id}/delete", s.handleProfileDelete)
		r.With(adminOnly).Post("/profiles/{id}/test", s.handleProfileTest)

		r.Get("/campaigns", s.handleCampaignsList)
		r.Post("/campaigns", s.handleCampaignCreate)
		r.Post("/campaigns/bulk-delete", s.handleCampaignsBulkDelete)
		r.Get("/campaigns/{id}", s.handleCampaignDetail)
		r.Post("/campaigns/{id}/rename", s.handleCampaignRename)
		r.Get("/campaigns/{id}/edit", s.handleCampaignEditForm)
		r.Post("/campaigns/{id}/edit", s.handleCampaignUpdate)
		r.Get("/campaigns/{id}/export.csv", s.handleCampaignExport)
		r.Get("/campaigns/{id}/report.pdf", s.handleCampaignReportPDF)
		r.Post("/campaigns/{id}/launch", s.handleCampaignLaunch)
		r.Post("/campaigns/{id}/schedule", s.handleCampaignSchedule)
		r.Post("/campaigns/{id}/cancel", s.handleCampaignCancel)
		r.Post("/campaigns/{id}/resend", s.handleCampaignResend)
		r.Post("/campaigns/{id}/archive", s.handleCampaignArchive)
		r.With(adminOnly).Post("/campaigns/{id}/delete", s.handleCampaignDelete)

		r.Get("/users", s.handleUsersList)
		r.With(adminOnly).Post("/users", s.handleUserCreate)
		r.With(adminOnly).Post("/users/{id}/delete", s.handleUserDelete)
		r.Post("/account/password", s.handleAccountPassword)
		r.Post("/account/apikey", s.handleAccountAPIKey)

		// IMAP config holds mailbox credentials → admin-only.
		r.With(adminOnly).Get("/settings/imap", s.handleIMAPForm)
		r.With(adminOnly).Post("/settings/imap", s.handleIMAPSave)

		// Audit trail of operator actions → admin-only.
		r.With(adminOnly).Get("/audit", s.handleAuditList)
	})

	// JSON REST API (authenticated by operator API keys, not sessions).
	r.Mount("/api", s.apiRouter())

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}
