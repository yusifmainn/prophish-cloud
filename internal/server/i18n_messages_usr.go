package server

// Users screen (users.html): the current-account panel (password change + REST
// API key), and the admin-only operator creation form and operator list.
func init() {
	addMessages(map[string]map[string]string{
		// ---- My account ----
		"usr.my_account":       {"az": "Hesabım", "en": "My account", "ru": "Мой аккаунт"},
		"usr.admin":            {"az": "Admin", "en": "Admin", "ru": "Админ"},
		"usr.current_password": {"az": "Cari parol", "en": "Current password", "ru": "Текущий пароль"},
		"usr.new_password":     {"az": "Yeni parol (min 12 simvol)", "en": "New password (min 12 characters)", "ru": "Новый пароль (мин. 12 символов)"},
		"usr.change_password":  {"az": "Parolu dəyiş", "en": "Change password", "ru": "Изменить пароль"},

		// ---- REST API key ----
		"usr.rest_api_key": {"az": "REST API açarı", "en": "REST API key", "ru": "Ключ REST API"},
		"usr.key_reveal":   {"az": "Açarınız (yalnız <strong>indi</strong> göstərilir, gizli saxlayın): ", "en": "Your key (shown <strong>only now</strong>, keep it secret): ", "ru": "Ваш ключ (показывается <strong>только сейчас</strong>, храните в секрете): "},
		"usr.usage":        {"az": "İstifadə: ", "en": "Usage: ", "ru": "Использование: "},
		"usr.key_active":   {"az": "Bir API açarı aktivdir (yalnız hash saxlanılır). İtirsəniz, yenisini yaradın.", "en": "An API key is active (only its hash is stored). If you lose it, create a new one.", "ru": "API-ключ активен (хранится только его хеш). Если потеряете, создайте новый."},
		"usr.rotate_key":   {"az": "Yeni açar yarat (köhnəni əvəz edir)", "en": "Create new key (replaces the old one)", "ru": "Создать новый ключ (заменит старый)"},
		"usr.revoke_key":   {"az": "Açarı ləğv et", "en": "Revoke key", "ru": "Отозвать ключ"},
		"usr.create_key":   {"az": "API açarı yarat", "en": "Create API key", "ru": "Создать API-ключ"},

		// ---- New operator ----
		"usr.new_operator":   {"az": "Yeni operator", "en": "New operator", "ru": "Новый оператор"},
		"usr.password_min12": {"az": "Parol (min 12)", "en": "Password (min 12)", "ru": "Пароль (мин. 12)"},
		"usr.role":           {"az": "Rol", "en": "Role", "ru": "Роль"},

		// ---- Operator list ----
		"usr.operators":      {"az": "Operatorlar", "en": "Operators", "ru": "Операторы"},
		"usr.last_login":     {"az": "Son giriş", "en": "Last login", "ru": "Последний вход"},
		"usr.delete_confirm": {"az": "Bu operator hesabı silinsin?", "en": "Delete this operator account?", "ru": "Удалить эту учётную запись оператора?"},
		"usr.admin_only":     {"az": "Operator idarəsi yalnız admin üçün əlçatandır.", "en": "Operator management is available to admins only.", "ru": "Управление операторами доступно только администраторам."},
	})
}
