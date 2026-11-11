package server

// Email templates list screen (templates.html): the new-template and .eml
// import forms, the variable/header hints, and the templates table.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Page header ----
		"tpl.title":    {"az": "Email şablonları", "en": "Email templates", "ru": "Email-шаблоны"},
		"tpl.subtitle": {"az": "Hədəf işçilərə göndərilən simulyasiya emailləri. Vizual redaktor və ya .eml import.", "en": "Simulation emails sent to target employees. Visual editor or .eml import.", "ru": "Имитационные письма, отправляемые целевым сотрудникам. Визуальный редактор или импорт .eml."},

		// ---- Flash / error notices ----
		"tpl.import_ok":  {"az": "Email import olundu.", "en": "Email imported.", "ru": "Письмо импортировано."},
		"tpl.import_err": {"az": "Email oxuna bilmədi — formatı yoxlayın.", "en": "Could not read the email — check the format.", "ru": "Не удалось прочитать письмо — проверьте формат."},
		"tpl.err_inuse":  {"az": "Şablon silinmədi — onu bir və ya bir neçə kampaniya istifadə edir. Əvvəlcə həmin kampaniyaları silin.", "en": "The template was not deleted — one or more campaigns use it. Delete those campaigns first.", "ru": "Шаблон не удалён — его использует одна или несколько кампаний. Сначала удалите эти кампании."},

		// ---- New template form ----
		"tpl.new":          {"az": "Yeni şablon", "en": "New template", "ru": "Новый шаблон"},
		"tpl.variables":    {"az": "Dəyişənlər:", "en": "Variables:", "ru": "Переменные:"},
		"tpl.subject":      {"az": "Mövzu (Subject)", "en": "Subject", "ru": "Тема"},
		"tpl.html_content": {"az": "HTML məzmun", "en": "HTML content", "ru": "HTML-содержимое"},
		"tpl.text_version": {"az": "Mətn versiyası (opsional)", "en": "Text version (optional)", "ru": "Текстовая версия (необязательно)"},

		// ---- Default sample email body (prefilled in the editor) ----
		"tpl.sample_greeting": {"az": "Salam", "en": "Hello", "ru": "Здравствуйте,"},
		"tpl.sample_confirm":  {"az": "Hesabınızı təsdiqləyin:", "en": "Confirm your account:", "ru": "Подтвердите вашу учётную запись:"},
		"tpl.sample_click":    {"az": "buraya klikləyin", "en": "click here", "ru": "нажмите здесь"},

		// ---- Custom headers hint ----
		"tpl.custom_headers_pre": {"az": "Xüsusi başlıqlar (opsional, hər sətir", "en": "Custom headers (optional, one per line", "ru": "Пользовательские заголовки (необязательно, по одному в строке"},
		"tpl.key_value":          {"az": "Açar: Dəyər", "en": "Key: Value", "ru": "Ключ: Значение"},

		// ---- .eml / raw import form ----
		"tpl.import_title": {"az": "Email import (.eml / xam mətn)", "en": "Email import (.eml / raw text)", "ru": "Импорт письма (.eml / сырой текст)"},
		"tpl.import_hint":  {"az": "Mövcud bir emaili olduğu kimi yapışdırın — mövzu və HTML/mətn avtomatik çıxarılacaq.", "en": "Paste an existing email as-is — the subject and HTML/text will be extracted automatically.", "ru": "Вставьте существующее письмо как есть — тема и HTML/текст будут извлечены автоматически."},
		"tpl.raw_email":    {"az": "Xam email mətni", "en": "Raw email text", "ru": "Сырой текст письма"},
		"tpl.import_btn":   {"az": "Import et", "en": "Import", "ru": "Импортировать"},

		// ---- Templates table ----
		"tpl.col_subject": {"az": "Mövzu", "en": "Subject", "ru": "Тема"},
		"tpl.empty":       {"az": "Hələ şablon yoxdur.", "en": "No templates yet.", "ru": "Шаблонов пока нет."},
	})
}
