package server

// Handler-side strings that cannot use the flash reverse-lookup: dynamic flash
// messages that interpolate a value (translated via s.tf at the call site) and
// the login/setup inline errors (some of which were originally English literals,
// so they are keyed rather than reverse-translated). See handlers_actions.go,
// handlers_groups.go, handlers_content.go, handlers_pages.go, handlers_auth.go.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Dynamic flash messages (interpolated; used via s.tf) ----
		"flash.scheduled_at":       {"az": "Kampaniya %s tarixinə planlaşdırıldı.", "en": "Campaign scheduled for %s.", "ru": "Кампания запланирована на %s."},
		"flash.requeued_unopened":  {"az": "%d nəfər (emaili açmayanlar) yenidən növbəyə alındı.", "en": "%d recipient(s) who hadn't opened the email were re-queued.", "ru": "%d адресат(ов), не открывших письмо, повторно поставлены в очередь."},
		"flash.targets_added_dup":  {"az": "%d hədəf əlavə edildi, %d təkrar/dublikat ötürüldü.", "en": "%d target(s) added, %d duplicate(s) skipped.", "ru": "Добавлено целей: %d, пропущено дубликатов: %d."},
		"flash.targets_added":      {"az": "%d hədəf əlavə edildi.", "en": "%d target(s) added.", "ru": "Добавлено целей: %d."},
		"flash.page_save_tmpl_err": {"az": "Səhifə saxlanmadı — şablon xətası (ehtimal ki, mətndə artıq {{ }} var): %s", "en": "Page not saved — template error (the text probably already contains {{ }}): %s", "ru": "Страница не сохранена — ошибка шаблона (вероятно, в тексте уже есть {{ }}): %s"},
		"flash.html_save_tmpl_err": {"az": "HTML saxlanmadı — şablon xətası (ehtimal ki, mətndə artıq {{ }} var): %s", "en": "HTML not saved — template error (the text probably already contains {{ }}): %s", "ru": "HTML не сохранён — ошибка шаблона (вероятно, в тексте уже есть {{ }}): %s"},

		// ---- Login / setup inline errors (keyed; shown via data["Error"]) ----
		"err.email_required":      {"az": "Email tələb olunur.", "en": "Email is required.", "ru": "Требуется email."},
		"err.create_user":         {"az": "İstifadəçi yaradıla bilmədi.", "en": "Could not create user.", "ru": "Не удалось создать пользователя."},
		"err.invalid_credentials": {"az": "Yanlış giriş məlumatları.", "en": "Invalid credentials.", "ru": "Неверные учётные данные."},
		"err.too_many_attempts":   {"az": "Çox sayda uğursuz cəhd. Bir az sonra yenidən cəhd edin.", "en": "Too many failed attempts. Try again later.", "ru": "Слишком много неудачных попыток. Повторите позже."},
	})
}
