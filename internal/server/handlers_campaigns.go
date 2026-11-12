package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"prophish/internal/auth"
	"prophish/internal/models"
)

// listenerData computes the listener-location recommendation for the campaign
// form: whether hosting the listener directly on this host is viable (cached
// detection, ≤3s on a cold cache) and the host's public URL. The URL is always
// derived live — never hardcoded: inbound-verified public IP → configured
// global default → this host's own outbound IP on the PUBLIC tracking port.
func (s *Server) listenerData(r *http.Request) (recommend bool, hostURL string) {
	if s.detector != nil {
		res := s.detector.Detect(r.Context())
		recommend = res.Viable
		if res.PublicIP != "" && res.Viable {
			hostURL = "http://" + res.PublicIP
		}
	}
	if hostURL == "" {
		hostURL = s.cfg.PublicBaseURL
	}
	if hostURL == "" {
		// Best-effort: this host's own LAN/outbound IP on the PUBLIC tracking
		// port. NEVER r.Host — that is the admin UI's address (admin port, and
		// often 127.0.0.1 over an SSH tunnel), which carries no /t /p /s routes,
		// so tracking links would 404. This value is reachable by targets only
		// if the public port is published to them (LAN) — the UI says as much.
		if ip := outboundIP(); ip != "" {
			hostURL = "http://" + net.JoinHostPort(ip, publicPort(s.cfg.PublicAddr))
		}
	}
	return recommend, hostURL
}

// outboundIP returns this host's primary outbound IP (the source address the
// kernel would use to reach the internet), or "" if it cannot be determined.
// The UDP "dial" sends no packets — it only resolves the routing source
// address — so it works even when the target is unreachable.
func outboundIP() string {
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer c.Close()
	if a, ok := c.LocalAddr().(*net.UDPAddr); ok && a.IP != nil {
		return a.IP.String()
	}
	return ""
}

// publicPort extracts the port from the public listener address
// (PROPHISH_PUBLIC_ADDR, e.g. "0.0.0.0:8080" -> "8080"), defaulting to "8080".
func publicPort(addr string) string {
	if _, port, err := net.SplitHostPort(addr); err == nil && port != "" {
		return port
	}
	return "8080"
}

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
	recommend, hostURL := s.listenerData(r)
	mode := "cloudflare"
	if recommend {
		mode = "host"
	}
	return map[string]any{
		"Campaigns": camps, "Clients": clients, "Authorizations": auths,
		"Groups": groups, "Templates": tmpls, "Pages": pages, "Profiles": profiles,
		"GlobalPublicURL": s.cfg.PublicBaseURL,
		"RecommendHost":   recommend,
		"HostPublicURL":   hostURL,
		"ListenerMode":    mode,
		"Sel":             map[string]any{"Name": "", "ClientID": "", "AuthID": "", "TemplateID": "", "PageID": "", "ProfileID": "", "GroupIDs": map[string]bool{}, "PublicBaseURL": ""},
	}
}

// resolveListenerMode turns the form's listener-location choice into the
// campaign's public_base_url + tunnel flag: host → the derived host URL pinned,
// cloudflare → a fresh managed quick tunnel, custom → the sanitized manual URL.
func (s *Server) resolveListenerMode(r *http.Request) (publicURL string, wantTunnel bool, mode string, ok bool) {
	mode = r.FormValue("listener_mode")
	switch mode {
	case "host":
		_, hostURL := s.listenerData(r)
		return hostURL, false, mode, true
	case "custom":
		u, uok := sanitizePublicBaseURL(r.FormValue("public_base_url"))
		return u, false, mode, uok
	default:
		mode = "cloudflare"
		return "", true, mode, true
	}
}

// sanitizePublicBaseURL validates the per-campaign public URL override: blank
// is fine (falls back to the global default), otherwise it must be an
// http(s):// URL; the trailing slash is stripped so link builders can append.
func sanitizePublicBaseURL(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", true
	}
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		return "", false
	}
	return strings.TrimRight(v, "/"), true
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
	publicURL, wantTunnel, listenerMode, urlOK := s.resolveListenerMode(r)

	// On any validation failure, re-render the page with a clear Azerbaijani
	// message AND the operator's selections preserved (instead of a blank 400).
	fail := func(msg string) {
		data := s.campaignPageData(r)
		data["Error"] = msg
		data["ListenerMode"] = listenerMode
		data["Sel"] = map[string]any{
			"Name": name, "ClientID": clientID, "AuthID": authID,
			"TemplateID": templateID, "PageID": pageID, "ProfileID": profileID,
			"GroupIDs": strset(groupIDs), "PublicBaseURL": r.FormValue("public_base_url"),
		}
		s.render(w, r, http.StatusBadRequest, "campaigns.html", data)
	}

	if name == "" || clientID == "" || authID == "" || templateID == "" || pageID == "" || profileID == "" || len(groupIDs) == 0 {
		fail("Bütün sahələri doldurun və ən azı bir hədəf qrupu seçin.")
		return
	}
	if !urlOK {
		fail("Public URL http:// və ya https:// ilə başlamalıdır.")
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

	campaignID, err := s.store.CreateCampaign(r.Context(), clientID, authID, name, templateID, pageID, profileID, s.currentUserID(r), publicURL, wantTunnel)
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
	if wantTunnel && s.tunnels != nil {
		if err := s.tunnels.Start(campaignID); err != nil {
			s.flashErr(r, "Kampaniya yaradıldı, amma tunnel başladıla bilmədi: "+err.Error())
			http.Redirect(w, r, "/campaigns/"+campaignID, http.StatusSeeOther)
			return
		}
		s.flashOK(r, "Kampaniya yaradıldı — tunnel başladılır, URL bir neçə saniyəyə hazır olur.")
		http.Redirect(w, r, "/campaigns/"+campaignID, http.StatusSeeOther)
		return
	}
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
	data["TunnelManaged"] = camp.TunnelManaged
	// Hazırki rejim: idarə olunan tunnel → cloudflare; törədilmiş host URL-i ilə
	// üst-üstə düşən override → host; qalan boş olmayan override → xüsusi.
	mode := data["ListenerMode"].(string)
	if camp.TunnelManaged {
		mode = "cloudflare"
	} else if camp.PublicBaseURL != "" {
		mode = "custom"
		if camp.PublicBaseURL == data["HostPublicURL"] {
			mode = "host"
		}
	}
	data["ListenerMode"] = mode
	data["Sel"] = map[string]any{
		"Name": camp.Name, "ClientID": camp.ClientID, "AuthID": camp.AuthorizationID,
		"TemplateID": camp.TemplateID, "PageID": camp.LandingPageID, "ProfileID": camp.SendingProfileID,
		"GroupIDs": strset(gids), "PublicBaseURL": camp.PublicBaseURL,
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
	publicURL, wantTunnel, listenerMode, urlOK := s.resolveListenerMode(r)

	fail := func(msg string) {
		data := s.campaignPageData(r)
		data["Error"] = msg
		data["EditID"] = id
		data["ListenerMode"] = listenerMode
		data["Sel"] = map[string]any{
			"Name": name, "ClientID": clientID, "AuthID": authID,
			"TemplateID": templateID, "PageID": pageID, "ProfileID": profileID,
			"GroupIDs": strset(groupIDs), "PublicBaseURL": r.FormValue("public_base_url"),
		}
		s.render(w, r, http.StatusBadRequest, "campaigns.html", data)
	}

	if name == "" || clientID == "" || authID == "" || templateID == "" || pageID == "" || profileID == "" || len(groupIDs) == 0 {
		fail("Bütün sahələri doldurun və ən azı bir hədəf qrupu seçin.")
		return
	}
	if !urlOK {
		fail("Public URL http:// və ya https:// ilə başlamalıdır.")
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

	ok, err := s.store.UpdateCampaignConfig(r.Context(), id, name, authID, templateID, pageID, profileID, publicURL, wantTunnel)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !ok {
		s.flashErr(r, "Kampaniya artıq qaralama deyil — redaktə edilə bilmədi.")
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	// Listener keçidi: Cloudflare-ə keçid təzə tunnel açır; ondan çıxış
	// idarə olunan tunneli dayandırır.
	if s.tunnels != nil && wantTunnel != camp.TunnelManaged {
		if wantTunnel {
			if err := s.tunnels.Start(id); err != nil {
				s.flashErr(r, "Kampaniya yeniləndi, amma tunnel başladıla bilmədi: "+err.Error())
				http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
				return
			}
		} else {
			s.tunnels.Stop(id)
		}
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
	_, hostURL := s.listenerData(r)
	s.render(w, r, http.StatusOK, "campaign_detail.html", map[string]any{
		"C":            view,
		"Camp":         camp,
		"HostPublicURL": hostURL,
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
