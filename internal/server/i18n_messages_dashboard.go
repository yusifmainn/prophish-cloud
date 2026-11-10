package server

// Dashboard screen (dashboard.html).
func init() {
	addMessages(map[string]map[string]string{
		"dash.subtitle":        {"az": "Bütün müştərilər üzrə simulyasiya fəaliyyətinin icmalı.", "en": "Overview of simulation activity across all clients.", "ru": "Обзор активности симуляций по всем клиентам."},
		"dash.new_campaign":    {"az": "Yeni kampaniya", "en": "New campaign", "ru": "Новая кампания"},
		"dash.kpi.active":      {"az": "Aktiv kampaniyalar", "en": "Active campaigns", "ru": "Активные кампании"},
		"dash.kpi.active_sub":  {"az": "%d ümumi kampaniya", "en": "%d total campaigns", "ru": "всего кампаний: %d"},
		"dash.kpi.tested":      {"az": "Test olunmuş işçilər", "en": "Employees tested", "ru": "Протестировано сотрудников"},
		"dash.kpi.tested_sub":  {"az": "göndərilmiş hədəflər", "en": "targets emailed", "ru": "отправлено адресатам"},
		"dash.kpi.clickrate":   {"az": "Orta klik faizi", "en": "Average click rate", "ru": "Средний процент переходов"},
		"dash.kpi.lower":       {"az": "aşağı = daha yaxşı", "en": "lower = better", "ru": "ниже = лучше"},
		"dash.kpi.reportrate":  {"az": "Phishing bildirmə faizi", "en": "Phishing report rate", "ru": "Процент сообщений о фишинге"},
		"dash.kpi.higher":      {"az": "yüksək = daha yaxşı", "en": "higher = better", "ru": "выше = лучше"},
		"dash.recent":          {"az": "Son kampaniyalar", "en": "Recent campaigns", "ru": "Недавние кампании"},
		"dash.empty":           {"az": "Hələ kampaniya yoxdur. Soldakı menyudan müştəri və kampaniya yaradın.", "en": "No campaigns yet. Create a client and a campaign from the menu on the left.", "ru": "Пока нет кампаний. Создайте клиента и кампанию в меню слева."},
	})
}
