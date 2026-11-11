package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// handleProfileTest sends a test email through a sending profile.
func (s *Server) handleProfileTest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	to := strings.TrimSpace(r.FormValue("to"))
	if to == "" || !strings.Contains(to, "@") {
		http.Redirect(w, r, "/profiles?test=bad", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 25*time.Second)
	defer cancel()
	if err := s.launcher.SendTestEmail(ctx, id, to); err != nil {
		// Surface the real reason (the handler used to swallow it): stash a
		// human-readable message in the session flash, read back on the redirect.
		s.sessions.Put(r.Context(), "flash_profile", classifySMTPError(err))
		http.Redirect(w, r, "/profiles?test=err", http.StatusSeeOther)
		return
	}
	s.audit(r, "profile.test", "profile", id, "test email to "+to)
	http.Redirect(w, r, "/profiles?test=ok", http.StatusSeeOther)
}

// classifySMTPError turns a raw send error into an actionable Azerbaijani
// message. The most common trap is a plaintext test server (e.g. MailHog) that
// advertises AUTH: Go's net/smtp refuses PLAIN auth over an unencrypted link to
// a non-localhost host, so the send aborts right after EHLO.
func classifySMTPError(err error) string {
	e := err.Error()
	switch {
	case strings.Contains(e, "unencrypted connection"):
		return "Server şifrəsizdir, amma profildə istifadəçi adı/parol var. Şifrəsiz serverə (məs. MailHog) göndərmək üçün istifadəçi adı və parolu boş saxlayın; real serverdə isə STARTTLS dəstəklənməlidir."
	case strings.Contains(e, "dial smtp"):
		return "SMTP serverinə qoşulmaq alınmadı — host, port və şəbəkə əlçatanlığını yoxlayın. (" + e + ")"
	case strings.Contains(e, "5.7.139") || strings.Contains(e, "SmtpClientAuthentication") || strings.Contains(e, "basic authentication is disabled"):
		return "Office365 bu poçt qutusu üçün SMTP AUTH-u söndürüb. Microsoft 365 admin mərkəzində həmin qutu üçün 'Authenticated SMTP'-ni aktiv edin (və ya PowerShell: Set-CASMailbox <user> -SmtpClientAuthenticationDisabled $false), sonra yenidən yoxlayın. (" + e + ")"
	case strings.Contains(e, "no supported mechanism") || strings.Contains(e, "5.7.4"):
		return "Server dəstəklənməyən autentifikasiya növü tələb edir. (" + e + ")"
	default:
		return "Test emaili göndərilə bilmədi: " + e
	}
}

// handleCampaignSchedule schedules a campaign for a future launch (+ optional drip window).
func (s *Server) handleCampaignSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	launchAt, err := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(r.FormValue("launch_at")), time.Local)
	if err != nil {
		s.flashErr(r, "Başlama vaxtı düzgün deyil.")
		http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
		return
	}
	var sendBy *time.Time
	if v := strings.TrimSpace(r.FormValue("send_by")); v != "" {
		if t, e := time.ParseInLocation("2006-01-02T15:04", v, time.Local); e == nil && t.After(launchAt) {
			sendBy = &t
		}
	}
	if err := s.store.ScheduleCampaign(r.Context(), id, launchAt, sendBy); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "campaign.schedule", "campaign", id, launchAt.Format("2006-01-02 15:04"))
	s.flashOK(r, "Kampaniya "+launchAt.Format("2006-01-02 15:04")+" tarixinə planlaşdırıldı.")
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

// handleCampaignCancel stops a running campaign immediately: it flips the DB
// status to 'cancelled' and signals the in-flight send goroutine to stop, so no
// further emails go out. Safe to call even if the send just finished.
func (s *Server) handleCampaignCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, _ = s.store.CancelCampaign(r.Context(), id)
	s.launcher.Cancel(id)
	s.audit(r, "campaign.cancel", "campaign", id, "")
	s.flashOK(r, "Göndərmə dayandırıldı — qalan emaillar getməyəcək.")
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

func (s *Server) handleCampaignArchive(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = s.store.SetCampaignArchived(r.Context(), id, true)
	s.audit(r, "campaign.archive", "campaign", id, "")
	s.flashOK(r, "Kampaniya arxivləndi.")
	http.Redirect(w, r, "/campaigns", http.StatusSeeOther)
}

func (s *Server) handleCampaignDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if s.tunnels != nil {
		s.tunnels.Stop(id)
	}
	// Capture the name before the cascading delete so the audit entry is meaningful.
	name := id
	if camp, err := s.store.GetCampaign(r.Context(), id); err == nil && camp != nil {
		name = camp.Name
	}
	_ = s.store.DeleteCampaign(r.Context(), id)
	s.audit(r, "campaign.delete", "campaign", id, name)
	s.flashOK(r, "Kampaniya silindi.")
	http.Redirect(w, r, "/campaigns", http.StatusSeeOther)
}

// handleCampaignResend re-queues targets who never opened, then relaunches.
func (s *Server) handleCampaignResend(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, err := s.store.ResetNonRespondersForResend(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		_ = s.store.SetCampaignStatus(r.Context(), id, "scheduled")
		s.audit(r, "campaign.resend", "campaign", id, fmt.Sprintf("%d non-openers re-queued", n))
		s.flashOK(r, fmt.Sprintf("%d nəfər (emaili açmayanlar) yenidən növbəyə alındı.", n))
		go func() {
			if err := s.launcher.Launch(context.Background(), id); err != nil {
				fmt.Printf("resend %s aborted: %v\n", id, err)
			}
		}()
	} else {
		s.flashErr(r, "Yenidən göndəriləcək hədəf yoxdur — hamı emaili açıb.")
	}
	http.Redirect(w, r, "/campaigns/"+id, http.StatusSeeOther)
}

// handleCampaignExport streams the per-target results as CSV.
func (s *Server) handleCampaignExport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.store.ListResultsDetailed(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"campaign_"+id+".csv\"")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"email", "first_name", "last_name", "position", "status", "sent_at"})
	for _, d := range rows {
		sent := ""
		if d.SentAt != nil {
			sent = d.SentAt.Format(time.RFC3339)
		}
		_ = cw.Write([]string{csvSafe(d.Email), csvSafe(d.FirstName), csvSafe(d.LastName), csvSafe(d.Position), d.Status, sent})
	}
	cw.Flush()
}

// csvSafe neutralizes spreadsheet formula injection: a cell beginning with a
// formula trigger is prefixed with a single quote so Excel/LibreOffice treat it
// as text instead of executing it.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
