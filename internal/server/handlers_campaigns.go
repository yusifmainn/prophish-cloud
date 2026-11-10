package server

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"secdrill/internal/auth"
	"secdrill/internal/models"
)

// campaignPageData gathers everything the campaigns page (list + create form)
// needs. Sel holds the currently-submitted selections so the form can be
// re-rendered with the operator's choices intact after a validation error.
func (s *Server) campaignPageData(r *http.Request) map[string]any {
	camps, _ := s.store.ListCampaigns(r.Context(), true)
	clients, _ := s.store.ListClients(r.Context())
	auths, _ := s.store.ListAllAuthorizations(r.Context())
	groups, _ := s.store.ListAllGroups(r.Context())
	tmpls, _ := s.store.ListTemplates(r.Context())
	pages, _ := s.store.ListLandingPages(r.Context())
	profiles, _ := s.store.ListSendingProfiles(r.Context())
	return map[string]any{
		"Campaigns": camps, "Clients": clients, "Authorizations": auths,
		"Groups": groups, "Templates": tmpls, "Pages": pages, "Profiles": profiles,
		"Sel": map[string]any{"Name": "", "ClientID": "", "AuthID": "", "TemplateID": "", "PageID": "", "ProfileID": "", "GroupIDs": map[string]bool{}},
	}
}

func (s *Server) handleCampaignsList(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "campaigns.html", s.campaignPageData(r))
}

func (s *Server) handleCampaignCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/campaigns", http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	clientID := r.FormValue("client_id")
	authID := r.FormValue("authorization_id")
	templateID := r.FormValue("template_id")
	pageID := r.FormValue("landing_page_id")
	profileID := r.FormValue("sending_profile_id")
	groupIDs := r.Form["group_ids"]

	// On any validation failure, re-render the page with a clear Azerbaijani
	// message AND the operator's selections preserved (instead of a blank 400).
	fail := func(msg string) {
		data := s.campaignPageData(r)
		data["Error"] = msg
		data["Sel"] = map[string]any{
			"Name": name, "ClientID": clientID, "AuthID": authID,
			"TemplateID": templateID, "PageID": pageID, "ProfileID": profileID,
			"GroupIDs": strset(groupIDs),
		}
		s.render(w, r, http.StatusBadRequest, "campaigns.html", data)
	}

	if name == "" || clientID == "" || authID == "" || templateID == "" || pageID == "" || profileID == "" || len(groupIDs) == 0 {
		fail("Bütün sahələri doldurun və ən azı bir hədəf qrupu seçin.")
		return
	}
	// Validate the authorization belongs to the chosen client.
	auths, _ := s.store.ListAuthorizations(r.Context(), clientID)
	if !containsAuth(auths, authID) {
		fail("Seçilən icazə bu müştəriyə aid deyil.")
		return
	}
	// Validate every selected group belongs to the chosen client.
	clientGroups, _ := s.store.ListGroups(r.Context(), clientID)
	allowed := map[string]bool{}
	for _, g := range clientGroups {
		allowed[g.ID] = true
	}
	for _, gid := range groupIDs {
		if !allowed[gid] {
			fail("Seçilən qrup bu müştəriyə aid deyil.")
			return
		}
	}

	// Resolve recipients BEFORE creating the campaign so an empty selection does
	// not leave an un-sendable draft behind.
	targetIDs, _ := s.store.TargetIDsForGroups(r.Context(), groupIDs)
	if len(targetIDs) == 0 {
		fail("Seçilmiş qrup(lar)da işçi yoxdur — əvvəlcə işçi əlavə edin.")
		return
	}

	campaignID, err := s.store.CreateCampaign(r.Context(), clientID, authID, name, templateID, pageID, profileID, s.currentUserID(r))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	tokens := make([]string, len(targetIDs))
	for i := range tokens {
		tokens[i] = auth.NewToken()
	}
	if err := s.store.CreateResults(r.Context(), campaignID, targetIDs, tokens); err != nil {
		http.Error(w, "could not create results", http.StatusInternalServerError)
		return
	}

	s.audit(r, "campaign.create", "campaign", campaignID, name)
	s.flashOK(r, "Kampaniya yaradıldı (qaralama). İndi onu işə sala və ya planlaşdıra bilərsiniz.")
	http.Redirect(w, r, "/campaigns/"+campaignID, http.StatusSeeOther)
}

func strset(ss []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range ss {
		m[s] = true
	}
	return m
}

// handleCampaignEditForm renders the create form pre-filled for a DRAFT campaign.
func (s *Server) handleCampaignEditForm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	camp, err := s.store.GetCampaign(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if camp.Status != "draft" {
		s.flashErr(r, "Yalnız qaralama (draft) kampaniyanı redaktə etmək olar.")
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	gids, _ := s.store.CampaignGroupIDs(r.Context(), id)
	data := s.campaignPageData(r)
	data["EditID"] = id
	data["Sel"] = map[string]any{
		"Name": camp.Name, "ClientID": camp.ClientID, "AuthID": camp.AuthorizationID,
		"TemplateID": camp.TemplateID, "PageID": camp.LandingPageID, "ProfileID": camp.SendingProfileID,
		"GroupIDs": strset(gids),
	}
	s.render(w, r, http.StatusOK, "campaigns.html", data)
}

// handleCampaignUpdate reconfigures a DRAFT campaign and regenerates its per-target
// results for the (possibly changed) groups. Launched campaigns are immutable.
func (s *Server) handleCampaignUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	camp, err := s.store.GetCampaign(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if camp.Status != "draft" {
		s.flashErr(r, "Yalnız qaralama (draft) kampaniyanı redaktə etmək olar.")
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	clientID := r.FormValue("client_id")
	authID := r.FormValue("authorization_id")
	templateID := r.FormValue("template_id")
	pageID := r.FormValue("landing_page_id")
	profileID := r.FormValue("sending_profile_id")
	groupIDs := r.Form["group_ids"]

	fail := func(msg string) {
		data := s.campaignPageData(r)
		data["Error"] = msg
		data["EditID"] = id
		data["Sel"] = map[string]any{
			"Name": name, "ClientID": clientID, "AuthID": authID,
			"TemplateID": templateID, "PageID": pageID, "ProfileID": profileID,
			"GroupIDs": strset(groupIDs),
		}
		s.render(w, r, http.StatusBadRequest, "campaigns.html", data)
	}

	if name == "" || clientID == "" || authID == "" || templateID == "" || pageID == "" || profileID == "" || len(groupIDs) == 0 {
		fail("Bütün sahələri doldurun və ən azı bir hədəf qrupu seçin.")
		return
	}
	auths, _ := s.store.ListAuthorizations(r.Context(), clientID)
	if !containsAuth(auths, authID) {
		fail("Seçilən icazə bu müştəriyə aid deyil.")
		return
	}
	clientGroups, _ := s.store.ListGroups(r.Context(), clientID)
	allowed := map[string]bool{}
	for _, g := range clientGroups {
		allowed[g.ID] = true
	}
	for _, gid := range groupIDs {
		if !allowed[gid] {
			fail("Seçilən qrup bu müştəriyə aid deyil.")
			return
		}
	}
	targetIDs, _ := s.store.TargetIDsForGroups(r.Context(), groupIDs)
	if len(targetIDs) == 0 {
		fail("Seçilmiş qrup(lar)da işçi yoxdur.")
		return
	}

	ok, err := s.store.UpdateCampaignConfig(r.Context(), id, name, authID, templateID, pageID, profileID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		s.flashErr(r, "Kampaniya artıq qaralama deyil — redaktə edilə bilmədi.")
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	// A draft has never sent, so clearing and recreating results is safe.
	if err := s.store.DeleteResultsForCampaign(r.Context(), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	tokens := make([]string, len(targetIDs))
	for i := range tokens {
		tokens[i] = auth.NewToken()
	}
	if err := s.store.CreateResults(r.Context(), id, targetIDs, tokens); err != nil {
		http.Error(w, "could not create results", http.StatusInternalServerError)
		return
	}
	s.audit(r, "campaign.update", "campaign", id, name)
	s.flashOK(r, "Kampaniya yeniləndi.")
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

func (s *Server) handleCampaignDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	view, err := s.store.CampaignView(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	stats, _ := s.store.CampaignStats(r.Context(), id)
	camp, _ := s.store.GetCampaign(r.Context(), id)
	events, _ := s.store.ListEventsDetailed(r.Context(), id)
	results, _ := s.store.ListResultsDetailed(r.Context(), id)
	groups, _ := s.store.CampaignStatsByGroup(r.Context(), id)
	pct := percentages(stats)
	riskLabel, riskClass := models.RiskLevel(pct.Submitted, pct.Clicked)
	s.render(w, r, http.StatusOK, "campaign_detail.html", map[string]any{
		"C":         view,
		"Camp":      camp,
		"Stats":     stats,
		"Pct":       pct,
		"Events":    events,
		"Results":   results,
		"Groups":    groups,
		"RiskLabel": riskLabel,
		"RiskClass": riskClass,
		"NowLocal":  time.Now().Format("2006-01-02T15:04"),
	})
}

// handleCampaignRename changes a campaign's display name only. Reconfiguring a
// campaign's components is intentionally not offered — delete the draft and
// recreate it, so a launched campaign's record stays faithful to what was sent.
func (s *Server) handleCampaignRename(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	if err := s.store.RenameCampaign(r.Context(), id, name); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "campaign.rename", "campaign", id, name)
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

func (s *Server) handleCampaignLaunch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	camp, err := s.store.GetCampaign(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if camp.Status != "draft" && camp.Status != "scheduled" {
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	items, _ := s.store.PendingResults(r.Context(), id)
	if len(items) == 0 {
		s.flashErr(r, "Göndəriləcək hədəf yoxdur — kampaniyada boş qrup ola bilər.")
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	s.audit(r, "campaign.launch", "campaign", id, camp.Name)
	s.flashOK(r, "Kampaniya işə salındı — göndərmə başladı.")
	// Run asynchronously so the request returns immediately; the engine re-checks
	// the authorization gate before sending anything.
	go func() {
		if err := s.launcher.Launch(context.Background(), id); err != nil {
			log.Printf("campaign %s launch aborted: %v", id, err)
		}
	}()
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

func containsAuth(auths []models.Authorization, id string) bool {
	for _, a := range auths {
		if a.ID == id {
			return true
		}
	}
	return false
}
