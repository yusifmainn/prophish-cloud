package server

// Client detail screen (client_detail.html): the info/authorizations/groups
// tabs, the authorization (RoE) list and editor, and the target-group list and
// editor.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Tabs ----
		"cld.tab_info":   {"az": "Məlumat", "en": "Info", "ru": "Информация"},
		"cld.tab_auth":   {"az": "İcazələr", "en": "Authorizations", "ru": "Разрешения"},
		"cld.tab_groups": {"az": "Qruplar", "en": "Groups", "ru": "Группы"},

		// ---- Info tab ----
		"cld.info_title":     {"az": "Müştəri məlumatı", "en": "Client information", "ru": "Информация о клиенте"},
		"cld.danger_warn":    {"az": "Müştərini silmək onun <b>bütün</b> icazə, qrup, işçi, şablon, səhifə və kampaniyalarını (nəticələri ilə) həmişəlik silir. Geri qaytarıla bilməz.", "en": "Deleting a client permanently removes <b>all</b> of its authorizations, groups, employees, templates, pages and campaigns (along with their results). This cannot be undone.", "ru": "Удаление клиента навсегда удаляет <b>все</b> его разрешения, группы, сотрудников, шаблоны, страницы и кампании (вместе с их результатами). Это нельзя отменить."},
		"cld.delete_confirm": {"az": "DİQQƏT: '%s' və ona bağlı BÜTÜN məlumat həmişəlik silinəcək. Davam edilsin?", "en": "WARNING: '%s' and ALL data linked to it will be permanently deleted. Continue?", "ru": "ВНИМАНИЕ: '%s' и ВСЕ связанные с ним данные будут удалены навсегда. Продолжить?"},
		"cld.delete_client":  {"az": "Müştərini sil", "en": "Delete client", "ru": "Удалить клиента"},

		// ---- Authorizations (RoE) tab ----
		"cld.auth_title":          {"az": "İcazələr və Razılaşma (RoE)", "en": "Authorizations and Rules of Engagement (RoE)", "ru": "Разрешения и правила проведения (RoE)"},
		"cld.auth_hint":           {"az": "Kampaniya yalnız etibarlı icazə onu əhatə etdiyi zaman göndərilə bilər.", "en": "A campaign can only be sent while a valid authorization covers it.", "ru": "Кампанию можно отправить только тогда, когда её покрывает действительное разрешение."},
		"cld.authstate_active":    {"az": "Aktiv", "en": "Active", "ru": "Активно"},
		"cld.authstate_future":    {"az": "Gələcək", "en": "Future", "ru": "В будущем"},
		"cld.authstate_expired":   {"az": "Bitib", "en": "Expired", "ru": "Истекло"},
		"cld.auth_delete_confirm": {"az": "Bu icazə silinsin?", "en": "Delete this authorization?", "ru": "Удалить это разрешение?"},
		"cld.auth_empty":          {"az": "Hələ icazə yoxdur. Aşağıdan əlavə edin.", "en": "No authorizations yet. Add one below.", "ru": "Разрешений пока нет. Добавьте ниже."},
		"cld.auth_new":            {"az": "Yeni icazə", "en": "New authorization", "ru": "Новое разрешение"},
		"cld.auth_add":            {"az": "İcazə əlavə et", "en": "Add authorization", "ru": "Добавить разрешение"},

		// ---- Authorization fields (edit + create forms) ----
		"cld.scope":             {"az": "Əhatə dairəsi", "en": "Scope", "ru": "Область действия"},
		"cld.scope_full":        {"az": "Əhatə dairəsi (icazə verilən domen/hədəflər)", "en": "Scope (permitted domain/targets)", "ru": "Область действия (разрешённый домен/цели)"},
		"cld.scope_ph":          {"az": "*.musteri.az daxili işçilər", "en": "*.musteri.az internal employees", "ru": "*.musteri.az внутренние сотрудники"},
		"cld.signed_by":         {"az": "İmzalayan", "en": "Signed by", "ru": "Кем подписано"},
		"cld.signed_by_full":    {"az": "İmzalayan (müştəri tərəfindən)", "en": "Signed by (on the client side)", "ru": "Кем подписано (со стороны клиента)"},
		"cld.document_ref":      {"az": "Sənəd istinadı", "en": "Document reference", "ru": "Ссылка на документ"},
		"cld.document_ref_full": {"az": "Sənəd istinadı (müqavilə/RoE fayl adı və ya nömrə)", "en": "Document reference (contract/RoE file name or number)", "ru": "Ссылка на документ (название или номер файла договора/RoE)"},
		"cld.valid_from":        {"az": "Etibarlıdır (başlanğıc)", "en": "Valid (from)", "ru": "Действительно (с)"},
		"cld.valid_to":          {"az": "Etibarlıdır (son)", "en": "Valid (to)", "ru": "Действительно (по)"},

		// ---- Target groups tab ----
		"cld.groups_title":         {"az": "Hədəf qrupları", "en": "Target groups", "ru": "Целевые группы"},
		"cld.open":                 {"az": "Aç", "en": "Open", "ru": "Открыть"},
		"cld.group_delete_confirm": {"az": "'%s' qrupu və işçiləri silinsin?", "en": "Delete group '%s' and its employees?", "ru": "Удалить группу '%s' и её сотрудников?"},
		"cld.group_name":           {"az": "Qrup adı", "en": "Group name", "ru": "Название группы"},
		"cld.group_name_ph":        {"az": "Maliyyə şöbəsi", "en": "Finance department", "ru": "Финансовый отдел"},
		"cld.groups_empty":         {"az": "Hələ qrup yoxdur. Aşağıdan yaradın.", "en": "No groups yet. Create one below.", "ru": "Групп пока нет. Создайте ниже."},
		"cld.group_new":            {"az": "Yeni qrup", "en": "New group", "ru": "Новая группа"},
		"cld.group_create":         {"az": "Qrup yarat", "en": "Create group", "ru": "Создать группу"},
	})
}
