package server

// Email template detail screen (template_detail.html): attachments list and
// upload, the edit form, the preview panel, and the danger zone.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Header ----
		"tpld.subject_label": {"az": "Mövzu:", "en": "Subject:", "ru": "Тема:"},

		// ---- Attachments ----
		"tpld.attachments":           {"az": "Əlavələr (attachments)", "en": "Attachments", "ru": "Вложения"},
		"tpld.col_file":              {"az": "Fayl", "en": "File", "ru": "Файл"},
		"tpld.col_type":              {"az": "Tip", "en": "Type", "ru": "Тип"},
		"tpld.attach_delete_confirm": {"az": "'%s' əlavəsi silinsin?", "en": "Delete attachment '%s'?", "ru": "Удалить вложение '%s'?"},
		"tpld.no_attachments":        {"az": "Əlavə yoxdur.", "en": "No attachments.", "ru": "Вложений нет."},
		"tpld.upload":                {"az": "Yüklə", "en": "Upload", "ru": "Загрузить"},

		// ---- Edit form ----
		"tpld.edit_title":         {"az": "Şablonu redaktə et", "en": "Edit template", "ru": "Редактировать шаблон"},
		"tpld.variables":          {"az": "Dəyişənlər:", "en": "Variables:", "ru": "Переменные:"},
		"tpld.subject":            {"az": "Mövzu (Subject)", "en": "Subject", "ru": "Тема"},
		"tpld.html_content":       {"az": "HTML məzmun", "en": "HTML content", "ru": "HTML-содержимое"},
		"tpld.text_version":       {"az": "Mətn versiyası (opsional)", "en": "Text version (optional)", "ru": "Текстовая версия (необязательно)"},
		"tpld.custom_headers_pre": {"az": "Xüsusi başlıqlar (opsional, hər sətir", "en": "Custom headers (optional, one per line", "ru": "Пользовательские заголовки (необязательно, по одному в строке"},
		"tpld.key_value":          {"az": "Açar: Dəyər", "en": "Key: Value", "ru": "Ключ: Значение"},

		// ---- Preview ----
		"tpld.preview":       {"az": "Önizləmə", "en": "Preview", "ru": "Предпросмотр"},
		"tpld.preview_hint":  {"az": "Nümunə işçi məlumatı ilə — dəyişənlər doldurulmuş halda göstərilir.", "en": "Shown with sample employee data — variables filled in.", "ru": "Показано с примерными данными сотрудника — переменные заполнены."},
		"tpld.preview_title": {"az": "Email önizləmə", "en": "Email preview", "ru": "Предпросмотр письма"},

		// ---- Danger zone ----
		"tpld.danger_hint":    {"az": "Şablonu bir kampaniya istifadə edirsə silinə bilməz (əvvəlcə həmin kampaniyanı silin).", "en": "A template cannot be deleted while a campaign uses it (delete that campaign first).", "ru": "Шаблон нельзя удалить, пока его использует кампания (сначала удалите эту кампанию)."},
		"tpld.delete_confirm": {"az": "'%s' şablonu həmişəlik silinsin?", "en": "Permanently delete template '%s'?", "ru": "Удалить шаблон '%s' навсегда?"},
		"tpld.delete_btn":     {"az": "Şablonu sil", "en": "Delete template", "ru": "Удалить шаблон"},
	})
}
