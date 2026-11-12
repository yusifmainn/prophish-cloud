package server

// Landing pages list (pages.html).
func init() {
	addMessages(map[string]map[string]string{
		"pg.title":            {"az": "Açılış səhifələri", "en": "Landing pages", "ru": "Целевые страницы"},
		"pg.subtitle":         {"az": "İşçi linkə klikləyəndən sonra gördüyü səhifə. Dəyərlər saxlanmır.", "en": "The page an employee sees after clicking the link. Values are not stored.", "ru": "Страница, которую видит сотрудник после перехода по ссылке. Значения не сохраняются."},
		"pg.import_ok":        {"az": "Sayt klonlandı.", "en": "Site cloned.", "ru": "Сайт клонирован."},
		"pg.import_err":       {"az": "Sayt yüklənə bilmədi — URL-i yoxlayın.", "en": "The site could not be fetched — check the URL.", "ru": "Не удалось загрузить сайт — проверьте URL."},
		"pg.clone_title":      {"az": "URL-dən klonla", "en": "Clone from URL", "ru": "Клонировать из URL"},
		"pg.clone_desc":       {"az": `Mövcud bir səhifəni çəkib açılış səhifəsinə çevirir. Formalar avtomatik <code>{{.SubmitURL}}</code>-ə yönəldilir; dəyərlər saxlanmır.`, "en": `Fetches an existing page and turns it into a landing page. Forms are automatically pointed at <code>{{.SubmitURL}}</code>; values are not stored.`, "ru": `Загружает существующую страницу и превращает её в целевую. Формы автоматически направляются на <code>{{.SubmitURL}}</code>; значения не сохраняются.`},
		"pg.redirect_opt":     {"az": "Yönləndirmə URL (opsional)", "en": "Redirect URL (optional)", "ru": "URL перенаправления (необязательно)"},
		"pg.clone_btn":        {"az": "Klonla", "en": "Clone", "ru": "Клонировать"},
		"pg.manual_title":     {"az": "Yeni səhifə (əl ilə)", "en": "New page (manual)", "ru": "Новая страница (вручную)"},
		"pg.manual_desc":      {"az": `Forma <code>action="{{.SubmitURL}}"</code> və <code>method="post"</code> olmalıdır. Dəyərlər <strong>saxlanılmır</strong>.`, "en": `The form must use <code>action="{{.SubmitURL}}"</code> and <code>method="post"</code>. Values are <strong>not stored</strong>.`, "ru": `Форма должна использовать <code>action="{{.SubmitURL}}"</code> и <code>method="post"</code>. Значения <strong>не сохраняются</strong>.`},
		"pg.html_label":       {"az": "HTML məzmun", "en": "HTML content", "ru": "HTML-содержимое"},
		"pg.redirect_col":     {"az": "Yönləndirmə", "en": "Redirect", "ru": "Перенаправление"},
		"pg.empty":            {"az": "Hələ səhifə yoxdur.", "en": "No pages yet.", "ru": "Пока нет страниц."},
		"pg.delete_confirm":   {"az": "Seçilmiş səhifələr silinəcək. Kampaniyada istifadə olunanlar silinməyəcək. Davam edilsin?", "en": "The selected pages will be deleted. Those used by a campaign will not be deleted. Continue?", "ru": "Выбранные страницы будут удалены. Используемые в кампании удалены не будут. Продолжить?"},
		"pg.builder_link":     {"az": "Vizual redaktorla yarat", "en": "Create in visual editor", "ru": "Создать в визуальном редакторе"},
		"pg.builder_hint_pre": {"az": "HTML bilmədən yığmaq üçün", "en": "To build without HTML, open", "ru": "Чтобы собрать без HTML, откройте"},
		"pg.builder_link2":    {"az": "vizual redaktoru", "en": "the visual editor", "ru": "визуальный редактор"},
	})
}
