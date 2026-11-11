package server

// Visual landing-page builder strings: the page chrome (builder.*, used via
// {{t}} in page_builder.html) and the client-side editor strings
// (builder.js.*, collected by messagesForPrefix and handed to builder.js as
// JSON, because a static JS file cannot call {{t}} itself).
func init() {
	addMessages(map[string]map[string]string{
		// ---- Page chrome (template) ----
		"builder.back":        {"az": "← Geri", "en": "← Back", "ru": "← Назад"},
		"builder.name_ph":     {"az": "Səhifə adı", "en": "Page name", "ru": "Название страницы"},
		"builder.snap":        {"az": "Söykənmə (5px)", "en": "Snap (5px)", "ru": "Привязка (5px)"},
		"builder.undo_tip":    {"az": "Geri al (Ctrl+Z)", "en": "Undo (Ctrl+Z)", "ru": "Отменить (Ctrl+Z)"},
		"builder.redo_tip":    {"az": "Təkrarla (Ctrl+Y)", "en": "Redo (Ctrl+Y)", "ru": "Повторить (Ctrl+Y)"},
		"builder.preview":     {"az": "Önizləmə", "en": "Preview", "ru": "Предпросмотр"},
		"builder.preview_tip": {"az": "Yeni vərəqdə tam işlək önizləmə (JS aktiv)", "en": "Full working preview in a new tab (JS enabled)", "ru": "Полный рабочий предпросмотр в новой вкладке (JS включён)"},
		"builder.create":      {"az": "Səhifəni yarat", "en": "Create page", "ru": "Создать страницу"},
		"builder.notice":      {"az": "Bu səhifə vizual redaktorla yaradılmayıb — kanvas boş başlayır.", "en": "This page was not created with the visual builder — the canvas starts empty.", "ru": "Эта страница создана не в визуальном редакторе — холст начинается пустым."},

		// ---- Toolbar tools (template) ----
		"builder.tool.select": {"az": "Seçim", "en": "Select", "ru": "Выбор"},
		"builder.tool.rect":   {"az": "Düzbucaqlı", "en": "Rectangle", "ru": "Прямоугольник"},
		"builder.tool.text":   {"az": "Mətn", "en": "Text", "ru": "Текст"},
		"builder.tool.button": {"az": "Düymə", "en": "Button", "ru": "Кнопка"},
		"builder.tool.image":  {"az": "Şəkil", "en": "Image", "ru": "Изображение"},
		"builder.tool.form":   {"az": "Form", "en": "Form", "ru": "Форма"},
		"builder.tool.line":   {"az": "Xətt", "en": "Line", "ru": "Линия"},
		"builder.tool.video":  {"az": "Video", "en": "Video", "ru": "Видео"},

		// ---- JS: element type names ----
		"builder.js.type.rect":   {"az": "Düzbucaqlı", "en": "Rectangle", "ru": "Прямоугольник"},
		"builder.js.type.text":   {"az": "Mətn", "en": "Text", "ru": "Текст"},
		"builder.js.type.button": {"az": "Düymə", "en": "Button", "ru": "Кнопка"},
		"builder.js.type.image":  {"az": "Şəkil", "en": "Image", "ru": "Изображение"},
		"builder.js.type.form":   {"az": "Form", "en": "Form", "ru": "Форма"},
		"builder.js.type.line":   {"az": "Xətt", "en": "Line", "ru": "Линия"},

		// ---- JS: page settings ----
		"builder.js.page":       {"az": "Səhifə", "en": "Page", "ru": "Страница"},
		"builder.js.page_title": {"az": "Səhifə başlığı (brauzer vərəqi)", "en": "Page title (browser tab)", "ru": "Заголовок страницы (вкладка браузера)"},
		"builder.js.canvas_w":   {"az": "Kanvas eni (px)", "en": "Canvas width (px)", "ru": "Ширина холста (px)"},
		"builder.js.canvas_h":   {"az": "Kanvas hündürlüyü (px)", "en": "Canvas height (px)", "ru": "Высота холста (px)"},
		"builder.js.page_bg":    {"az": "Fon rəngi", "en": "Background color", "ru": "Цвет фона"},
		"builder.js.redirect":   {"az": "Yönləndirmə URL (opsional)", "en": "Redirect URL (optional)", "ru": "URL перенаправления (необязательно)"},
		"builder.js.form_hint":  {"az": "Form elementi avtomatik olaraq {{.SubmitURL}} ünvanına post edir — “göndərdi” hadisəsi qeyd olunur, dəyərlər saxlanmır.", "en": "The form element automatically posts to {{.SubmitURL}} — the “submitted” event is recorded; values are never stored.", "ru": "Элемент формы автоматически отправляет данные на {{.SubmitURL}} — событие «отправлено» фиксируется, значения не сохраняются."},

		// ---- JS: common element properties ----
		"builder.js.dim_w":         {"az": "En", "en": "Width", "ru": "Ширина"},
		"builder.js.dim_h":         {"az": "Hünd.", "en": "Height", "ru": "Высота"},
		"builder.js.text":          {"az": "Mətn", "en": "Text", "ru": "Текст"},
		"builder.js.bg":            {"az": "Fon", "en": "Background", "ru": "Фон"},
		"builder.js.text_color":    {"az": "Mətn rəngi", "en": "Text color", "ru": "Цвет текста"},
		"builder.js.color":         {"az": "Rəng", "en": "Color", "ru": "Цвет"},
		"builder.js.border_w":      {"az": "Çərçivə eni", "en": "Border width", "ru": "Толщина рамки"},
		"builder.js.border_c":      {"az": "Çərçivə rəngi", "en": "Border color", "ru": "Цвет рамки"},
		"builder.js.border_style":  {"az": "Çərçivə stili", "en": "Border style", "ru": "Стиль рамки"},
		"builder.js.border_solid":  {"az": "Davamlı", "en": "Solid", "ru": "Сплошная"},
		"builder.js.border_dashed": {"az": "Kəsik", "en": "Dashed", "ru": "Пунктирная"},
		"builder.js.border_none":   {"az": "Yoxdur", "en": "None", "ru": "Нет"},
		"builder.js.radius":        {"az": "Künc radiusu", "en": "Corner radius", "ru": "Радиус углов"},
		"builder.js.shadow":        {"az": "Kölgə", "en": "Shadow", "ru": "Тень"},
		"builder.js.font_size":     {"az": "Şrift ölçüsü", "en": "Font size", "ru": "Размер шрифта"},
		"builder.js.font_weight":   {"az": "Qalınlıq", "en": "Font weight", "ru": "Насыщенность"},
		"builder.js.weight_400":    {"az": "Adi", "en": "Regular", "ru": "Обычный"},
		"builder.js.weight_600":    {"az": "Yarımqalın", "en": "Semi-bold", "ru": "Полужирный"},
		"builder.js.weight_700":    {"az": "Qalın", "en": "Bold", "ru": "Жирный"},
		"builder.js.font":          {"az": "Şrift", "en": "Font", "ru": "Шрифт"},
		"builder.js.font_system":   {"az": "Sistem", "en": "System", "ru": "Системный"},
		"builder.js.align":         {"az": "Hizalama", "en": "Alignment", "ru": "Выравнивание"},
		"builder.js.align_left":    {"az": "Sol", "en": "Left", "ru": "Слева"},
		"builder.js.align_center":  {"az": "Mərkəz", "en": "Center", "ru": "По центру"},
		"builder.js.align_right":   {"az": "Sağ", "en": "Right", "ru": "Справа"},
		"builder.js.opacity":       {"az": "Opaqlıq (%)", "en": "Opacity (%)", "ru": "Непрозрачность (%)"},

		// ---- JS: button actions ----
		"builder.js.action":      {"az": "Klik davranışı", "en": "Click behavior", "ru": "Действие при клике"},
		"builder.js.action_none": {"az": "Heç nə", "en": "Nothing", "ru": "Ничего"},
		"builder.js.action_url":  {"az": "URL-ə yönləndir", "en": "Redirect to URL", "ru": "Перейти по URL"},
		"builder.js.action_js":   {"az": "JS kodu çalışdır", "en": "Run JS code", "ru": "Выполнить JS-код"},
		"builder.js.new_tab":     {"az": "Yeni vərəqdə aç", "en": "Open in new tab", "ru": "Открыть в новой вкладке"},
		"builder.js.js_code":     {"az": "JS kodu", "en": "JS code", "ru": "JS-код"},
		"builder.js.js_hint":     {"az": "Kod yalnız canlı səhifədə işləyir — redaktorda klik seçim üçündür. Kliklənən düyməyə “el” dəyişəni ilə müraciət edə bilərsiniz.", "en": "The code runs only on the live page — clicks in the editor are for selection. Refer to the clicked button via the “el” variable.", "ru": "Код работает только на опубликованной странице — клики в редакторе служат для выделения. К нажатой кнопке можно обратиться через переменную «el»."},

		// ---- JS: image ----
		"builder.js.img_url":  {"az": "Şəkil URL", "en": "Image URL", "ru": "URL изображения"},
		"builder.js.alt":      {"az": "Alt mətn (opsional)", "en": "Alt text (optional)", "ru": "Alt-текст (необязательно)"},
		"builder.js.img_hint": {"az": "Şəkil URL-i sağ paneldə daxil edin", "en": "Enter the image URL in the right panel", "ru": "Укажите URL изображения в правой панели"},

		// ---- JS: form composite ----
		"builder.js.form_title":   {"az": "Forma başlığı", "en": "Form title", "ru": "Заголовок формы"},
		"builder.js.user_ph":      {"az": "İstifadəçi adı (placeholder)", "en": "Username (placeholder)", "ru": "Имя пользователя (плейсхолдер)"},
		"builder.js.pass_ph":      {"az": "Parol (placeholder)", "en": "Password (placeholder)", "ru": "Пароль (плейсхолдер)"},
		"builder.js.btn_text":     {"az": "Düymə mətni", "en": "Button text", "ru": "Текст кнопки"},
		"builder.js.btn_bg":       {"az": "Düymə fonu", "en": "Button background", "ru": "Фон кнопки"},
		"builder.js.btn_color":    {"az": "Düymə mətni rəngi", "en": "Button text color", "ru": "Цвет текста кнопки"},
		"builder.js.input_bg":     {"az": "Input fonu", "en": "Input background", "ru": "Фон поля ввода"},
		"builder.js.input_border": {"az": "Input çərçivəsi", "en": "Input border", "ru": "Рамка поля ввода"},

		// ---- JS: z-order / delete ----
		"builder.js.z_back":    {"az": "Ən geriyə", "en": "Send to back", "ru": "На задний план"},
		"builder.js.z_bwd":     {"az": "Geriyə", "en": "Backward", "ru": "Назад"},
		"builder.js.z_fwd":     {"az": "Önə", "en": "Forward", "ru": "Вперёд"},
		"builder.js.z_front":   {"az": "Ən önə", "en": "Bring to front", "ru": "На передний план"},
		"builder.js.delete_el": {"az": "Elementi sil", "en": "Delete element", "ru": "Удалить элемент"},

		// ---- JS: default content for new elements/pages ----
		"builder.js.def_text":       {"az": "Mətn", "en": "Text", "ru": "Текст"},
		"builder.js.def_button":     {"az": "Düymə", "en": "Button", "ru": "Кнопка"},
		"builder.js.def_form_title": {"az": "Hesabınıza daxil olun", "en": "Sign in to your account", "ru": "Войдите в свой аккаунт"},
		"builder.js.def_user_ph":    {"az": "İstifadəçi adı", "en": "Username", "ru": "Имя пользователя"},
		"builder.js.def_pass_ph":    {"az": "Parol", "en": "Password", "ru": "Пароль"},
		"builder.js.def_btn":        {"az": "Daxil ol", "en": "Sign in", "ru": "Войти"},
		"builder.js.def_page_title": {"az": "Açılış səhifəsi", "en": "Landing page", "ru": "Целевая страница"},

		// ---- Template gallery ----
		"builder.gallery":       {"az": "Şablonlar", "en": "Templates", "ru": "Шаблоны"},
		"builder.gallery_title": {"az": "Şablon seç", "en": "Choose a template", "ru": "Выберите шаблон"},
		"builder.gallery_sub":   {"az": "Hazır düzənlə başlayın — hər element sonradan tam redaktə və köçürülə bilər.", "en": "Start from a ready-made layout — every element stays fully editable and movable afterwards.", "ru": "Начните с готового макета — каждый элемент остаётся полностью редактируемым и перемещаемым."},

		// ---- JS: gallery ----
		"builder.js.gallery_replace": {"az": "Kanvasda elementlər var — seçilən şablon mövcud məzmunu ƏVƏZ edəcək. Davam edilsin?", "en": "The canvas has elements — the chosen template will REPLACE the current content. Continue?", "ru": "На холсте есть элементы — выбранный шаблон ЗАМЕНИТ текущее содержимое. Продолжить?"},
		"builder.js.tpl_blank":       {"az": "Boş səhifə", "en": "Blank page", "ru": "Пустая страница"},
		"builder.js.tpl_ms365":       {"az": "Microsoft 365 giriş", "en": "Microsoft 365 sign-in", "ru": "Вход в Microsoft 365"},
		"builder.js.tpl_google":      {"az": "Google hesabı girişi", "en": "Google account sign-in", "ru": "Вход в аккаунт Google"},
		"builder.js.tpl_instagram":   {"az": "Instagram giriş", "en": "Instagram sign-in", "ru": "Вход в Instagram"},
		"builder.js.tpl_facebook":    {"az": "Facebook giriş", "en": "Facebook sign-in", "ru": "Вход в Facebook"},
		"builder.js.tpl_brand_note":  {"az": "Brend işarələri sahiblərinə aiddir", "en": "Brand assets belong to their owners", "ru": "Элементы бренда принадлежат их владельцам"},
		"builder.js.tpl_tour":        {"az": "Ekskursiya — səyahət saytı", "en": "Excursion — travel site", "ru": "Экскурсии — сайт путешествий"},
		"builder.js.tpl_shop":        {"az": "Onlayn mağaza", "en": "Online shop", "ru": "Интернет-магазин"},

		// ---- JS: çoxsəhifəli redaktə (səhifə zolağı) ----
		"builder.js.home_page":        {"az": "Ana səhifə", "en": "Home page", "ru": "Главная страница"},
		"builder.js.new_page":         {"az": "Yeni səhifə", "en": "New page", "ru": "Новая страница"},
		"builder.js.page_add":         {"az": "Səhifə əlavə et", "en": "Add page", "ru": "Добавить страницу"},
		"builder.js.page_rename":      {"az": "Səhifənin adı:", "en": "Page name:", "ru": "Название страницы:"},
		"builder.js.page_del_confirm": {"az": "Səhifə və içindəki elementlər silinsin?", "en": "Delete the page and its elements?", "ru": "Удалить страницу и её элементы?"},
		"builder.js.page_del":         {"az": "Sil", "en": "Delete", "ru": "Удалить"},
		"builder.js.action_page":      {"az": "Səhifəyə keç", "en": "Go to page", "ru": "Перейти на страницу"},

		// ---- JS: video elementi ----
		"builder.js.type.video":  {"az": "Video", "en": "Video", "ru": "Видео"},
		"builder.js.video_src":   {"az": "Video (mənbə)", "en": "Video (source)", "ru": "Видео (источник)"},
		"builder.js.poster_url":  {"az": "Poster şəkli URL", "en": "Poster image URL", "ru": "URL постера"},
		"builder.js.loop":        {"az": "Təkrarla (loop)", "en": "Loop", "ru": "Зациклить"},
		"builder.js.tpl_shahdag": {"az": "Şahdağ Turu", "en": "Shahdag Tour", "ru": "Тур на Шахдаг"},

		// ---- JS: vizual effektlər ----
		"builder.js.grad":       {"az": "Qradyent fon", "en": "Gradient background", "ru": "Градиентный фон"},
		"builder.js.sh_none":    {"az": "Yoxdur", "en": "None", "ru": "Нет"},
		"builder.js.sh_soft":    {"az": "Yüngül", "en": "Soft", "ru": "Лёгкая"},
		"builder.js.sh_med":     {"az": "Orta", "en": "Medium", "ru": "Средняя"},
		"builder.js.sh_dram":    {"az": "Dramatik", "en": "Dramatic", "ru": "Драматичная"},
		"builder.js.hover":      {"az": "Hover effekti", "en": "Hover effect", "ru": "Эффект при наведении"},
		"builder.js.hvr_none":   {"az": "Yoxdur", "en": "None", "ru": "Нет"},
		"builder.js.hvr_lift":   {"az": "Qalxır", "en": "Lift", "ru": "Подъём"},
		"builder.js.hvr_grow":   {"az": "Böyür", "en": "Grow", "ru": "Увеличение"},
		"builder.js.hvr_glow":   {"az": "Parıltı", "en": "Glow", "ru": "Свечение"},
		"builder.js.hvr_darken": {"az": "Tündləşir", "en": "Darken", "ru": "Затемнение"},
		"builder.js.anim":       {"az": "Giriş animasiyası", "en": "Entrance animation", "ru": "Анимация появления"},
		"builder.js.anim_none":  {"az": "Yoxdur", "en": "None", "ru": "Нет"},
		"builder.js.anim_fu":    {"az": "Yuxarı sürüşmə", "en": "Fade in up", "ru": "Снизу вверх"},
		"builder.js.anim_f":     {"az": "Solma", "en": "Fade in", "ru": "Появление"},
		"builder.js.anim_z":     {"az": "Yaxınlaşma", "en": "Zoom in", "ru": "Приближение"},
		"builder.js.anim_kb":    {"az": "Ken Burns (yavaş zum)", "en": "Ken Burns (slow zoom)", "ru": "Кен Бёрнс (медленный зум)"},
		"builder.js.resp":       {"az": "Responsiv miqyas (mobil)", "en": "Responsive scaling (mobile)", "ru": "Адаптивный масштаб (мобильные)"},

		// ---- Cihaz önizləmə paneli (preview shell) + kanvas viewport keçidi ----
		"builder.view_desktop": {"az": "Masaüstü", "en": "Desktop", "ru": "Десктоп"},
		"builder.view_tablet":  {"az": "Planşet", "en": "Tablet", "ru": "Планшет"},
		"builder.view_mobile":  {"az": "Mobil", "en": "Mobile", "ru": "Мобильный"},
		"dev.title":            {"az": "Cihaz önizləməsi", "en": "Device preview", "ru": "Предпросмотр по устройствам"},
		"dev.back":             {"az": "Redaktora qayıt", "en": "Back to editor", "ru": "Назад в редактор"},
		"dev.desktop":          {"az": "Masaüstü", "en": "Desktop", "ru": "Десктоп"},
		"dev.laptop":           {"az": "Laptop", "en": "Laptop", "ru": "Ноутбук"},
		"dev.tablet":           {"az": "Planşet", "en": "Tablet", "ru": "Планшет"},
		"dev.mobile":           {"az": "Mobil", "en": "Mobile", "ru": "Мобильный"},

		// ---- JS: form sahələri redaktoru ----
		"builder.js.form_fields":       {"az": "Form sahələri", "en": "Form fields", "ru": "Поля формы"},
		"builder.js.field_add":         {"az": "Sahə əlavə et", "en": "Add field", "ru": "Добавить поле"},
		"builder.js.field_type":        {"az": "Tip", "en": "Type", "ru": "Тип"},
		"builder.js.field_name":        {"az": "Ad (name)", "en": "Name (attribute)", "ru": "Имя (атрибут)"},
		"builder.js.field_label":       {"az": "Label (opsional)", "en": "Label (optional)", "ru": "Подпись (необязательно)"},
		"builder.js.field_ph":          {"az": "Placeholder", "en": "Placeholder", "ru": "Плейсхолдер"},
		"builder.js.field_req":         {"az": "Məcburi", "en": "Required", "ru": "Обязательное"},
		"builder.js.field_opts":        {"az": "Seçimlər (hər sətir bir)", "en": "Options (one per line)", "ru": "Варианты (по одному на строку)"},
		"builder.js.field_checkbox_ph": {"az": "Qaydaları qəbul edirəm", "en": "I accept the terms", "ru": "Я принимаю условия"},
		"builder.js.ftype_text":        {"az": "Mətn", "en": "Text", "ru": "Текст"},
		"builder.js.ftype_email":       {"az": "E-poçt", "en": "Email", "ru": "Эл. почта"},
		"builder.js.ftype_password":    {"az": "Parol", "en": "Password", "ru": "Пароль"},
		"builder.js.ftype_tel":         {"az": "Telefon", "en": "Phone", "ru": "Телефон"},
		"builder.js.ftype_number":      {"az": "Rəqəm", "en": "Number", "ru": "Число"},
		"builder.js.ftype_date":        {"az": "Tarix", "en": "Date", "ru": "Дата"},
		"builder.js.ftype_textarea":    {"az": "Uzun mətn", "en": "Long text", "ru": "Многострочный текст"},
		"builder.js.ftype_select":      {"az": "Açılan siyahı", "en": "Dropdown", "ru": "Выпадающий список"},
		"builder.js.ftype_checkbox":    {"az": "Checkbox", "en": "Checkbox", "ru": "Флажок"},

		// ---- JS: form composite extras (fidelity variants) ----
		"builder.js.input_style":     {"az": "Input stili", "en": "Input style", "ru": "Стиль поля"},
		"builder.js.input_box":       {"az": "Qutu", "en": "Box", "ru": "Рамка"},
		"builder.js.input_underline": {"az": "Altxətt", "en": "Underline", "ru": "Подчёркивание"},
		"builder.js.input_outline":   {"az": "Kontur", "en": "Outline", "ru": "Контур"},
		"builder.js.btn_pill":        {"az": "Hap formalı düymə (pill)", "en": "Pill-shaped button", "ru": "Кнопка-«пилюля»"},
		"builder.js.subtitle":        {"az": "Alt başlıq", "en": "Subtitle", "ru": "Подзаголовок"},
		"builder.js.links":           {"az": "Keçid sətirləri (hər sətir: Mətn və ya Mətn|https://…)", "en": "Link rows (one per line: Text or Text|https://…)", "ru": "Строки-ссылки (по одной: Текст или Текст|https://…)"},
		"builder.js.link_color":      {"az": "Link rəngi", "en": "Link color", "ru": "Цвет ссылок"},
		"builder.js.font_roboto":     {"az": "Roboto (daxili)", "en": "Roboto (embedded)", "ru": "Roboto (встроен)"},

		// ---- JS: design suggestions ----
		"builder.js.tips":            {"az": "Təkliflər", "en": "Suggestions", "ru": "Предложения"},
		"builder.js.tip_center_form": {"az": "Formu üfüqi mərkəzlə", "en": "Center the form horizontally", "ru": "Выровнять форму по центру"},
		"builder.js.tip_align_left":  {"az": "Sol kənarları hizala", "en": "Align left edges", "ru": "Выровнять левые края"},
		"builder.js.tip_equal_width": {"az": "Enləri bərabərləşdir", "en": "Equalize widths", "ru": "Выровнять ширину"},
		"builder.js.tip_spacing":     {"az": "Şaquli məsafələri bərabərləşdir", "en": "Even out vertical spacing", "ru": "Выровнять вертикальные отступы"},
		"builder.js.tip_contrast":    {"az": "Mətn/fon kontrastını artır", "en": "Increase text/background contrast", "ru": "Увеличить контраст текста/фона"},
		"builder.js.tip_presets":     {"az": "Dizayn presetləri", "en": "Design presets", "ru": "Дизайн-пресеты"},
		"builder.js.preset_corp":     {"az": "Korporativ mavi", "en": "Corporate blue", "ru": "Корпоративный синий"},
		"builder.js.preset_dark":     {"az": "Tünd rejim", "en": "Dark mode", "ru": "Тёмная тема"},
		"builder.js.preset_minimal":  {"az": "Minimal ağ", "en": "Minimal white", "ru": "Минималистичный белый"},
		"builder.js.preset_green":    {"az": "Yaşıl", "en": "Green", "ru": "Зелёный"},

		// ---- JS: font picker ----
		"builder.js.font_group_sans":    {"az": "Sans", "en": "Sans", "ru": "Без засечек"},
		"builder.js.font_group_serif":   {"az": "Serif", "en": "Serif", "ru": "С засечками"},
		"builder.js.font_group_mono":    {"az": "Mono", "en": "Mono", "ru": "Моноширинный"},
		"builder.js.font_group_display": {"az": "Display", "en": "Display", "ru": "Декоративный"},
		"builder.js.font_custom":        {"az": "Xüsusi…", "en": "Custom…", "ru": "Свой шрифт…"},
		"builder.js.font_custom_ph":     {"az": "məs. 'Inter', sans-serif", "en": "e.g. 'Inter', sans-serif", "ru": "напр. 'Inter', sans-serif"},
	})
}
