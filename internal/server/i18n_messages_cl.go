package server

// Clients list screen (clients.html): the page header, the new-client form and
// the client table.
func init() {
	addMessages(map[string]map[string]string{
		"cl.subtitle":       {"az": "Test etdiyiniz təşkilatlar, icazələr və hədəf qrupları.", "en": "The organizations you test, their authorizations and target groups.", "ru": "Организации, которые вы тестируете, их разрешения и целевые группы."},
		"cl.new_client":     {"az": "Yeni müştəri", "en": "New client", "ru": "Новый клиент"},
		"cl.empty":          {"az": "Hələ müştəri yoxdur.", "en": "No clients yet.", "ru": "Клиентов пока нет."},
		"cl.delete_confirm": {"az": "Seçilmiş müştərilər və onların BÜTÜN icazələri, qrupları, hədəfləri, şablonları, səhifələri və kampaniyaları həmişəlik silinəcək. Davam edilsin?", "en": "The selected clients and ALL their authorizations, groups, targets, templates, pages and campaigns will be permanently deleted. Continue?", "ru": "Выбранные клиенты и ВСЕ их разрешения, группы, цели, шаблоны, страницы и кампании будут удалены навсегда. Продолжить?"},
	})
}
