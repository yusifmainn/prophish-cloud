package server

// SMTP sending-profiles screen (profiles.html): the page header, test/flash
// notes, the new-profile form, the profile list table, the per-row edit form
// and delete confirmation.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Page header ----
		"prof.title":    {"az": "SMTP göndərmə profilləri", "en": "SMTP sending profiles", "ru": "Профили отправки SMTP"},
		"prof.subtitle": {"az": "Emaillərin göndərildiyi poçt serveri.", "en": "The mail server used to send emails.", "ru": "Почтовый сервер, с которого отправляются письма."},

		// ---- Test / flash notes ----
		"prof.test_ok":    {"az": "Test emaili göndərildi.", "en": "Test email sent.", "ru": "Тестовое письмо отправлено."},
		"prof.test_err":   {"az": "Test emaili göndərilə bilmədi — host/port/parolu yoxlayın.", "en": "Could not send the test email — check host/port/password.", "ru": "Не удалось отправить тестовое письмо — проверьте хост/порт/пароль."},
		"prof.test_bad":   {"az": "Düzgün email ünvanı daxil edin.", "en": "Enter a valid email address.", "ru": "Введите корректный адрес электронной почты."},
		"prof.test_inuse": {"az": "Profil silinmədi — onu bir və ya bir neçə kampaniya istifadə edir. Əvvəlcə həmin kampaniyaları silin.", "en": "Profile not deleted — it is used by one or more campaigns. Delete those campaigns first.", "ru": "Профиль не удалён — его использует одна или несколько кампаний. Сначала удалите эти кампании."},

		// ---- New-profile form ----
		"prof.new":              {"az": "Yeni profil", "en": "New profile", "ru": "Новый профиль"},
		"prof.new_hint":         {"az": "Parol AES-256-GCM ilə şifrələnir. Xüsusi başlıqlar: hər sətir <code>Açar: Dəyər</code>.", "en": "The password is encrypted with AES-256-GCM. Custom headers: each line <code>Key: Value</code>.", "ru": "Пароль шифруется по AES-256-GCM. Пользовательские заголовки: каждая строка <code>Ключ: Значение</code>."},
		"prof.smtp_host":        {"az": "SMTP host", "en": "SMTP host", "ru": "SMTP-хост"},
		"prof.port":             {"az": "Port", "en": "Port", "ru": "Порт"},
		"prof.starttls_hint":    {"az": "STARTTLS dəstəklənir (adətən port <b>587</b> və ya 25). Port <b>465</b> (implicit TLS) dəstəklənmir.", "en": "STARTTLS is supported (usually port <b>587</b> or 25). Port <b>465</b> (implicit TLS) is not supported.", "ru": "Поддерживается STARTTLS (обычно порт <b>587</b> или 25). Порт <b>465</b> (implicit TLS) не поддерживается."},
		"prof.from_address":     {"az": "Göndərən ünvan (From)", "en": "Sender address (From)", "ru": "Адрес отправителя (From)"},
		"prof.from_placeholder": {"az": "IT Dəstək <it@example.com>", "en": "IT Support <it@example.com>", "ru": "ИТ-поддержка <it@example.com>"},
		"prof.smtp_user":        {"az": "SMTP istifadəçi (opsional)", "en": "SMTP username (optional)", "ru": "Имя пользователя SMTP (опционально)"},
		"prof.smtp_password":    {"az": "SMTP parol (opsional)", "en": "SMTP password (optional)", "ru": "Пароль SMTP (опционально)"},
		"prof.headers_optional": {"az": "Xüsusi başlıqlar (opsional)", "en": "Custom headers (optional)", "ru": "Пользовательские заголовки (опционально)"},

		// ---- Profile list table ----
		"prof.host":    {"az": "Host", "en": "Host", "ru": "Хост"},
		"prof.from":    {"az": "From", "en": "From", "ru": "Отправитель"},
		"prof.auth":    {"az": "Auth", "en": "Auth", "ru": "Аутент."},
		"prof.test":    {"az": "Test", "en": "Test", "ru": "Тест"},
		"prof.actions": {"az": "Əməliyyat", "en": "Action", "ru": "Действие"},

		"prof.pill_has_pw":  {"az": "Parol var", "en": "Has password", "ru": "Есть пароль"},
		"prof.pill_no_pw":   {"az": "Parolsuz", "en": "No password", "ru": "Без пароля"},
		"prof.pill_no_auth": {"az": "Auth yoxdur", "en": "No auth", "ru": "Без аутент."},

		"prof.test_to_ph":   {"az": "nümunə@şirkət.az", "en": "example@company.com", "ru": "пример@компания.ru"},
		"prof.test_to_aria": {"az": "Test email ünvanı", "en": "Test email address", "ru": "Адрес для тестового письма"},
		"prof.send":         {"az": "Göndər", "en": "Send", "ru": "Отправить"},

		// ---- Per-row edit form ----
		"prof.smtp_password_edit": {"az": "SMTP parol (boş = dəyişmə)", "en": "SMTP password (empty = unchanged)", "ru": "Пароль SMTP (пусто = без изменений)"},
		"prof.clear_password":     {"az": "Saxlanmış parolu təmizlə", "en": "Clear saved password", "ru": "Очистить сохранённый пароль"},
		"prof.headers_label":      {"az": "Xüsusi başlıqlar (hər sətir <code>Açar: Dəyər</code>)", "en": "Custom headers (each line <code>Key: Value</code>)", "ru": "Пользовательские заголовки (каждая строка <code>Ключ: Значение</code>)"},

		// ---- Delete / empty ----
		"prof.delete_confirm": {"az": "'%s' profili silinsin?", "en": "Delete profile '%s'?", "ru": "Удалить профиль '%s'?"},
		"prof.empty":          {"az": "Hələ profil yoxdur.", "en": "No profiles yet.", "ru": "Профилей пока нет."},
	})
}
