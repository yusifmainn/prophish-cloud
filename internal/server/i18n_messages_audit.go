package server

// Audit log screen (audit.html): the intro blurb and the recent-events table.
func init() {
	addMessages(map[string]map[string]string{
		"audit.intro": {
			"az": "Operatorların təhlükəsizliyə təsir edən əməliyyatları — kim, nə vaxt, haradan. Yalnız fakt qeyd olunur; <b>heç bir parol və ya gizli dəyər saxlanmır</b>. Son 300 qeyd.",
			"en": "Security-relevant operations performed by operators — who, when, from where. Only the fact is recorded; <b>no password or secret value is stored</b>. Last 300 records.",
			"ru": "Операции операторов, влияющие на безопасность — кто, когда, откуда. Записывается только факт; <b>никакие пароли или секретные значения не хранятся</b>. Последние 300 записей.",
		},
		"audit.recent":     {"az": "Son hadisələr", "en": "Recent events", "ru": "Последние события"},
		"audit.col_time":   {"az": "Vaxt", "en": "Time", "ru": "Время"},
		"audit.col_action": {"az": "Əməliyyat", "en": "Action", "ru": "Действие"},
		"audit.col_object": {"az": "Obyekt", "en": "Object", "ru": "Объект"},
		"audit.col_detail": {"az": "Təfərrüat", "en": "Detail", "ru": "Детали"},
		"audit.col_ip":     {"az": "IP", "en": "IP", "ru": "IP"},
		"audit.empty":      {"az": "Hələ audit qeydi yoxdur.", "en": "No audit records yet.", "ru": "Записей аудита пока нет."},
	})
}
