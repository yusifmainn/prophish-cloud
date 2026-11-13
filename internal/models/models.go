package models

import (
	"math"
	"time"
)

type User struct {
	ID          string
	Email       string
	Role        string
	IsActive    bool
	CreatedAt   time.Time
	LastLoginAt *time.Time
}

type Client struct {
	ID           string
	Name         string
	ContactEmail string
	Notes        string
	CreatedAt    time.Time
}

type Authorization struct {
	ID           string
	ClientID     string
	Scope        string
	AuthorizedBy string
	DocumentRef  string
	ValidFrom    time.Time
	ValidTo      time.Time
	CreatedAt    time.Time
}

type Group struct {
	ID        string
	ClientID  string
	Name      string
	CreatedAt time.Time
	Targets   int
}

type Target struct {
	ID        string
	GroupID   string
	Email     string
	FirstName string
	LastName  string
	Position  string
	CreatedAt time.Time
}

type EmailTemplate struct {
	ID          string
	ClientID    *string
	Name        string
	Subject     string
	HTMLBody    string
	TextBody    string
	Headers     map[string]string
	Attachments []Attachment
	CreatedAt   time.Time
}

type Attachment struct {
	ID          string
	TemplateID  string
	Filename    string
	ContentType string
	Content     []byte
}

type LandingPage struct {
	ID          string
	ClientID    *string
	Name        string
	HTMLBody    string
	RedirectURL string
	CreatedAt   time.Time
}

type SendingProfile struct {
	ID          string
	Name        string
	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	FromAddress string
	HasPassword bool
	Headers     map[string]string
	CreatedAt   time.Time
}

type Campaign struct {
	ID               string
	ClientID         string
	AuthorizationID  string
	Name             string
	TemplateID       string
	LandingPageID    string
	SendingProfileID string
	Status           string
	// Kind is the delivery style: "local" (a hosted landing page is served on
	// click, the classic flow) or "public" (a click is tracked and then
	// redirected to RedirectURL, a real external site — no landing page, no form).
	Kind string
	// RedirectURL is the external destination a "public" campaign redirects a
	// click to. Empty for "local" campaigns.
	RedirectURL string
	LaunchAt    *time.Time
	SendBy      *time.Time
	CompletedAt *time.Time
	Archived    bool
	CreatedAt   time.Time
	// PublicBaseURL overrides the global public base URL for this campaign's
	// tracking links when non-empty (empty = use the global default).
	PublicBaseURL string
	// TunnelManaged marks campaigns whose public URL comes from an app-managed
	// per-campaign cloudflared quick tunnel (restarted on boot/exit).
	TunnelManaged bool
}

type CampaignResult struct {
	ID          string
	CampaignID  string
	TargetID    string
	ResultToken string
	Status      string
	SentAt      *time.Time
}

// ResultStats is an aggregated view for reporting.
type ResultStats struct {
	Total     int
	Sent      int
	Opened    int
	Clicked   int
	Submitted int
	Reported  int
}

// Pct returns n as a percentage of total, rounded to the nearest whole percent.
// A nonzero n never renders as 0% (it floors at 1%), so a single real event is
// never shown as "0%". total <= 0 yields 0. Shared by the screen and the PDF.
func Pct(n, total int) int {
	if total <= 0 {
		return 0
	}
	p := int(math.Round(float64(n) / float64(total) * 100.0))
	if p == 0 && n > 0 {
		p = 1
	}
	return p
}

// RiskLevel is the single source of truth for a campaign's risk band, shared by
// the on-screen report and the PDF. It takes the submitted and clicked rates as
// whole percentages and returns a label plus a status class (ok|warn|crit).
func RiskLevel(submittedPct, clickedPct int) (label, class string) {
	switch {
	case submittedPct >= 15:
		return "Yüksək risk", "crit"
	case submittedPct >= 5:
		return "Orta risk", "warn"
	case clickedPct >= 25:
		return "Orta risk", "warn"
	default:
		return "Aşağı risk", "ok"
	}
}
