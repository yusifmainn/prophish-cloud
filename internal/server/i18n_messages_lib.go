package server

// Template & landing-page library (library.html) and its nav entry / flashes.
func init() {
	addMessages(map[string]map[string]string{
		"nav.library": {"az": "Kitabxana", "en": "Library", "ru": "Библиотека"},

		"lib.title": {"az": "Şablon kitabxanası", "en": "Template library", "ru": "Библиотека шаблонов"},
		"lib.subtitle": {
			"az": "Real brend giriş səhifələrinə birebir bənzəyən hazır dəstlər. «İstifadə et» ilə redaktə oluna bilən açılış səhifəsi və email şablonuna çevrilir.",
			"en": "Ready-made kits that faithfully mirror real brand sign-in pages. «Use» turns one into an editable landing page and email template.",
			"ru": "Готовые наборы, точно повторяющие реальные страницы входа брендов. «Использовать» создаёт редактируемую целевую страницу и шаблон письма.",
		},
		"lib.privacy_note": {
			"az": "Bütün səhifələr formaları <code>{{.SubmitURL}}</code>-ə göndərir — daxil edilən dəyərlər (istifadəçi adı, parol) <strong>oxunmur və saxlanmır</strong>; yalnız göndərmə faktı qeyd olunur.",
			"en": "Every page posts its form to <code>{{.SubmitURL}}</code> — entered values (username, password) are <strong>never read or stored</strong>; only the fact of submission is recorded.",
			"ru": "Каждая страница отправляет форму на <code>{{.SubmitURL}}</code> — введённые значения (логин, пароль) <strong>не читаются и не сохраняются</strong>; фиксируется лишь факт отправки.",
		},
		"lib.badge_email":   {"az": "+ email şablonu", "en": "+ email template", "ru": "+ шаблон письма"},
		"lib.cat_login":     {"az": "Giriş səhifəsi", "en": "Login page", "ru": "Страница входа"},
		"lib.cat_document":  {"az": "Sənəd paylaşımı", "en": "Document share", "ru": "Доступ к документу"},
		"lib.cat_notice":    {"az": "Bildiriş portalı", "en": "Notice portal", "ru": "Портал уведомлений"},
		"lib.preview_page":  {"az": "Səhifəyə bax", "en": "Preview page", "ru": "Просмотр страницы"},
		"lib.preview_email": {"az": "Məktuba bax", "en": "Preview email", "ru": "Просмотр письма"},
		"lib.use_btn":       {"az": "İstifadə et", "en": "Use", "ru": "Использовать"},
		"lib.empty":         {"az": "Kitabxana boşdur.", "en": "The library is empty.", "ru": "Библиотека пуста."},
		"lib.goto": {
			"az": "Sıfırdan yazmaq istəmirsiniz? Hazır brend şablonları üçün → <a href=\"/library\">Kitabxana</a>",
			"en": "Don't want to start from scratch? Ready-made brand kits → <a href=\"/library\">Library</a>",
			"ru": "Не хотите начинать с нуля? Готовые наборы брендов → <a href=\"/library\">Библиотека</a>",
		},

		// Flash toasts assembled with s.tf (one %s = the kit name).
		"lib.flash_used_both": {
			"az": "«%s» kitabxanadan əlavə edildi: açılış səhifəsi və email şablonu yaradıldı.",
			"en": "«%s» added from the library: a landing page and an email template were created.",
			"ru": "«%s» добавлено из библиотеки: созданы целевая страница и шаблон письма.",
		},
		"lib.flash_used_page": {
			"az": "«%s» açılış səhifəsi kitabxanadan əlavə edildi.",
			"en": "«%s» landing page was added from the library.",
			"ru": "Целевая страница «%s» добавлена из библиотеки.",
		},

		// Audit-log label for the "Use" action.
		"action.library.use": {"az": "Kitabxanadan əlavə", "en": "Added from library", "ru": "Добавлено из библиотеки"},
	})
}
