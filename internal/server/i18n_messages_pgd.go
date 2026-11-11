package server

// Landing page detail (page_detail.html).
func init() {
	addMessages(map[string]map[string]string{
		"pgd.back":                {"az": "← Açılış səhifələri", "en": "← Landing pages", "ru": "← Целевые страницы"},
		"pgd.edit_title":          {"az": "Səhifəni redaktə et", "en": "Edit page", "ru": "Редактировать страницу"},
		"pgd.edit_hint":           {"az": `Forma <code>action="{{.SubmitURL}}"</code> və <code>method="post"</code> olmalıdır — yoxsa “göndərdi” hadisəsi qeyd olunmaz. Dəyişənlər: <code>{{.FirstName}}</code>, <code>{{.LastName}}</code>, <code>{{.Position}}</code>, <code>{{.SubmitURL}}</code>.`, "en": `The form must use <code>action="{{.SubmitURL}}"</code> and <code>method="post"</code> — otherwise the “submitted” event is not recorded. Variables: <code>{{.FirstName}}</code>, <code>{{.LastName}}</code>, <code>{{.Position}}</code>, <code>{{.SubmitURL}}</code>.`, "ru": `Форма должна использовать <code>action="{{.SubmitURL}}"</code> и <code>method="post"</code> — иначе событие «отправил» не будет записано. Переменные: <code>{{.FirstName}}</code>, <code>{{.LastName}}</code>, <code>{{.Position}}</code>, <code>{{.SubmitURL}}</code>.`},
		"pgd.html_label":          {"az": "HTML məzmun", "en": "HTML content", "ru": "HTML-содержимое"},
		"pgd.redirect_opt":        {"az": "Yönləndirmə URL (opsional)", "en": "Redirect URL (optional)", "ru": "URL перенаправления (необязательно)"},
		"pgd.preview_title":       {"az": "Önizləmə", "en": "Preview", "ru": "Предпросмотр"},
		"pgd.preview_hint":        {"az": "Nümunə işçi məlumatı ilə render olunur (Aysel Məmmədova · Mühasib). Dəyişiklikdən sonra yenilənir.", "en": "Rendered with sample employee data (Aysel Məmmədova · Accountant). Updates after you make changes.", "ru": "Отображается с примером данных сотрудника (Aysel Məmmədova · Бухгалтер). Обновляется после изменений."},
		"pgd.preview_iframe":      {"az": "Landing önizləmə", "en": "Landing preview", "ru": "Предпросмотр целевой страницы"},
		"pgd.delete_note":         {"az": "Səhifəni bir kampaniya istifadə edirsə silinə bilməz (əvvəlcə həmin kampaniyaları silin).", "en": "A page cannot be deleted while a campaign uses it (delete those campaigns first).", "ru": "Страницу нельзя удалить, пока её использует кампания (сначала удалите эти кампании)."},
		"pgd.delete_confirm_pre":  {"az": "'", "en": "Permanently delete the landing page '", "ru": "Удалить целевую страницу «"},
		"pgd.delete_confirm_post": {"az": "' açılış səhifəsi həmişəlik silinsin?", "en": "'?", "ru": "» навсегда?"},
		"pgd.delete_btn":          {"az": "Səhifəni sil", "en": "Delete page", "ru": "Удалить страницу"},
		"pgd.builder_link":        {"az": "Vizual redaktorda aç", "en": "Open in visual editor", "ru": "Открыть в визуальном редакторе"},
	})
}
