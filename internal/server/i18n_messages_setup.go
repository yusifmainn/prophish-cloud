package server

// First-run setup screen (setup.html): creating the platform's first operator
// account.
func init() {
	addMessages(map[string]map[string]string{
		"setup.title":          {"az": "İlk operator hesabı", "en": "First operator account", "ru": "Первый аккаунт оператора"},
		"setup.subtitle":       {"az": "Bu, platformanın ilk operator hesabıdır.", "en": "This is the platform's first operator account.", "ru": "Это первый аккаунт оператора на платформе."},
		"setup.email":          {"az": "Email", "en": "Email", "ru": "Email"},
		"setup.password":       {"az": "Parol", "en": "Password", "ru": "Пароль"},
		"setup.min_chars":      {"az": "Ən azı 12 simvol.", "en": "At least 12 characters.", "ru": "Минимум 12 символов."},
		"setup.create_account": {"az": "Hesabı yarat", "en": "Create account", "ru": "Создать аккаунт"},
	})
}
