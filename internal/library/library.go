// Package library is a built-in, read-only catalog of ready-made phishing
// simulation "kits": high-fidelity recreations of the login pages real
// attackers imitate (Microsoft 365, Outlook Web App, Google, OneDrive) plus a
// generic IT password-expiry portal, each paired with a matching email lure.
//
// The pages are faithful to the real brand UI on purpose — a training exercise
// only measures a real risk if the decoy looks like what employees actually
// see. In keeping with the platform's privacy-by-design rule, every landing
// form posts to {{.SubmitURL}} and no field value is ever read or stored; only
// the fact of submission is recorded.
//
// Kits are templates, not live objects: an operator instantiates one with
// "Use", which copies its HTML into an ordinary, fully editable landing page
// (and email template) in the database. The catalog itself is versioned with
// the binary, so fresh installs ship with content and upgrades refresh it.
package library

import "embed"

//go:embed assets/*.html
var assets embed.FS

func mustRead(name string) string {
	b, err := assets.ReadFile("assets/" + name)
	if err != nil {
		// A missing asset is a build/packaging error, not a runtime condition.
		panic("library: embedded asset not found: " + name)
	}
	return string(b)
}

// Kit is one catalog entry. Title/Summary/EmailSubject are localized (az/en/ru);
// the landing and email HTML are brand-faithful and intentionally not localized
// (the login screens employees really see are in the brand's own wording).
type Kit struct {
	ID       string // stable slug, used in URLs
	Brand    string // display brand, e.g. "Microsoft 365"
	Category string // "login" | "document" | "notice"

	Title   map[string]string // short display name per language
	Summary map[string]string // one-line description per language

	RedirectURL string // where the landing form lands after submit ("" = built-in awareness notice)

	pageFile string // embedded landing page asset

	// Email lure (optional). HasEmail reports whether EmailSubject/EmailHTML apply.
	EmailSubject map[string]string
	emailFile    string
}

// LandingHTML returns the kit's landing page markup (a Go html/template body
// that uses {{.FirstName}}, {{.SubmitURL}}, …).
func (k Kit) LandingHTML() string { return mustRead(k.pageFile) }

// HasEmail reports whether the kit ships a matching email template.
func (k Kit) HasEmail() bool { return k.emailFile != "" }

// EmailHTML returns the kit's email body ("" when the kit has no email).
func (k Kit) EmailHTML() string {
	if k.emailFile == "" {
		return ""
	}
	return mustRead(k.emailFile)
}

// catalog is the ordered list shown in the gallery (most-used brands first).
var catalog = []Kit{
	{
		ID:       "m365",
		Brand:    "Microsoft 365",
		Category: "login",
		Title:    map[string]string{"az": "Microsoft 365 girişi", "en": "Microsoft 365 sign-in", "ru": "Вход Microsoft 365"},
		Summary: map[string]string{
			"az": "Azure AD / Entra iki-mərhələli giriş ekranı + «qeyri-adi giriş» xəbərdarlıq məktubu.",
			"en": "Azure AD / Entra two-step sign-in screen + an «unusual sign-in» alert email.",
			"ru": "Двухэтапный экран входа Azure AD / Entra + письмо о «необычном входе».",
		},
		pageFile:  "m365.page.html",
		emailFile: "m365.email.html",
		EmailSubject: map[string]string{
			"az": "Microsoft hesabınızda qeyri-adi giriş fəaliyyəti",
			"en": "Unusual sign-in activity on your Microsoft account",
			"ru": "Необычный вход в вашу учётную запись Microsoft",
		},
	},
	{
		ID:       "owa",
		Brand:    "Outlook Web App",
		Category: "login",
		Title:    map[string]string{"az": "Outlook Web App girişi", "en": "Outlook Web App sign-in", "ru": "Вход Outlook Web App"},
		Summary: map[string]string{
			"az": "Klassik Exchange / OWA korporativ giriş səhifəsi + «poçt qutusu doludur» məktubu.",
			"en": "Classic Exchange / OWA corporate sign-in page + a «mailbox full» email.",
			"ru": "Классическая корпоративная страница входа Exchange / OWA + письмо «почтовый ящик переполнен».",
		},
		pageFile:  "owa.page.html",
		emailFile: "owa.email.html",
		EmailSubject: map[string]string{
			"az": "Poçt qutunuz dolmaq üzrədir — təsdiqləyin",
			"en": "Your mailbox is almost full — please verify",
			"ru": "Ваш почтовый ящик почти переполнен — подтвердите",
		},
	},
	{
		ID:       "google",
		Brand:    "Google Workspace",
		Category: "login",
		Title:    map[string]string{"az": "Google girişi", "en": "Google sign-in", "ru": "Вход Google"},
		Summary: map[string]string{
			"az": "Google Workspace / Gmail hesab giriş ekranı + təhlükəsizlik xəbərdarlığı məktubu.",
			"en": "Google Workspace / Gmail account sign-in screen + a security-alert email.",
			"ru": "Экран входа в аккаунт Google Workspace / Gmail + письмо с предупреждением о безопасности.",
		},
		pageFile:  "google.page.html",
		emailFile: "google.email.html",
		EmailSubject: map[string]string{
			"az": "Təhlükəsizlik bildirişi: hesabınıza yeni giriş",
			"en": "Security alert: new sign-in to your account",
			"ru": "Оповещение системы безопасности: новый вход в аккаунт",
		},
	},
	{
		ID:       "onedrive",
		Brand:    "OneDrive / SharePoint",
		Category: "document",
		Title:    map[string]string{"az": "OneDrive paylaşılan sənəd", "en": "OneDrive shared document", "ru": "Общий документ OneDrive"},
		Summary: map[string]string{
			"az": "«Sizinlə sənəd paylaşıldı» səhifəsi (adla şəxsiləşdirilir) + paylaşım dəvəti məktubu.",
			"en": "A «document shared with you» page (personalized by name) + a sharing-invite email.",
			"ru": "Страница «с вами поделились документом» (с именем) + письмо-приглашение к доступу.",
		},
		pageFile:  "onedrive.page.html",
		emailFile: "onedrive.email.html",
		EmailSubject: map[string]string{
			"az": "{{.FirstName}}, sizinlə sənəd paylaşıldı",
			"en": "{{.FirstName}}, a document was shared with you",
			"ru": "{{.FirstName}}, с вами поделились документом",
		},
	},
	{
		ID:       "password",
		Brand:    "IT Password Portal",
		Category: "notice",
		Title:    map[string]string{"az": "Parol müddəti bitmə portalı", "en": "Password-expiry portal", "ru": "Портал истечения пароля"},
		Summary: map[string]string{
			"az": "Brend-müstəqil «parolunuz bu gün bitir» özünəxidmət portalı + IT xəbərdarlıq məktubu.",
			"en": "A brand-neutral «your password expires today» self-service portal + an IT notice email.",
			"ru": "Нейтральный портал самообслуживания «пароль истекает сегодня» + письмо от ИТ-службы.",
		},
		pageFile:  "password.page.html",
		emailFile: "password.email.html",
		EmailSubject: map[string]string{
			"az": "Parolunuzun müddəti bu gün bitir",
			"en": "Your password expires today",
			"ru": "Срок действия вашего пароля истекает сегодня",
		},
	},
}

// Catalog returns the ordered list of available kits.
func Catalog() []Kit { return catalog }

// Get returns the kit with the given ID.
func Get(id string) (Kit, bool) {
	for _, k := range catalog {
		if k.ID == id {
			return k, true
		}
	}
	return Kit{}, false
}

// Pick resolves a localized field for lang, falling back to Azerbaijani (the
// source language) and then to any present value, so a card never renders blank.
func Pick(m map[string]string, lang string) string {
	if m == nil {
		return ""
	}
	if s := m[lang]; s != "" {
		return s
	}
	if s := m["az"]; s != "" {
		return s
	}
	for _, s := range m {
		if s != "" {
			return s
		}
	}
	return ""
}
