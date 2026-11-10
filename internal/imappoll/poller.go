// Package imappoll polls a mailbox for employee-reported phishing emails and
// records them (the equivalent of Gophish's IMAP "reported email" monitor).
package imappoll

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	"prophish/internal/auth"
	"prophish/internal/store"
)

const (
	maxPerPoll = 200     // max messages processed per poll
	maxBody    = 1 << 20 // 1 MB cap per fetched message body
)

// Monitor periodically polls the configured IMAP mailbox.
type Monitor struct {
	store  *store.Store
	encKey []byte
}

func New(st *store.Store, encKey []byte) *Monitor {
	return &Monitor{store: st, encKey: encKey}
}

// Run loops until ctx is cancelled, polling on the configured interval.
func (m *Monitor) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		wait := 60 * time.Second
		cfg, err := m.store.GetIMAPConfig(ctx)
		if err == nil && cfg.Enabled {
			if cfg.PollSeconds > 0 {
				wait = time.Duration(cfg.PollSeconds) * time.Second
			}
			if perr := m.poll(ctx, cfg); perr != nil {
				log.Printf("imap poll error: %v", perr)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (m *Monitor) poll(ctx context.Context, cfg *store.IMAPConfig) error {
	var password string
	if len(cfg.PasswordEnc) > 0 {
		pw, err := auth.Decrypt(m.encKey, cfg.PasswordEnc)
		if err != nil {
			return fmt.Errorf("decrypt imap password: %w", err)
		}
		password = string(pw)
	}

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	tlsCfg := &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}
	var c *client.Client
	var err error
	if cfg.UseTLS {
		c, err = client.DialTLS(addr, tlsCfg)
	} else {
		// Plaintext connect, then REQUIRE STARTTLS before sending credentials —
		// we never transmit the mailbox password in cleartext.
		c, err = client.Dial(addr)
		if err == nil {
			if terr := c.StartTLS(tlsCfg); terr != nil {
				c.Logout()
				return fmt.Errorf("imap starttls required but unavailable: %w", terr)
			}
		}
	}
	if err != nil {
		return fmt.Errorf("dial imap: %w", err)
	}
	defer c.Logout()

	if err := c.Login(cfg.Username, password); err != nil {
		return fmt.Errorf("imap login: %w", err)
	}
	folder := cfg.Folder
	if folder == "" {
		folder = "INBOX"
	}
	if _, err := c.Select(folder, false); err != nil {
		return fmt.Errorf("select folder: %w", err)
	}

	// Fetch messages with UID greater than the last seen one.
	criteria := imap.NewSearchCriteria()
	criteria.Uid = new(imap.SeqSet)
	criteria.Uid.AddRange(uint32(cfg.LastUID+1), 0) // 0 == '*'
	uids, err := c.UidSearch(criteria)
	if err != nil {
		return fmt.Errorf("uid search: %w", err)
	}
	if len(uids) == 0 {
		return nil
	}
	// Bound work per poll: process the lowest UIDs first so we make forward
	// progress, and never materialize an unbounded batch.
	sort.Slice(uids, func(i, j int) bool { return uids[i] < uids[j] })
	if len(uids) > maxPerPoll {
		uids = uids[:maxPerPoll]
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)
	// Fetch ONLY the X-PROPhish-RID header (peeked, so it isn't marked read),
	// never the whole message body — a hostile/oversized message can't be buffered.
	section := &imap.BodySectionName{
		BodyPartName: imap.BodyPartName{
			Specifier: imap.HeaderSpecifier,
			Fields:    []string{"X-PROPhish-RID"},
		},
		Peek: true,
	}
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.UidFetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}, messages)
	}()

	for msg := range messages {
		var reporter, subject string
		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
			if len(msg.Envelope.From) > 0 {
				reporter = msg.Envelope.From[0].Address()
			}
		}

		rid := ""
		if r := msg.GetBody(section); r != nil {
			// Bounded read, then parse only the TOP-LEVEL headers (not the whole
			// body) so a quoted/forwarded copy of the header cannot forge a report.
			if parsed, perr := mail.ReadMessage(io.LimitReader(r, maxBody)); perr == nil {
				rid = strings.TrimSpace(parsed.Header.Get("X-PROPhish-RID"))
			}
		}

		marked := false
		if rid != "" {
			// Only accept the report if the sender actually is the target that
			// received this token (prevents forged "reported" results).
			if target, terr := m.store.TargetEmailByToken(ctx, rid); terr == nil &&
				reporter != "" && strings.EqualFold(target, reporter) {
				if err := m.store.MarkReportedByToken(ctx, rid, reporter); err == nil {
					marked = true
				}
			}
		}
		if !marked {
			_ = m.store.InsertReportedEmail(ctx, reporter, subject)
		}

		// Advance last_uid per message so a crash cannot re-process hostile mail.
		if int64(msg.Uid) > cfg.LastUID {
			_ = m.store.SetIMAPLastUID(ctx, cfg.ID, int64(msg.Uid))
			cfg.LastUID = int64(msg.Uid)
		}
	}
	if err := <-done; err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	return nil
}
