package server

// IMAP settings screen (imap.html): the mailbox configuration form and the
// list of reported (forwarded) emails.
func init() {
	addMessages(map[string]map[string]string{
		"imapui.title":   {"az": "Parametrlər — Bildirilən emaillər (IMAP)", "en": "Settings — Reported emails (IMAP)", "ru": "Настройки — Сообщённые письма (IMAP)"},
		"imapui.mailbox": {"az": "IMAP qutusu", "en": "IMAP mailbox", "ru": "Почтовый ящик IMAP"},
		"imapui.mailbox_hint": {
			"az": "İşçilərin \"şübhəli\" deyə bildirdiyi (forward etdiyi) emailləri bu qutudan oxuyuruq. Göndərilən emaillərdə gizli <code>X-PROPhish-RID</code> başlığı olduğu üçün bildirən işçi avtomatik kampaniyaya bağlanır.",
			"en": "We read from this mailbox the emails that employees report (forward) as \"suspicious\". Because sent emails carry a hidden <code>X-PROPhish-RID</code> header, the reporting employee is linked to the campaign automatically.",
			"ru": "Из этого ящика мы читаем письма, которые сотрудники помечают (пересылают) как «подозрительные». Поскольку отправленные письма содержат скрытый заголовок <code>X-PROPhish-RID</code>, сообщивший сотрудник автоматически привязывается к кампании.",
		},
		"imapui.host":          {"az": "Host", "en": "Host", "ru": "Хост"},
		"imapui.port":          {"az": "Port", "en": "Port", "ru": "Порт"},
		"imapui.username":      {"az": "İstifadəçi", "en": "User", "ru": "Пользователь"},
		"imapui.password":      {"az": "Parol (dəyişmək üçün doldurun)", "en": "Password (fill in to change)", "ru": "Пароль (заполните, чтобы изменить)"},
		"imapui.folder":        {"az": "Qovluq", "en": "Folder", "ru": "Папка"},
		"imapui.poll_interval": {"az": "Yoxlama aralığı (saniyə)", "en": "Poll interval (seconds)", "ru": "Интервал опроса (секунды)"},
		"imapui.active":        {"az": "Aktiv", "en": "Active", "ru": "Активно"},

		"imapui.reported_title": {"az": "Bildirilən emaillər", "en": "Reported emails", "ru": "Сообщённые письма"},
		"imapui.col_time":       {"az": "Vaxt", "en": "Time", "ru": "Время"},
		"imapui.col_reporter":   {"az": "Bildirən", "en": "Reporter", "ru": "Отправитель сообщения"},
		"imapui.col_subject":    {"az": "Mövzu", "en": "Subject", "ru": "Тема"},
		"imapui.empty":          {"az": "Hələ bildiriş yoxdur.", "en": "No reports yet.", "ru": "Сообщений пока нет."},
	})
}
