package server

import (
	"bytes"
	"testing"
	"time"

	"prophish/internal/models"
	"prophish/internal/store"
)

// TestTemplatesRender parses every embedded template and executes it with
// representative data, so a bad field reference or range fails the build/tests
// instead of surfacing only at runtime.
func TestTemplatesRender(t *testing.T) {
	tmpl, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}

	now := time.Now()
	cid := "client-1"
	cases := map[string]any{
		"setup.html": map[string]any{"Error": "x"},
		"login.html": map[string]any{"Error": "x"},
		"dashboard.html": map[string]any{
			"Stats":  store.DashboardStats{ActiveCampaigns: 1, Campaigns: 4, Tested: 120, ClickRate: 38, ReportRate: 29},
			"Recent": []store.CampaignRow{{ID: "c1", Name: "C", ClientName: "ACME", Status: "completed", CreatedAt: now}},
		},
		"clients.html": map[string]any{"Clients": []models.Client{
			{ID: cid, Name: "ACME", ContactEmail: "a@acme.az", CreatedAt: now},
		}},
		"client_detail.html": map[string]any{
			"Client":         &models.Client{ID: cid, Name: "ACME", ContactEmail: "a@acme.az", Notes: "n"},
			"Authorizations": []models.Authorization{{Scope: "*.acme.az", AuthorizedBy: "CTO", DocumentRef: "RoE-1", ValidFrom: now, ValidTo: now}},
			"Groups":         []models.Group{{ID: "g1", Name: "Finance", Targets: 5}},
			"Today":          now.Format("2006-01-02"),
		},
		"group_detail.html": map[string]any{
			"Group":   &models.Group{ID: "g1", ClientID: cid, Name: "Finance"},
			"Client":  &models.Client{ID: cid, Name: "ACME"},
			"Targets": []models.Target{{Email: "e@acme.az", FirstName: "E", LastName: "F", Position: "Acc"}},
		},
		"templates.html": map[string]any{"Templates": []models.EmailTemplate{{Name: "T", Subject: "S", CreatedAt: now}}},
		"pages.html":     map[string]any{"Pages": []models.LandingPage{{Name: "P", RedirectURL: "u", CreatedAt: now}}},
		"page_builder.html": map[string]any{"Edit": true,
			"P":         &models.LandingPage{ID: "p1", Name: "P", HTMLBody: "<html></html>", RedirectURL: "u"},
			"JSStrings": map[string]string{"builder.js.page": "Səhifə"}},
		"profiles.html": map[string]any{"Profiles": []models.SendingProfile{{Name: "SMTP", SMTPHost: "h", SMTPPort: 587, FromAddress: "f"}}},
		"campaigns.html": map[string]any{
			"Campaigns":      []store.CampaignRow{{ID: "c1", Name: "C", ClientName: "ACME", Status: "draft", CreatedAt: now}},
			"Clients":        []models.Client{{ID: cid, Name: "ACME"}},
			"Authorizations": []store.AuthRow{{ID: "a1", ClientName: "ACME", Scope: "*.acme.az", ValidTo: now}},
			"Groups":         []store.GroupRow{{ID: "g1", ClientName: "ACME", Name: "Finance", Targets: 5}},
			"Templates":      []models.EmailTemplate{{ID: "t1", Name: "T"}},
			"Pages":          []models.LandingPage{{ID: "p1", Name: "P"}},
			"Profiles":       []models.SendingProfile{{ID: "s1", Name: "SMTP"}},
		},
		"campaign_detail.html": map[string]any{
			"C": &store.CampaignView{ID: "c1", Name: "C", Status: "draft", ClientName: "ACME",
				TemplateName: "T", PageName: "P", ProfileName: "SMTP", AuthScope: "*.acme.az", AuthValidTo: now},
			"Camp":  &models.Campaign{ID: "c1", Name: "C", TunnelManaged: true, PublicBaseURL: "https://x.trycloudflare.com"},
			"Stats": models.ResultStats{Total: 10, Sent: 10, Opened: 6, Clicked: 3, Submitted: 1},
			"Pct":   Percentages{Sent: 100, Opened: 60, Clicked: 30, Submitted: 10},
		},
	}

	for name, data := range cases {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
			t.Errorf("execute %s: %v", name, err)
		}
		if buf.Len() == 0 {
			t.Errorf("execute %s: empty output", name)
		}
	}
}
