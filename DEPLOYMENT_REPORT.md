# PROPhish — Deployment Report

**Self-hosted phishing-awareness simulation platform**
Document version: 1.0 · Date: 13 November 2026 · Status: Deployed and operational

---

## 1. Purpose of this document

This report documents how PROPhish was built and brought into service, from an empty host to a running two-server deployment. It covers the decisions we made along the way, the tooling we relied on and why, what every part of the codebase is responsible for, and the full feature set as it stands today. It is written for the engineers who will operate the platform and for the stakeholders who need to understand what was delivered and how it is kept safe.

PROPhish is an internally hosted platform for running authorized phishing-awareness exercises. Under a signed engagement, it lets us measure how employees of a client organization respond to simulated social-engineering emails and follow up with training for those who need it. It is comparable in scope to Gophish, with one deliberate difference in philosophy: nothing ever leaves our own infrastructure, and the platform refuses, by design, to capture the credentials a victim would type. More on that below.

---

## 2. What we deployed, at a glance

The running system is a single Go binary that exposes two independent HTTP servers backed by one PostgreSQL database, all orchestrated with Docker Compose:

| Component | Port | Who reaches it | Role |
|---|---|---|---|
| Admin panel | `8443` | Our team only, over VPN / SSH tunnel / IP allowlist | Clients, authorizations, groups, templates, campaigns, reporting |
| Public server | `8080` | The simulation targets (employees) | Landing pages and open/click/submit tracking only |
| PostgreSQL | `5432` | The app (localhost only) | All persistent data |

The split is the heart of the design. The admin panel holds everything sensitive and is never exposed to the internet. The public server exposes only three URL prefixes — `/t/` (click), `/p/` (open pixel), `/s/` (form submission) — and is the only surface a target ever touches.

```
                ┌──────────────────────────────┐
  Our team      │  ADMIN PANEL  :8443          │  ← VPN / localhost / IP allowlist
  (VPN) ───────▶│  clients, RoE, groups,       │
                │  templates, campaigns, reports│
                └──────────────┬───────────────┘
                               │  PostgreSQL (isolated, localhost-bound)
                ┌──────────────┴───────────────┐
  Employees     │  PUBLIC SERVER  :8080        │  ← only /t/ /p/ /s/
  (target LAN)  │  landing pages + tracking     │
                └──────────────────────────────┘
```

---

## 3. Technology stack and why we chose it

Every dependency earns its place; we kept the tree small on purpose.

| Tool / library | Version | What it does | Why it is here |
|---|---|---|---|
| **Go** | 1.27 | Application language | Compiles to a single static binary with no runtime to install, which is exactly what a locked-down host wants. Strong standard library for HTTP, crypto and templating. |
| **PostgreSQL** | 16 (alpine) | Relational database | Multi-tenant data with real constraints (CHECK, foreign keys, JSONB for custom headers). Reliable, well understood, isolated on the host. |
| **Docker + Compose** | — | Build and run | Reproducible builds and a one-command stack. Also sidesteps a practical annoyance: Windows Defender tends to quarantine executables that contain the word "phishing" or phishing markup. Running inside a Linux container removes that friction entirely and matches production. |
| **`jackc/pgx` v5** | 5.9.2 | Postgres driver + pool | High-performance native driver with a connection pool; no `database/sql` overhead. |
| **`go-chi/chi` v5** | 5.1.0 | HTTP router | Lightweight, idiomatic routing and middleware (throttling, recoverer) for both servers. |
| **`alexedwards/scs` v2** | 2.8.0 | Session management | Server-side sessions with HttpOnly, SameSite=Lax, short lifetimes. |
| **`golang.org/x/crypto`** | 0.58.0 | bcrypt, HKDF | bcrypt for operator passwords; HKDF to derive a separate encryption key (§7). |
| **`emersion/go-imap`** | 1.2.1 | IMAP client | Monitors a mailbox for employee-reported phishing (the "I reported it" signal). |
| **`go-pdf/fpdf`** | 0.9.0 | PDF generation | Exportable campaign reports. |
| **cloudflared** | latest (static) | Per-campaign public tunnels | Gives each campaign a reachable `https://…trycloudflare.com` URL when targets are off the LAN (webmail, phones, external networks). Pulled as the official static binary and run without a shell. |
| **MailHog** | v1.0.1 | Local test SMTP (optional) | A throwaway mail sink for rehearsing campaigns without touching a real mail server. Enabled only via the `test` Compose profile. |
| **distroless/static-debian12:nonroot** | — | Runtime base image | No shell, no package manager, runs as non-root, ships CA certificates. Minimal attack surface for an internet-adjacent service. |

---

## 4. Repository layout — what each file is for

### Top level

| Path | Purpose |
|---|---|
| `Dockerfile` | Multi-stage build. Stage one compiles a static, trimmed binary (`CGO_ENABLED=0`, `-trimpath`, `-ldflags "-s -w"`) and downloads `cloudflared`. Stage two copies both into a distroless non-root image. HTML templates are embedded in the binary, so the final image carries no loose files. |
| `docker-compose.yml` | Defines the `db`, `app` and optional `mailhog` services, the health check gate, port bindings, the named data volume, and a 1400-byte network MTU (see §6). |
| `.env.example` | Annotated template for the real `.env`. Documents every environment variable and its security implications. |
| `.dockerignore` / `.gitignore` | Keep secrets (`.env`), logs, local binaries and demo screenshots out of the image and out of version control. |
| `go.mod` / `go.sum` | Dependency manifest and checksums. |
| `README.md` | Operator-facing quick start and security checklist. |
| `cmd/prophish/main.go` | Program entry point: loads config, connects and migrates the DB, recovers interrupted campaigns, starts background workers and both HTTP servers, and handles graceful shutdown. |

### `internal/` packages

| Package | Responsibility |
|---|---|
| `config` | Loads and validates all runtime configuration from the environment (and an optional `.env`). Rejects weak or missing keys. Derives the credential-encryption key. |
| `database` | Connects the pgx pool (`db.go`) and runs embedded SQL migrations on startup (`migrate.go`). |
| `store` | The data-access layer — one file per domain (`clients.go`, `campaigns.go`, `targets.go`, `events.go`, `reporting.go`, …). All SQL lives here. |
| `models` | Shared struct definitions for the domain entities. |
| `auth` | bcrypt password hashing, AES-256-GCM credential encryption (`crypto.go`), and secure token/API-key generation (`token.go`). |
| `engine` | The sending engine: template rendering with recipient variables (`templating.go`), SMTP delivery (`mailer.go`), the scheduler, and the re-checked authorization gate (`engine.go`). |
| `publicsrv` | The internet-facing listener — serves landing pages, records opens/clicks/submissions, enforces rate limits (`ratelimit.go`), and deliberately discards submitted form values. |
| `server` | The admin panel: routing, middleware, i18n, and one handler file per screen (`handlers_campaigns.go`, `handlers_clients.go`, `handlers_builder.go`, the REST API in `api.go`, the in-app documentation in `handlers_docs.go`, and so on). |
| `tunnelmgr` | Supervises one `cloudflared` process per campaign, scrapes the assigned tunnel URL from its output, and persists it. Restarts with backoff. |
| `netutil` | Detects whether this host is directly reachable from the internet, so the campaign form can recommend the right listener (direct host vs Cloudflare gateway). |
| `imappoll` | Background IMAP poller that detects employee-reported simulation emails. |
| `importer` | Parses `.eml` and raw-text email imports into templates. |
| `library` | The built-in, versioned template library (brand-faithful login pages and matching lures) plus its static assets. |
| `report` | PDF report generation and its embedded fonts. |

### `migrations/`

Seven ordered SQL files plus `embed.go`, which bakes them into the binary. They are the single source of truth for the schema and apply automatically at startup. The sequence tells its own story of how the product grew:

1. `0001_init.sql` — core schema: users, clients, authorizations, groups, targets, email templates, landing pages, sending profiles, campaigns, results, events.
2. `0002_full_features.sql` — API keys, gradual-send windows, archiving, custom headers (JSONB), attachments, IMAP config, reported-email table, widened status/event constraints.
3. `0003_reported_flag.sql` — a dedicated `reported` flag so reporting an email no longer overwrites a "clicked/submitted" result.
4. `0004_audit_log.sql` — append-only audit trail of operator actions.
5. `0005_campaign_public_url.sql` — per-campaign external base URL.
6. `0006_campaign_tunnel.sql` — marks campaigns whose Cloudflare tunnel we manage.
7. `0007_campaign_kind.sql` — adds the `public` campaign kind (track-and-redirect, no hosted form) and makes the landing page optional for it.

### `samples/`

Ready-to-import example lures and landing pages (Microsoft 365, OWA, Google Workspace, OneDrive/SharePoint share, IT password-expiry, plus Fortinet and GlobalProtect VPN and an HR year-end bonus theme). The `.eml` files are Outlook-compatible and can be imported directly or opened in a mail client for inspection.

---

## 5. Deployment steps, start to finish

This is the exact path we followed to stand the platform up, and the one an operator repeats on a fresh host.

### Step 1 — Prepare the host

A single Linux host with Docker and the Compose plugin. The host sits on an isolated/internal network; the database and admin ports are never published to the public internet.

### Step 2 — Generate the session secret

```bash
openssl rand -base64 32
```

This produces the 32-byte secret PROPhish uses to sign session cookies. The same secret is *not* reused directly for encryption — the app derives a second, independent key from it via HKDF (§7). The config loader refuses to start if the key is missing, is the placeholder value, or does not decode to exactly 32 bytes.

### Step 3 — Write the `.env` file

Copy `.env.example` to `.env` and fill in the real values. The settings that actually matter at deploy time:

```dotenv
PROPHISH_DB_PASSWORD=<strong, unique password>
PROPHISH_SESSION_KEY=<output of the openssl command above>
PROPHISH_TIMEZONE=Asia/Baku
PROPHISH_SECURE_COOKIES=false        # set true once behind HTTPS
PROPHISH_ADMIN_ALLOW_CIDRS=10.30.22.0/24,10.20.51.0/24,10.20.25.223/32
```

The time zone matters more than it looks: without it, a campaign scheduled for "09:00" fires at 09:00 UTC rather than 09:00 local. The IANA time-zone database is embedded in the binary, so any zone name resolves even inside the minimal container.

### Step 4 — Decide where tracking links point

This is the single most common source of "why aren't opens showing up" and we treat it deliberately. The tracking pixel and the landing-page link are written into every email, so they must resolve from the *recipient's* machine, not from the server.

- **On-LAN recipients:** leave `PROPHISH_PUBLIC_BASE_URL` empty and set `PROPHISH_PUBLIC_BIND=0.0.0.0`. The app auto-detects this host's LAN address and uses `http://<lan-ip>:8080`. A `localhost`/`127.0.0.1` value is a trap: it points every link at the recipient's own machine, so nothing ever registers. The public server now explicitly refuses to pin a loopback tracking URL.
- **Off-LAN recipients** (webmail, phones, external networks): use the Cloudflare gateway listener per campaign, which hands out a public `https` URL automatically. A private-IP host URL cannot be reached by a webmail provider's image proxy, which is why opens won't record in host mode for those recipients.

### Step 5 — Bring the stack up

```bash
docker compose up -d --build
```

Compose builds the image, starts PostgreSQL, waits for its health check to pass, then starts the app. On first start the app creates the `schema_migrations` table and applies all seven migrations inside individual transactions.

### Step 6 — Create the first administrator

Open the admin panel at `http://127.0.0.1:8443` (over a tunnel if remote). The first visit walks through creating the initial admin account, with a minimum 12-character password enforced server-side.

### Step 7 — Verify and hand over

Confirm both servers are listening (the logs print both addresses), confirm migrations applied, and confirm the admin panel is reachable only from the allowed networks. The platform is now ready for the operating workflow in §9.

Routine commands:

```bash
docker compose down        # stop, keep data
docker compose down -v      # stop and delete all data
docker compose --profile test up -d   # also start the MailHog test mail sink
```

---

## 6. Two deployment problems we solved

Two issues surfaced during bring-up that are worth recording, because they are not obvious and they will recur on new hosts.

**Network MTU black-holing.** Some corporate NAT and VPN/overlay networks have a path MTU below 1500 and silently drop the ICMP "fragmentation needed" message. Full-size packets then vanish while small requests succeed, which breaks the TLS handshake the per-campaign Cloudflare tunnel makes to `api.trycloudflare.com` — the tunnel appears to hang for no reason. Docker's legacy default bridge happens to clamp MSS and hides this; user-defined Compose networks do not. We set a conservative `1400` MTU on the Compose network to make tunnels reliable out of the box, and it can be raised toward 1500 where the host path allows.

**The `api.trycloudflare.com` false positive.** `cloudflared` prints `https://api.trycloudflare.com` in its control-plane and in failure messages. That is the API endpoint, not an assigned tunnel hostname, and saving it as a campaign's public URL would send every target to a dead link. The tunnel manager's URL extractor explicitly filters it out and only accepts a real assigned `*.trycloudflare.com` hostname.

---

## 7. Configuration reference

| Variable | Default | Meaning |
|---|---|---|
| `PROPHISH_DB_URL` | — (required) | PostgreSQL connection string. |
| `PROPHISH_DB_PASSWORD` | — (required) | Database password; Compose refuses to start without it. |
| `PROPHISH_SESSION_KEY` | — (required) | 32-byte base64/hex secret for cookie signing. |
| `PROPHISH_ADMIN_ADDR` | `127.0.0.1:8443` | Admin panel listen address. |
| `PROPHISH_ADMIN_BIND` | `127.0.0.1` | Host bind for the published admin port. `0.0.0.0` enables direct network access. |
| `PROPHISH_ADMIN_ALLOW_CIDRS` | operator networks | CIDRs permitted to open the admin panel. Loopback is always allowed; enforced in-app on the real peer IP. |
| `PROPHISH_PUBLIC_ADDR` | `127.0.0.1:8080` | Public server listen address. |
| `PROPHISH_PUBLIC_BIND` | `127.0.0.1` | Host bind for the public port. `0.0.0.0` publishes it on the LAN. |
| `PROPHISH_PUBLIC_BASE_URL` | empty | External base URL baked into emails. Empty = auto-detect the host's LAN address. |
| `PROPHISH_SECURE_COOKIES` | `false` | Set `true` when served over HTTPS. |
| `PROPHISH_TIMEZONE` | `UTC` | IANA zone for scheduling and display. |
| `PROPHISH_TRUSTED_PROXIES` | empty | CIDRs whose `X-Forwarded-For` the public server will honor. Empty = trust only the direct peer, so client IPs cannot be spoofed. |

A note on key separation, because it is a deliberate design point. The one secret you configure (`PROPHISH_SESSION_KEY`) is used *only* to sign session cookies. The key that encrypts stored SMTP and IMAP passwords is derived from it with HKDF-SHA256 under a fixed label, so the two keys are cryptographically independent even though you manage a single secret. The loader also refuses a raw passphrase as the key — it must be properly encoded 32 random bytes — so a low-entropy password can never become the key protecting stored credentials.

---

## 8. Feature set

### Engagement governance (what sets PROPhish apart)

- **Multi-tenant isolation.** Every record is scoped to a `client_id`; one client's data never bleeds into another's.
- **Authorization / Rules of Engagement gate.** A campaign cannot be launched unless a valid written authorization exists for the client — scope, signatory, document reference, validity dates. The gate is re-checked on *every message* during a send, including scheduled and gradual sends, so an expired authorization halts delivery immediately rather than only at launch.
- **Audit log.** An append-only trail of operator actions.

### Targets and content

- Target groups with one-by-one entry or CSV import (`email, first_name, last_name, position`).
- Email templates with recipient variables (`{{.FirstName}}`, `{{.LastName}}`, `{{.Email}}`, `{{.Position}}`, `{{.URL}}`, `{{.TrackingPixel}}`), HTML and text.
- Email import from `.eml` or raw message text.
- File attachments and custom email headers (per template and per sending profile).
- Landing pages, including cloning a real site from a URL.
- **Visual landing-page builder** with multi-page scenarios and per-viewport (tablet/mobile) layouts.
- **Built-in template library** — brand-faithful, versioned login pages (Microsoft 365 / Entra two-step, Outlook Web App, Google Workspace, OneDrive/SharePoint share, IT password-expiry) each paired with an Outlook-ready lure. One click turns a library entry into an editable page and template. These are static, hand-built HTML — no AI generation — tuned to Outlook's Word-based rendering engine (table layout, inline styles, `bgcolor` buttons, preheader, and a copy-link fallback line).

### Sending and campaigns

- SMTP sending profiles with the password encrypted at rest.
- "Send test email" before you commit to a group.
- Campaign creation and launch with double-launch protection — pressing launch twice still sends each target exactly once.
- Scheduling: future launch time plus a gradual-send window.
- Immediate **Stop** of a running campaign (un-sent mail does not go out).
- Archive, delete, and resend-to-non-openers.
- Three campaign kinds: local (host-served landing page), gateway (Cloudflare tunnel), and **public** (track the click, then redirect to a real site with no hosted form).

### Tracking and reporting

- Open, click and form-submission tracking. Each event is recorded only on the first transition into that stage, so neither a target nor anyone who guesses a token can flood the events table.
- Per-employee timeline and a results table with counts and percentages, plus a translated risk band.
- CSV export (with spreadsheet formula-injection neutralized) and PDF reports.
- **IMAP reported-email monitoring** — detects which employees forwarded the simulation to the security team; tracked as a separate flag so it never masks a failure.

### Operations and access

- User management: operators, roles (RBAC — sensitive actions are admin-only), password changes.
- REST API with hashed API keys (`X-API-Key`), e.g. `GET /api/campaigns`.
- In-app illustrated documentation.
- Full tri-lingual interface and documentation: Azerbaijani, English, Russian, with an in-app switcher.

### The one thing it deliberately does **not** do

PROPhish never captures the username or password a target types. When the landing-page form is submitted, the public server records the *fact* of submission and discards the posted values — it does not even read them. This is privacy-by-design: the awareness goal is served by knowing someone would have fallen for it, and storing real credentials would be an unnecessary liability. Live testing confirmed zero credential leakage in the database.

---

## 9. Operating workflow

1. Add the **client**.
2. Record the **authorization (RoE)** — scope, signatory, document reference, validity dates. Without it, nothing sends.
3. Create a **target group** (manual or CSV import).
4. Build an **email template** (or import one, or take one from the library).
5. Build a **landing page** whose form posts to `{{.SubmitURL}}`.
6. Add an **SMTP sending profile**.
7. Assemble the **campaign**: client + authorization + group(s) + template + page + SMTP → draft.
8. **Launch.** The system re-verifies the authorization, then sends.
9. **Stop** at any time if needed.
10. Read the **report**: sent / opened / clicked / submitted, with counts and percentages.

---

## 10. Security posture

Security was treated as a first-class deliverable, not an afterthought. A structured, multi-pass review of the whole codebase was run, with each finding independently re-verified before it was accepted, and every confirmed finding was remediated. A second review round after the fixes found no Critical or High issues remaining and confirmed the earlier fixes held; the additional lower-severity items it surfaced were also closed.

Measures now in place:

- **SSRF protection** on URL-based site cloning: the real resolved IP is checked at dial time and loopback/private/link-local/metadata (`169.254.169.254`) targets are rejected, which also defeats DNS-rebinding.
- **Email header safety:** all header values are stripped of CR/LF and subjects are RFC 2047-encoded, closing header-injection.
- **Sessions** are validated against the database on every request, so a disabled or deleted operator is ejected immediately. Short lifetime (8h) and idle timeout (30m), HttpOnly, SameSite=Lax.
- **CSRF:** SameSite plus an Origin/Referer check.
- **Login hardening:** brute-force lockout (per account, not bypassable via `X-Forwarded-For`) and timing-constant comparison.
- **Public listener hardening:** a global in-flight cap and a per-real-client-IP rate limit, so flooding cannot amplify into the shared database; `X-Forwarded-For` is honored only from configured trusted proxies.
- **Secrets at rest:** SMTP/IMAP passwords encrypted with AES-256-GCM under an HKDF-derived key; API keys stored only as SHA-256 hashes (the plaintext is shown once); operator passwords bcrypt-hashed.
- **Minimal runtime:** distroless, non-root, shell-less image; static binary with filesystem paths stripped.
- **Crash recovery:** campaigns left `running` by a restart are reset to `scheduled` so an operator must deliberately relaunch; a relaunch only sends to targets not already sent.

### Production checklist

- Keep the admin panel (`:8443`) off the public internet — VPN, SSH tunnel, or the IP allowlist.
- Front the public server with a TLS reverse proxy and forward only `/t/`, `/p/`, `/s/`.
- Set `PROPHISH_SECURE_COOKIES=true` under HTTPS.
- Never expose the Postgres port; apply at-rest disk encryption.
- Keep `.env` out of git (already ignored); use a strong DB password.
- Hold a signed RoE for every campaign and record it in the system.
- Define a data-retention policy for after each engagement.
- Run an independent security audit before going live — this is a real step, not a formality.

---

## 11. Known limitations and follow-ups

- The **IMAP reported-email monitor** is complete in code but should be validated against a live mailbox before it is relied on operationally. Every other feature has been exercised in live testing.
- Lower-risk items still open as recommendations: a separate DB pool for the public server; an event-retention/archive job; DKIM/DMARC alignment checks on reported mail (needs mail-authentication infrastructure); and scheduled rotation of the `.env` secrets in production.

---

## 12. Legal note

PROPhish is for authorized security-awareness and penetration-testing work only, against organizations that have given written permission. Unauthorized use is illegal. The authorization/RoE workflow is built into the platform precisely to keep every exercise on the right side of that line.

---

## Appendix — Test and build

```bash
go test ./...          # templating, send-engine variable merge, tracking pixel,
                       # encryption round-trip, landing render, admin IP allowlist
docker compose up -d --build
```

On some Windows hosts, Defender may temporarily block freshly built test executables (`Access is denied`). That is an antivirus reaction to the subject matter, not a code fault; running the tests under Docker/Linux avoids it.
