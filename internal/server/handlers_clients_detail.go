package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"prophish/internal/store"
)

func (s *Server) handleClientDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	client, err := s.store.GetClient(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	auths, _ := s.store.ListAuthorizations(r.Context(), id)
	groups, _ := s.store.ListGroups(r.Context(), id)
	s.render(w, r, http.StatusOK, "client_detail.html", map[string]any{
		"Client":         client,
		"Authorizations": auths,
		"Groups":         groups,
		"Today":          time.Now().Format("2006-01-02"),
	})
}

func (s *Server) handleClientUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/clients/"+id, http.StatusSeeOther)
		return
	}
	if err := s.store.UpdateClient(r.Context(), id, name,
		strings.TrimSpace(r.FormValue("contact_email")), strings.TrimSpace(r.FormValue("notes"))); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "client.update", "client", id, name)
	s.flashOK(r, "Müştəri yadda saxlanıldı.")
	http.Redirect(w, r, "/clients/"+id, http.StatusSeeOther)
}

// handleClientDelete removes a client and everything linked to it (cascade).
// Admin-only; the UI confirms the irreversible cascade before posting.
func (s *Server) handleClientDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := id
	if cl, err := s.store.GetClient(r.Context(), id); err == nil && cl != nil {
		name = cl.Name
	}
	if err := s.store.DeleteClient(r.Context(), id); err != nil {
		if errors.Is(err, store.ErrClientReferenced) {
			// A client-scoped template/page is still referenced by another
			// client's campaign — refuse cleanly instead of a raw 500.
			s.flashErr(r, "Müştəri silinmədi — ona aid şablon və ya səhifəyə başqa müştərinin kampaniyası istinad edir. Əvvəlcə həmin kampaniyaları silin.")
			http.Redirect(w, r, "/clients/"+id, http.StatusSeeOther)
			return
		}
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "client.delete", "client", id, name)
	http.Redirect(w, r, "/clients", http.StatusSeeOther)
}

func (s *Server) handleAuthCreate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	scope := strings.TrimSpace(r.FormValue("scope"))
	by := strings.TrimSpace(r.FormValue("authorized_by"))
	docRef := strings.TrimSpace(r.FormValue("document_ref"))
	from, err1 := parseDay(r.FormValue("valid_from"), false)
	to, err2 := parseDay(r.FormValue("valid_to"), true)
	if scope == "" || by == "" || err1 != nil || err2 != nil || !to.After(from) {
		s.flashErr(r, "İcazə əlavə edilmədi — sahələri yoxlayın; son tarix başlanğıcdan sonra olmalıdır.")
		http.Redirect(w, r, "/clients/"+id+"#auth", http.StatusSeeOther)
		return
	}
	authID, err := s.store.CreateAuthorization(r.Context(), id, scope, by, docRef, from, to)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "authorization.create", "authorization", authID,
		"client "+id+", by "+by+", valid "+from.Format("2006-01-02")+"→"+to.Format("2006-01-02"))
	s.flashOK(r, "İcazə əlavə edildi.")
	http.Redirect(w, r, "/clients/"+id+"#auth", http.StatusSeeOther)
}

func (s *Server) handleAuthUpdate(w http.ResponseWriter, r *http.Request) {
	aid := chi.URLParam(r, "aid")
	a, err := s.store.GetAuthorization(r.Context(), aid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	scope := strings.TrimSpace(r.FormValue("scope"))
	by := strings.TrimSpace(r.FormValue("authorized_by"))
	docRef := strings.TrimSpace(r.FormValue("document_ref"))
	from, err1 := parseDay(r.FormValue("valid_from"), false)
	to, err2 := parseDay(r.FormValue("valid_to"), true)
	if scope == "" || by == "" || err1 != nil || err2 != nil || !to.After(from) {
		s.flashErr(r, "İcazə yenilənmədi — bütün sahələri və tarixləri yoxlayın.")
		http.Redirect(w, r, "/clients/"+a.ClientID, http.StatusSeeOther)
		return
	}
	if err := s.store.UpdateAuthorization(r.Context(), aid, scope, by, docRef, from, to); err != nil {
		s.flashErr(r, "İcazə yenilənmədi (baza xətası).")
		http.Redirect(w, r, "/clients/"+a.ClientID, http.StatusSeeOther)
		return
	}
	s.audit(r, "authorization.update", "authorization", aid, "by "+by)
	s.flashOK(r, "İcazə yeniləndi.")
	http.Redirect(w, r, "/clients/"+a.ClientID, http.StatusSeeOther)
}

func (s *Server) handleAuthDelete(w http.ResponseWriter, r *http.Request) {
	aid := chi.URLParam(r, "aid")
	a, err := s.store.GetAuthorization(r.Context(), aid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := s.store.CountCampaignsUsingAuthorization(r.Context(), aid)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		s.flashErr(r, "İcazə silinmədi — ona bağlı kampaniya(lar) var. Əvvəlcə onları silin.")
		http.Redirect(w, r, "/clients/"+a.ClientID, http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteAuthorization(r.Context(), aid); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "authorization.delete", "authorization", aid, "client "+a.ClientID)
	s.flashOK(r, "İcazə silindi.")
	http.Redirect(w, r, "/clients/"+a.ClientID, http.StatusSeeOther)
}

func (s *Server) handleGroupCreate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/clients/"+id, http.StatusSeeOther)
		return
	}
	groupID, err := s.store.CreateGroup(r.Context(), id, name)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "group.create", "group", groupID, name+" (client "+id+")")
	s.flashOK(r, "Qrup yaradıldı.")
	http.Redirect(w, r, "/clients/"+id+"#groups", http.StatusSeeOther)
}

// parseDay parses an input[type=date] value. endOfDay sets 23:59:59 for valid_to.
func parseDay(s string, endOfDay bool) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), time.Local)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t, nil
}
