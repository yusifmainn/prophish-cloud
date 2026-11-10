package server

// Target-group detail screen (group_detail.html): the target list and inline
// editor, the single-target add form, and the CSV import card.
func init() {
	addMessages(map[string]map[string]string{
		// ---- Target list + rename ----
		"grp.targets_n":             {"az": "Hədəflər (%d)", "en": "Targets (%d)", "ru": "Цели (%d)"},
		"grp.rename_group":          {"az": "Qrup adını dəyiş", "en": "Rename group", "ru": "Переименовать группу"},
		"grp.group_name":            {"az": "Qrup adı", "en": "Group name", "ru": "Название группы"},
		"grp.target_delete_confirm": {"az": "'%s' siyahıdan silinsin?", "en": "Remove '%s' from the list?", "ru": "Удалить '%s' из списка?"},
		"grp.last_name":             {"az": "Soyad", "en": "Surname", "ru": "Фамилия"},
		"grp.targets_empty":         {"az": "Hələ hədəf yoxdur. Aşağıdan tək-tək və ya CSV ilə əlavə edin.", "en": "No targets yet. Add them below one by one or via CSV.", "ru": "Целей пока нет. Добавьте их ниже по одной или через CSV."},

		// ---- Add single target ----
		"grp.add_single": {"az": "Tək hədəf əlavə et", "en": "Add a single target", "ru": "Добавить одну цель"},

		// ---- CSV import ----
		"grp.csv_import": {"az": "CSV import", "en": "CSV import", "ru": "Импорт CSV"},
		"grp.csv_hint":   {"az": "Sütun sırası: <code>email, first_name, last_name, position</code>. Birinci sətir başlıq ola bilər. Nəticə yuxarıda banner kimi göstəriləcək.", "en": "Column order: <code>email, first_name, last_name, position</code>. The first row may be a header. The result is shown as a banner above.", "ru": "Порядок столбцов: <code>email, first_name, last_name, position</code>. Первая строка может быть заголовком. Результат показывается в виде баннера выше."},
		"grp.upload":     {"az": "Yüklə", "en": "Upload", "ru": "Загрузить"},
	})
}
