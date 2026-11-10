package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	stats, _ := s.store.DashboardStats(r.Context())
	recent, _ := s.store.ListCampaigns(r.Context(), false)
	if len(recent) > 6 {
		recent = recent[:6]
	}
	s.render(w, r, http.StatusOK, "dashboard.html", map[string]any{
		"Stats":  stats,
		"Recent": recent,
	})
}

func (s *Server) handleClientsList(w http.ResponseWriter, r *http.Request) {
	clients, err := s.store.ListClients(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, http.StatusOK, "clients.html", map[string]any{"Clients": clients})
}

func (s *Server) handleClientsCreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/clients", http.StatusSeeOther)
		return
	}
	newID, err := s.store.CreateClient(r.Context(),
		name, strings.TrimSpace(r.FormValue("contact_email")), strings.TrimSpace(r.FormValue("notes")))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "client.create", "client", newID, name)
	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}
