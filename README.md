# PROPhish

**Öz serverində işləyən, təhlükəsizlik-prioritetli phishing məlumatlandırma (awareness) simulyasiya platforması.**
Müqavilə bağladığınız müştəri təşkilatlarda, yazılı icazə əsasında, işçilərin sosial mühəndislik hücumlarına həssaslığını ölçmək və keçməyənlərə təlim vermək üçün nəzərdə tutulub.

> Bu alət Gophish-ə bənzər funksiyalar verir, amma tamamilə sizin infrastrukturunuzda qalır — müştəri məlumatları heç bir üçüncü tərəf buluduna çıxmır.

---

## Əsas təhlükəsizlik və privacy prinsipləri

- 🔒 **Parollar saxlanılmır.** Landing səhifəsində işçi forma göndərəndə yalnız *göndərmə faktı* qeyd olunur. Daxil edilən dəyərlər (istifadəçi adı, parol) **oxunmur və yazılmır**. (Canlı testlə təsdiqləndi: DB-də 0 leak.)
- 🧱 **İki ayrı server:** *admin panel* (yalnız sizin komanda, VPN arxasında) və *public server* (işçilərin gördüyü, yalnız `/t/`, `/p/`, `/s/` yolları).
- 📝 **İcazə şərti:** Kampaniya yalnız müştəri üçün *etibarlı yazılı icazə (Authorization/RoE)* mövcud olduqda göndərilə bilir. Göndərmə mühərriki bunu hər dəfə yenidən yoxlayır.
- 🏢 **Multi-tenant:** Hər müştərinin məlumatı `client_id` ilə izolə olunur.
- 🔐 SMTP parolları AES-256-GCM ilə şifrələnərək saxlanılır; admin sessiyaları HttpOnly cookie; bütün cavablarda sərt təhlükəsizlik başlıqları (CSP, X-Frame-Options və s.).

---

## Funksiyalar (Gophish pariteti)

| Funksiya | Status |
|---|---|
| Müştərilər (multi-tenant) + icazə/RoE şərti | ✅ (Gophish-də yoxdur) |
| Hədəf qrupları + CSV import | ✅ |
| Email şablonları (dəyişənlər, HTML+mətn) | ✅ |
| **Email import (.eml / xam mətn)** | ✅ |
| **Attachment-lar** (fayl əlavələri) | ✅ |
| **Xüsusi email başlıqları** (şablon + profil) | ✅ |
| Landing səhifələri | ✅ |
| **Sayt klonlama (URL-dən import)** | ✅ |
| **Hazır şablon kitabxanası** (real brend login səhifələri + Outlook email-ləri) | ✅ (Gophish-də hazır gəlmir) |
| SMTP profilləri (şifrəli parol) | ✅ |
| **Test email göndər** | ✅ |
| Kampaniya yaratma + işə salma (təkrar işə salma qorunur) | ✅ |
| **Planlaşdırma (gələcək tarix + tədricən göndərmə)** | ✅ |
| **Göndərməni dərhal dayandır (Stop)** | ✅ |
| **Kampaniya: arxivlə / sil / açmayanlara təkrar göndər** | ✅ |
| İzləmə: açılma / klik / forma göndərişi | ✅ |
| **Hər-işçi timeline + nəticə cədvəli** | ✅ |
| **CSV export** | ✅ |
| **İstifadəçi idarəsi** (operatorlar, rollar, parol dəyişmə) | ✅ |
| **REST API + API açarları** | ✅ |
| **IMAP reported-email izləmə** | ✅ (canlı poçt qutusu ilə test lazımdır) |
| Parol tutma | ❌ **qəsdən yoxdur** (privacy-by-design) |

> **Qeyd:** IMAP reported-email monitoru kod səviyyəsində hazırdır, amma real IMAP poçt qutusu ilə canlı sınaqdan keçirilməlidir. Digər bütün funksiyalar canlı test olundu.

### Şablon kitabxanası (hazır dəstlər)

*Səhifələr → Kitabxana* bölməsində real brend giriş səhifələrinə **birebir bənzəyən** hazır dəstlər var — hər dəstin yanında uyğun email lure-u da gəlir. Bir kliklə («İstifadə et») redaktə oluna bilən açılış səhifəsinə və email şablonuna çevrilir (heç bir AI generasiyası deyil, sabit, versiyalanmış HTML).

| Dəst | Növ |
|---|---|
| Microsoft 365 girişi (Azure AD / Entra, iki-mərhələli) | Giriş səhifəsi |
| Outlook Web App (Exchange) | Giriş səhifəsi |
| Google Workspace / Gmail | Giriş səhifəsi |
| OneDrive / SharePoint paylaşılan sənəd (adla şəxsiləşdirilir) | Sənəd paylaşımı |
| IT parol müddəti bitmə portalı (brend-müstəqil) | Bildiriş portalı |

**Privacy-by-design saxlanılır:** bütün səhifələr formaları `{{.SubmitURL}}`-ə göndərir — daxil edilən dəyərlər oxunmur/saxlanmır, yalnız göndərmə faktı qeyd olunur.

**Outlook uyğunluğu:** email şablonları Outlook-un (Word render mühərriki) tələblərinə görə qurulub — cədvəl-əsaslı struktur, inline stillər, `bgcolor` düymələr (VML/şərti şərhlər yox, çünki `html/template` HTML şərhlərini silir), preheader və «linki brauzerə kopyala» ehtiyat sətri. Eyni lure-lar `samples/library-*.eml` kimi də gəlir (multipart/alternative, UTF-8) — *Şablonlar → Email import* vasitəsilə idxal edilə və ya birbaşa Outlook-da açıla bilər.

### REST API nümunəsi
```bash
# İstifadəçilər səhifəsində API açarı yaradın, sonra:
curl -H "X-API-Key: <açar>" http://127.0.0.1:8443/api/campaigns
curl -H "X-API-Key: <açar>" http://127.0.0.1:8443/api/campaigns/<id>
```

## Memarlıq

```
                ┌──────────────────────────┐
  Sizin komanda │  ADMIN panel  :8443      │  ← VPN / localhost, GİZLİ
  (VPN) ───────▶│  müştəri, icazə, qrup,   │
                │  şablon, kampaniya, hesabat│
                └────────────┬─────────────┘
                             │  Postgres (izolə)
                ┌────────────┴─────────────┐
  İşçi/hədəf    │  PUBLIC server :8080     │  ← yalnız /t/ /p/ /s/
  (internet) ──▶│  landing + tracking      │
                └──────────────────────────┘
```

| Komponent | Paket |
|---|---|
| Admin HTTP | `internal/server` |
| Public (landing+tracking) | `internal/publicsrv` |
| Göndərmə mühərriki (SMTP) | `internal/engine` |
| Verilənlər bazası qatı | `internal/store` |
| Şifrələmə/parol/token | `internal/auth` |

---

## Tez başlanğıc (Docker — tövsiyə olunan yol)

> **Qeyd:** Windows-da Defender bəzən adında/məzmununda phishing olan icra fayllarını bloklaya bilər. Docker yolu bunu tamamilə aradan qaldırır (tətbiq Linux konteynerində işləyir) və eyni zamanda production mühitinə uyğundur.

1. **Sessiya açarı yaradın** (32 bayt):
   ```bash
   openssl rand -base64 32
   ```
2. `.env` faylını yaradın (`.env.example`-dən köçürün) və açarı yazın:
   ```
   PROPHISH_SESSION_KEY=<yuxarıdakı açar>
   PROPHISH_PUBLIC_BASE_URL=http://localhost:8080
   PROPHISH_TIMEZONE=Asia/Baku
   ```
   > `PROPHISH_TIMEZONE` planlaşdırma vaxtlarının və tarixlərin hansı saat qurşağında göstərilməsini təyin edir (defolt: UTC). Təyin etməsəniz, "09:00"-a planlaşdırılan kampaniya 09:00 UTC-də gedər.
3. Stacki qaldırın:
   ```bash
   docker compose up -d --build
   ```
4. Admin panelə daxil olun: <http://127.0.0.1:8443>
   İlk açılışda **ilk admin hesabını** yaradacaqsınız (parol ≥ 12 simvol).

Dayandırmaq: `docker compose down` · Məlumatı da silmək: `docker compose down -v`

---

## İş axını (addım-addım)

1. **Müştəri** əlavə edin (*Müştərilər*).
2. Müştəri səhifəsində **İcazə (Authorization/RoE)** qeyd edin — scope, imzalayan, sənəd istinadı, etibarlılıq tarixləri. *Bu olmadan kampaniya göndərilmir.*
3. **Hədəf qrupu** yaradın və işçiləri əlavə edin (tək-tək və ya **CSV import**: `email, first_name, last_name, position`).
4. **Email şablonu** yaradın. Dəyişənlər: `{{.FirstName}}`, `{{.LastName}}`, `{{.Email}}`, `{{.Position}}`, `{{.URL}}` (phishing linki), `{{.TrackingPixel}}` (açılma pikseli — qeyd etməsəniz avtomatik əlavə olunur).
5. **Landing səhifəsi** yaradın. Forma `action="{{.SubmitURL}}" method="post"` olmalıdır. (Dəyərlər saxlanılmır.)
6. **SMTP profili** əlavə edin (host, port, from, istifadəçi/parol).
7. **Kampaniya** yaradın: müştəri + icazə + qrup(lar) + şablon + səhifə + SMTP → *draft*.
8. Kampaniya səhifəsində **İşə sal**. Sistem icazəni yenidən yoxlayır, sonra göndərir. *İşə sal düyməsini iki dəfə bassanız belə email yalnız bir dəfə gedir.* Göndərmə boyu (xüsusən planlı/tədricən göndərmədə) icazənin etibarlılığı **hər mesajda** yenidən yoxlanır — icazə müddəti bitərsə göndərmə dərhal dayanır.
9. Lazım gəlsə **Göndərməni dayandır** düyməsi ilə işləyən kampaniyanı dərhal saxlaya bilərsiniz (hələ göndərilməmiş emaillar getməyəcək).
10. **Hesabat:** göndərildi / açıldı / kliklədi / məlumat göndərdi — say və faizlə.

---

## Təhlükəsizlik auditi (2026-11-07)

Çox-agentli audit (106 agent, hər tapıntı 3 skeptiklə təsdiqlənib) keçirildi və **təsdiqlənmiş bütün tapıntılar düzəldildi**:

- **SSRF (Yüksək):** `CloneSite` artıq dial vaxtı real IP-ni yoxlayır; loopback/private/link-local/metadata (169.254.169.254) rədd olunur (DNS-rebinding-ə qarşı).
- **Email CRLF injeksiyası:** bütün başlıq dəyərləri CR/LF-dən təmizlənir, Subject RFC 2047 ilə kodlanır.
- **Sessiya:** hər sorğuda DB-yə qarşı yoxlanır (deaktiv/silinmiş operator dərhal çıxarılır).
- **Rol icazəsi (RBAC):** SMTP/IMAP konfiqi, icazə yaratma, istifadəçi idarəsi, kampaniya silmə → yalnız **admin**.
- **CSRF:** SameSite=Lax üstünə Origin/Referer yoxlaması.
- **Login:** brute-force lockout + timing-sabit yoxlama.
- **Public server:** rate-limit + concurrency cap; XFF yalnız `PROPHISH_TRUSTED_PROXIES`-dən qəbul olunur.
- **CSV export:** formul (DDE) injeksiyası neytrallaşdırılır.
- **API açarları:** yalnız SHA-256 hash saxlanır (plaintext bir dəfə göstərilir).
- **IMAP:** STARTTLS məcburidir, oxuma limitli, RID yalnız üst başlıqdan oxunur və göndərici target ilə uyğunlaşdırılır.
- **Kripto açar ayrılması:** kredensial-şifrələmə açarı sessiya açarından HKDF ilə ayrılır.

**Təkrar audit (2-ci raund)** düzəlişlərdən sonra keçirildi: Kritik/Yüksək **yoxdur**, əvvəlki 15 tapıntı bağlı qaldı. Aşkar olunan yeni 4 Orta + 6 Aşağı da düzəldildi:
- admin router-dən `RealIP` çıxarıldı (login-lockout artıq XFF ilə keçilmir); hesab üzrə əlavə sayğac.
- public rate-limiter real müştəri IP-sinə görə açarlanır.
- open/click/submit event-ləri yalnız ilk keçiddə yazılır (DB-doldurma DoS bağlandı).
- **"reported" ayrıca bayraqdır** — `submitted`/`clicked` statusunu artıq üstələmir (işçi uğursuzluğunu "ağ yuya" bilmir).
- login/setup `checkOrigin` altındadır; `decodeKey` yalnız base64/hex 32 bayt qəbul edir; multipart temp fayllar silinir; parol dəyişmə cari parol tələb edir + sessiya rotasiyası; IMAP yalnız RID başlığını çəkir; Postgres parolu compose-da `PROPHISH_DB_PASSWORD` ilə parametrləşdirilib.

Qalan aşağı-riskli tövsiyələr: public server üçün ayrıca DB pool; event retention/arxiv job; reported üçün DKIM/DMARC uyğunluq yoxlaması (mail-auth infrastrukturu tələb edir); production-da `.env` açarlarının rotasiyası.

## Təhlükəsizlik yoxlama siyahısı (production)

- [ ] Admin paneli (`:8443`) **internetə açmayın** — VPN və ya SSH tunel arxasında saxlayın.
- [ ] Public serveri birbaşa yox, **TLS reverse proxy** (Nginx/Caddy) arxasında verin; yalnız `/t/`, `/p/`, `/s/` yollarını ötürün.
- [ ] `PROPHISH_SECURE_COOKIES=true` (HTTPS altında).
- [ ] Postgres portunu (`5432`) publik açmayın; disk şifrələməsi (at rest) tətbiq edin.
- [ ] `.env`-i heç vaxt git-ə əlavə etməyin (artıq `.gitignore`-dadır); güclü DB parolu seçin.
- [ ] Hər kampaniya üçün müştəri ilə imzalanmış **RoE/icazə** olsun və sistemdə qeyd edilsin.
- [ ] Data retention: kampaniya bitəndən sonra şəxsi dataları silmə siyasəti qurun.
- [ ] **Canlıya çıxmazdan əvvəl müstəqil təhlükəsizlik auditi** keçirin (bu, mühüm addımdır).

---

## Test

```bash
go test ./...
```
Testlər şablonların renderini, göndərmə mühərrikinin dəyişən birləşdirməsini və tracking pikselini, şifrələmə roundtrip-ini və landing renderini yoxlayır.

> Qeyd: Bəzi Windows mühitlərində Defender yeni qurulan test `.exe` fayllarını müvəqqəti bloklaya bilər (`Access is denied`). Bu, kod səhvi deyil; testləri Docker/Linux-da işlətmək problemi aradan qaldırır.

---

## Hüquqi qeyd

Bu alət **yalnız yazılı icazə** verilmiş təşkilatlarda, qanuni məlumatlandırma/pentest məqsədilə istifadə üçündür. İcazəsiz istifadə qanunsuzdur. İcazə qeydiyyatı və RoE prosesi məhz bunun üçün platformaya daxil edilib.
