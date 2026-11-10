package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

// apiRouter exposes a JSON REST API authenticated by operator API keys.
func (s *Server) apiRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(secureHeaders)
	r.Use(s.apiAuth)

	r.Get("/clients", s.apiClientsList)
	r.Post("/clients", s.apiClientsCreate)
	r.Get("/campaigns", s.apiCampaignsList)
	r.Get("/campaigns/{id}", s.apiCampaignGet)
	r.Get("/templates", s.apiTemplatesList)
	r.Get("/pages", s.apiPagesList)
	r.Get("/profiles", s.apiProfilesList)
	return r
}

// apiAuth authenticates via "Authorization: Bearer <key>" or "X-API-Key: <key>".
func (s *Server) apiAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				key = strings.TrimPrefix(h, "Bearer ")
			}
		}
		if key == "" {
			apiError(w, http.StatusUnauthorized, "missing API key")
			return
		}
		u, err := s.store.UserByAPIKey(r.Context(), key)
		if err != nil || u == nil {
			apiError(w, http.StatusUnauthorized, "invalid API key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func apiJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func apiError(w http.ResponseWriter, status int, msg string) {
	apiJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) apiClientsList(w http.ResponseWriter, r *http.Request) {
	clients, err := s.store.ListClients(r.Context())
	if err != nil {
		apiError(w, 500, "db error")
		return
	}
	apiJSON(w, 200, clients)
}

func (s *Server) apiClientsCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name         string `json:"name"`
		ContactEmail string `json:"contact_email"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || strings.TrimSpace(in.Name) == "" {
		apiError(w, 400, "name is required")
		return
	}
	id, err := s.store.CreateClient(r.Context(), in.Name, in.ContactEmail, in.Notes)
	if err != nil {
		apiError(w, 500, "db error")
		return
	}
	apiJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) apiCampaignsList(w http.ResponseWriter, r *http.Request) {
	camps, err := s.store.ListCampaigns(r.Context(), true)
	if err != nil {
		apiError(w, 500, "db error")
		return
	}
	apiJSON(w, 200, camps)
}

func (s *Server) apiCampaignGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	view, err := s.store.CampaignView(r.Context(), id)
	if err != nil {
		apiError(w, 404, "not found")
		return
	}
	stats, _ := s.store.CampaignStats(r.Context(), id)
	results, _ := s.store.ListResultsDetailed(r.Context(), id)
	apiJSON(w, 200, map[string]any{"campaign": view, "stats": stats, "results": results})
}

func (s *Server) apiTemplatesList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListTemplates(r.Context())
	if err != nil {
		apiError(w, 500, "db error")
		return
	}
	apiJSON(w, 200, items)
}

func (s *Server) apiPagesList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListLandingPages(r.Context())
	if err != nil {
		apiError(w, 500, "db error")
		return
	}
	apiJSON(w, 200, items)
}

func (s *Server) apiProfilesList(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSendingProfiles(r.Context())
	if err != nil {
		apiError(w, 500, "db error")
		return
	}
	apiJSON(w, 200, items)
}
