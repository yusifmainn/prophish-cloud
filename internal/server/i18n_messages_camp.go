package server

// Campaign list + builder screen (campaigns.html): the page header, the
// create/edit form (basics, authorization, target-group picker, content and
// sending), and the "all campaigns" table. Shared strings (client, status,
// created, cancel, save changes, open, employees count) come from common.*/nav.*.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Page header ----
		"camp.subtitle": {"az": "Simulyasiya kampaniyalarını qurun, işə salın və izləyin.", "en": "Build, launch and track simulation campaigns.", "ru": "Создавайте, запускайте и отслеживайте симуляционные кампании."},

		// ---- Form title ----
		"camp.edit_title": {"az": "Kampaniyanı redaktə et (qaralama)", "en": "Edit campaign (draft)", "ru": "Редактировать кампанию (черновик)"},
		"camp.new_title":  {"az": "Yeni kampaniya", "en": "New campaign", "ru": "Новая кампания"},

		// ---- Prerequisites note (inline links; href kept intact) ----
		"camp.need_intro":    {"az": "Kampaniya qurmaq üçün əvvəlcə lazımdır:", "en": "To build a campaign you first need:", "ru": "Чтобы создать кампанию, сначала нужны:"},
		"camp.need_clients":  {"az": `bir <a href="/clients">müştəri + icazə + qrup</a>`, "en": `a <a href="/clients">client + authorization + group</a>`, "ru": `<a href="/clients">клиент + разрешение + группа</a>`},
		"camp.need_template": {"az": `bir <a href="/templates">email şablonu</a>`, "en": `an <a href="/templates">email template</a>`, "ru": `<a href="/templates">шаблон письма</a>`},
		"camp.need_page":     {"az": `bir <a href="/pages">açılış səhifəsi</a>`, "en": `a <a href="/pages">landing page</a>`, "ru": `<a href="/pages">целевая страница</a>`},
		"camp.need_profile":  {"az": `bir <a href="/profiles">SMTP profili</a>`, "en": `an <a href="/profiles">SMTP profile</a>`, "ru": `<a href="/profiles">SMTP-профиль</a>`},

		// ---- Basics ----
		"camp.sect_basics": {"az": "Əsas", "en": "Basics", "ru": "Основное"},
		"camp.name_label":  {"az": "Kampaniya adı", "en": "Campaign name", "ru": "Название кампании"},
		"camp.name_ph":     {"az": "məs. Q1 maliyyə təlimi", "en": "e.g. Q1 finance training", "ru": "напр. финансовое обучение Q1"},
		"camp.select":      {"az": "— seçin —", "en": "— select —", "ru": "— выберите —"},

		// ---- Authorization ----
		"camp.auth_label":   {"az": "İcazə (RoE)", "en": "Authorization (RoE)", "ru": "Разрешение (RoE)"},
		"camp.auth_expires": {"az": "(bitmə: %s)", "en": "(expires: %s)", "ru": "(истекает: %s)"},
		"camp.filter_hint":  {"az": "Müştəri seçiləndə icazə və qruplar avtomatik ona görə süzülür. İcazəsiz kampaniya göndərilmir.", "en": "When a client is selected, authorizations and groups are filtered to it automatically. A campaign cannot be sent without an authorization.", "ru": "При выборе клиента разрешения и группы фильтруются по нему автоматически. Без разрешения кампания не отправляется."},
		"camp.no_data_pre":  {"az": "Bu müştəri üçün icazə və ya hədəf qrupu yoxdur — əvvəlcə", "en": "This client has no authorization or target group — first add one in", "ru": "У этого клиента нет разрешения или целевой группы — сначала добавьте в разделе"},
		"camp.no_data_post": {"az": " bölməsində əlavə edin.", "en": " section.", "ru": "."},

		// ---- Target groups ----
		"camp.sect_groups":      {"az": "Hədəf qrupları", "en": "Target groups", "ru": "Целевые группы"},
		"camp.no_groups":        {"az": "Qrup yoxdur.", "en": "No groups.", "ru": "Групп нет."},
		"camp.selected_targets": {"az": "Seçilmiş hədəf", "en": "Selected targets", "ru": "Выбранные цели"},
		"camp.employees_word":   {"az": "işçi", "en": "employees", "ru": "сотрудников"},

		// ---- Content and sending ----
		"camp.sect_content": {"az": "Məzmun və göndərmə", "en": "Content and sending", "ru": "Содержание и отправка"},

		// ---- Listener location (tracking links) ----
		"camp.sect_delivery":    {"az": "İzləmə linkləri", "en": "Tracking links", "ru": "Отслеживаемые ссылки"},
		"camp.listener_q":       {"az": "Listener harada olsun?", "en": "Where should the listener run?", "ru": "Где разместить слушатель?"},
		"camp.listener_host":    {"az": "Hazırki hostda", "en": "On this host", "ru": "На текущем хосте"},
		"camp.listener_host_hint": {"az": "Public IP olduqda tövsiyə olunur — linklər birbaşa bu serverə işarə edir", "en": "Recommended when the host has a public IP — links point directly to this server", "ru": "Рекомендуется при наличии публичного IP — ссылки указывают прямо на этот сервер"},
		"camp.listener_cf":      {"az": "Cloudflare gateway vasitəsilə", "en": "Via Cloudflare gateway", "ru": "Через шлюз Cloudflare"},
		"camp.listener_cf_hint": {"az": "Private IP olduqda tövsiyə olunur. Hər kampaniya üçün YENİ tunnel avtomatik açılır. Qeyd: tunnel yenidən başladılanda URL dəyişir — köhnə linklər ölür.", "en": "Recommended when behind a private IP. A NEW tunnel is opened automatically per campaign. Note: when the tunnel restarts, the URL changes — old links die.", "ru": "Рекомендуется при частном IP. Для каждой кампании автоматически открывается НОВЫЙ туннель. Внимание: при перезапуске URL меняется — старые ссылки перестают работать."},
		"camp.listener_custom":  {"az": "Xüsusi URL", "en": "Custom URL", "ru": "Свой URL"},
		"camp.listener_custom_hint": {"az": "Öz domemniz/reverse proxy-niz varsa — təkmil istifadəçilər üçün", "en": "If you have your own domain/reverse proxy — for advanced users", "ru": "Если у вас свой домен/reverse proxy — для продвинутых"},
		"camp.recommended":      {"az": "Tövsiyə olunur", "en": "Recommended", "ru": "Рекомендуется"},
		"camp.listener_label":   {"az": "Listener", "en": "Listener", "ru": "Слушатель"},
		"camp.mode_host":        {"az": "Hazırki host", "en": "This host", "ru": "Текущий хост"},
		"camp.mode_cf":          {"az": "Cloudflare gateway", "en": "Cloudflare gateway", "ru": "Шлюз Cloudflare"},
		"camp.mode_custom":      {"az": "Xüsusi URL", "en": "Custom URL", "ru": "Свой URL"},
		"camp.public_url_label": {"az": "Public URL", "en": "Public URL", "ru": "Публичный URL"},
		"camp.public_url_hint":  {"az": "İzləmə linkləri bu ünvanla qurulur. http:// və ya https:// ilə başlamalıdır.", "en": "Tracking links are built with this URL. It must start with http:// or https://.", "ru": "Отслеживаемые ссылки строятся на этом URL. Должен начинаться с http:// или https://."},

		// ---- Per-campaign quick tunnel ----
		"camp.tunnel_label":        {"az": "Hər kampaniya üçün yeni gateway (tunnel) yarat", "en": "Create a new gateway (tunnel) per campaign", "ru": "Создавать новый шлюз (туннель) для каждой кампании"},
		"camp.tunnel_hint":         {"az": "Avtomatik pulsuz Cloudflare tunnel açılır. Qeyd: tunnel yenidən başladılanda URL dəyişir — köhnə linklər ölür.", "en": "A free Cloudflare tunnel is opened automatically. Note: when the tunnel restarts, the URL changes — old links die.", "ru": "Автоматически открывается бесплатный Cloudflare-туннель. Внимание: при перезапуске туннеля URL меняется — старые ссылки перестают работать."},
		"camp.tunnel_managed_note": {"az": "Bu kampaniyanın linkləri avtomatik idarə olunan tunnel ilə qurulur — URL əl ilə dəyişmir.", "en": "This campaign's links are built from an automatically managed tunnel — the URL cannot be edited manually.", "ru": "Ссылки этой кампании строятся через автоматически управляемый туннель — URL нельзя изменить вручную."},
		"camp.tunnel_live":         {"az": "Tunnel aktivdir", "en": "Tunnel is live", "ru": "Туннель активен"},
		"camp.tunnel_starting":     {"az": "Tunnel hazırlanır…", "en": "Tunnel is starting…", "ru": "Туннель запускается…"},
		"camp.tunnel_note":         {"az": "Sürətli tunnel URL-i yenidən başladılmada dəyişir — göndərilmiş köhnə linklər ölür.", "en": "The quick-tunnel URL changes on restart — previously sent links die.", "ru": "URL быстрого туннеля меняется при перезапуске — отправленные ссылки перестают работать."},
		"camp.template_label":      {"az": "Email şablonu", "en": "Email template", "ru": "Шаблон письма"},
		"camp.page_label":          {"az": "Açılış səhifəsi", "en": "Landing page", "ru": "Целевая страница"},
		"camp.smtp_label":          {"az": "SMTP profili", "en": "SMTP profile", "ru": "SMTP-профиль"},
		"camp.create_draft":        {"az": "Kampaniya yarat (qaralama)", "en": "Create campaign (draft)", "ru": "Создать кампанию (черновик)"},

		// ---- All campaigns table ----
		"camp.all_title": {"az": "Bütün kampaniyalar", "en": "All campaigns", "ru": "Все кампании"},
		"camp.empty":     {"az": "Hələ kampaniya yoxdur. Yuxarıdan birincini yaradın.", "en": "No campaigns yet. Create the first one above.", "ru": "Кампаний пока нет. Создайте первую выше."},
	})
}
