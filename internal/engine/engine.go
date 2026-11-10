package engine

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"secdrill/internal/auth"
	"secdrill/internal/store"
)

// Launcher sends campaigns. It enforces that a valid authorization exists before
// a single email goes out.
type Launcher struct {
	store         *store.Store
	encKey        []byte // decrypts the SMTP password at send time (credential-encryption key)
	publicBaseURL string
	sendDelay     time.Duration // simple rate limit between messages

	mu      sync.Mutex                    // guards running
	running map[string]context.CancelFunc // campaignID -> cancel for an in-flight send
}

func NewLauncher(st *store.Store, encKey []byte, publicBaseURL string) *Launcher {
	return &Launcher{
		store:         st,
		encKey:        encKey,
		publicBaseURL: publicBaseURL,
		sendDelay:     750 * time.Millisecond,
		running:       map[string]context.CancelFunc{},
	}
}

// register/unregister/Cancel implement a tiny in-memory registry of in-flight
// sends so an operator's "Dayandır" (Stop) request can interrupt a running
// campaign — including a slow drip send that would otherwise run for hours.
func (l *Launcher) register(id string, cancel context.CancelFunc) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.running[id] = cancel
}

func (l *Launcher) unregister(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.running, id)
}

// Cancel signals a running campaign's send loop to stop as soon as possible.
// It returns true if an in-flight send was found and signalled.
func (l *Launcher) Cancel(id string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if cancel, ok := l.running[id]; ok {
		cancel()
		return true
	}
	return false
}

// Launch validates and sends a campaign. Safe to run in a goroutine.
func (l *Launcher) Launch(ctx context.Context, campaignID string) error {
	camp, err := l.store.GetCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("load campaign: %w", err)
	}
	if camp.Status != "draft" && camp.Status != "scheduled" {
		return fmt.Errorf("campaign is not launchable (status=%s)", camp.Status)
	}

	// Authorization gate: a live written authorization MUST cover the launch time.
	ok, err := l.store.ActiveAuthorizationExists(ctx, camp.AuthorizationID, time.Now())
	if err != nil {
		return fmt.Errorf("check authorization: %w", err)
	}
	if !ok {
		return fmt.Errorf("refusing to launch: no active authorization covers this moment")
	}

	// Atomically claim the campaign BEFORE doing any work: flip draft/scheduled
	// -> running in a single UPDATE. Only one caller can win, so a double-clicked
	// launch or the scheduler firing twice cannot send the campaign twice.
	claimed, err := l.store.StartCampaign(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("claim campaign: %w", err)
	}
	if !claimed {
		// Someone else already launched it (or it is no longer launchable).
		return nil
	}

	// From here on the campaign is 'running'; register a cancel so a Stop request
	// can interrupt the send loop, and make sure we always clean up the registry.
	// runCtx is derived from Background (not the caller's ctx) so that the ONLY
	// thing which cancels an in-flight send is an explicit operator Stop — never
	// the request context or the scheduler's shutdown signal. A send interrupted
	// by process shutdown simply stops and is recovered to 'scheduled' on reboot.
	runCtx, cancel := context.WithCancel(context.Background())
	l.register(campaignID, cancel)
	defer cancel()
	defer l.unregister(campaignID)

	tmpl, err := l.store.GetTemplate(ctx, camp.TemplateID)
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}
	profile, enc, err := l.store.GetSendingProfileWithSecret(ctx, camp.SendingProfileID)
	if err != nil {
		return fmt.Errorf("load sending profile: %w", err)
	}
	var password string
	if len(enc) > 0 {
		pw, derr := auth.Decrypt(l.encKey, enc)
		if derr != nil {
			return fmt.Errorf("decrypt smtp password: %w", derr)
		}
		password = string(pw)
	}

	items, err := l.store.PendingResults(ctx, campaignID)
	if err != nil {
		return fmt.Errorf("load pending results: %w", err)
	}

	smtpCfg := SMTPConfig{
		Host: profile.SMTPHost, Port: profile.SMTPPort,
		Username: profile.SMTPUser, Password: password, From: profile.FromAddress,
	}

	// Merge headers: sending-profile headers first, template headers override.
	headers := map[string]string{}
	for k, v := range profile.Headers {
		headers[k] = v
	}
	for k, v := range tmpl.Headers {
		headers[k] = v
	}

	// Determine per-message delay. If a send-by window is set and in the future,
	// spread the messages evenly across it ("drip" send). Otherwise use the
	// default rate limit.
	delay := l.sendDelay
	if camp.SendBy != nil && len(items) > 1 {
		if window := time.Until(*camp.SendBy); window > 0 {
			delay = window / time.Duration(len(items))
		}
	}

	// writeCtx records what HAS happened (a result marked sent, an event logged)
	// and MUST NOT be the cancellable runCtx: an operator Stop can land while a
	// message is being delivered by Send(), and if the follow-up MarkResultSent
	// ran on a cancelled context it would be dropped — leaving a delivered email
	// recorded as still 'scheduled', which a later resend/reschedule would send a
	// SECOND time. runCtx governs ONLY whether we keep going (loop-top check and
	// the inter-message delay); it never governs recording a fact.
	writeCtx := context.Background()

	// stoppedEarly is true only if the loop was interrupted BEFORE sending every
	// item — by an operator Stop (runCtx cancelled) or because the written
	// authorization stopped covering "now" mid-send. A run that sends every item
	// completes normally even if a Stop arrives after the last send.
	stoppedEarly := false
	for i, it := range items {
		if runCtx.Err() != nil {
			stoppedEarly = true
			break
		}
		// Re-check the authorization BEFORE every message. A scheduled or gradual
		// ("drip") send can run for a long time, so checking only once at launch
		// is not enough: if the written authorization's validity window ends while
		// the campaign is still sending, we must stop immediately and never
		// deliver a simulated phishing email outside the authorized period.
		if okAuth, aerr := l.store.ActiveAuthorizationExists(writeCtx, camp.AuthorizationID, time.Now()); aerr != nil || !okAuth {
			log.Printf("campaign %s: authorization no longer valid — halting send", campaignID)
			stoppedEarly = true
			break
		}

		data := TargetData{
			FirstName:     it.First,
			LastName:      it.Last,
			Email:         it.Email,
			Position:      it.Position,
			URL:           clickURL(l.publicBaseURL, it.Token),
			TrackingPixel: pixelTag(l.publicBaseURL, it.Token),
		}

		subject, e1 := renderText(tmpl.Subject, data)
		htmlBody, e2 := renderHTML(tmpl.HTMLBody, data)
		textBody, _ := renderText(tmpl.TextBody, data)
		if e1 != nil || e2 != nil {
			log.Printf("campaign %s: render error for %s: %v %v", campaignID, it.Email, e1, e2)
			_ = l.store.MarkResultError(writeCtx, it.ResultID)
			continue
		}

		// Per-message headers: carry the result token so a reported (forwarded)
		// email can be mapped back to this target.
		msgHeaders := map[string]string{"X-SecDrill-RID": it.Token}
		for k, v := range headers {
			msgHeaders[k] = v
		}

		err := Send(smtpCfg, Message{
			To: it.Email, Subject: subject, HTMLBody: htmlBody, TextBody: textBody,
			Headers: msgHeaders, Attachments: tmpl.Attachments,
		})
		rid := it.ResultID
		// Record on writeCtx so the outcome is never lost to an in-flight Stop.
		if err != nil {
			log.Printf("campaign %s: send error for %s: %v", campaignID, it.Email, err)
			_ = l.store.MarkResultError(writeCtx, rid)
			_ = l.store.RecordEvent(writeCtx, campaignID, &rid, "error", "", "")
		} else {
			_ = l.store.MarkResultSent(writeCtx, rid)
			_, _ = l.store.AdvanceStatus(writeCtx, rid, "sent")
			_ = l.store.RecordEvent(writeCtx, campaignID, &rid, "email_sent", "", "")
		}

		// Interruptible inter-message delay — but only BETWEEN messages, never
		// after the last one (so a Stop during a trailing cooldown cannot mislabel
		// a fully-sent campaign as cancelled). A Stop takes effect promptly even
		// mid-drip.
		if i < len(items)-1 {
			select {
			case <-runCtx.Done():
				stoppedEarly = true
			case <-time.After(delay):
			}
			if stoppedEarly {
				break
			}
		}
	}

	// Terminal status, written with a fresh context (runCtx may be cancelled) and
	// guarded so it only transitions a still-'running' campaign — it never
	// overwrites a status an operator Stop already set.
	final := context.Background()
	if stoppedEarly {
		_, _ = l.store.CancelCampaign(final, campaignID)
	} else {
		_ = l.store.MarkCampaignCompleted(final, campaignID)
	}
	return nil
}

// RunScheduler periodically launches campaigns whose scheduled launch time has
// arrived. Safe to run in a goroutine for the life of the process.
func (l *Launcher) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ids, err := l.store.DueScheduledCampaigns(ctx, time.Now())
			if err != nil {
				log.Printf("scheduler: %v", err)
				continue
			}
			// Launch each due campaign in its own goroutine so a long drip send
			// does not block the scheduler loop (which would delay every other due
			// campaign past its launch time). The atomic StartCampaign claim makes
			// concurrent launches safe: only one goroutine can claim a campaign, so
			// a campaign still returned by a fast second tick is not double-sent.
			// Background context: an in-flight send is stopped only by an operator
			// Stop, not by the scheduler's shutdown.
			for _, id := range ids {
				go func(id string) {
					if err := l.Launch(context.Background(), id); err != nil {
						log.Printf("scheduler launch %s: %v", id, err)
					}
				}(id)
			}
		}
	}
}

// SendTestEmail sends a one-off test message through a sending profile so the
// operator can confirm SMTP settings before launching a campaign.
func (l *Launcher) SendTestEmail(ctx context.Context, profileID, to string) error {
	profile, enc, err := l.store.GetSendingProfileWithSecret(ctx, profileID)
	if err != nil {
		return fmt.Errorf("load sending profile: %w", err)
	}
	var password string
	if len(enc) > 0 {
		pw, derr := auth.Decrypt(l.encKey, enc)
		if derr != nil {
			return fmt.Errorf("decrypt smtp password: %w", derr)
		}
		password = string(pw)
	}
	cfg := SMTPConfig{
		Host: profile.SMTPHost, Port: profile.SMTPPort,
		Username: profile.SMTPUser, Password: password, From: profile.FromAddress,
	}
	return Send(cfg, Message{
		To:       to,
		Subject:  "SecDrill test email",
		TextBody: "Bu, SecDrill SMTP profilinin test mesajıdır. Göndərmə işləyir.",
		HTMLBody: "<p>Bu, <strong>SecDrill</strong> SMTP profilinin test mesajıdır. Göndərmə işləyir.</p>",
		Headers:  profile.Headers,
	})
}
