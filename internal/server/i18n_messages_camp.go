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
		"camp.sect_content":   {"az": "Məzmun və göndərmə", "en": "Content and sending", "ru": "Содержание и отправка"},
		"camp.template_label": {"az": "Email şablonu", "en": "Email template", "ru": "Шаблон письма"},
		"camp.page_label":     {"az": "Açılış səhifəsi", "en": "Landing page", "ru": "Целевая страница"},
		"camp.smtp_label":     {"az": "SMTP profili", "en": "SMTP profile", "ru": "SMTP-профиль"},
		"camp.create_draft":   {"az": "Kampaniya yarat (qaralama)", "en": "Create campaign (draft)", "ru": "Создать кампанию (черновик)"},

		// ---- All campaigns table ----
		"camp.all_title": {"az": "Bütün kampaniyalar", "en": "All campaigns", "ru": "Все кампании"},
		"camp.empty":     {"az": "Hələ kampaniya yoxdur. Yuxarıdan birincini yaradın.", "en": "No campaigns yet. Create the first one above.", "ru": "Кампаний пока нет. Создайте первую выше."},
	})
}
