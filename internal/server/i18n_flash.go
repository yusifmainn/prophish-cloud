package server

// flashMessages reverse-translates handler-set messages. Flash toasts and inline
// form errors are written in Azerbaijani at the call site (s.flashOK/flashErr,
// data["Error"]); rather than touch ~70 call sites, render() looks the stored
// Azerbaijani text up here and swaps in the active language. The data lives in
// i18n_flash_data.go (keyed by the exact Azerbaijani source string).
var flashMessages = map[string]map[string]string{}

// addFlash registers a batch of reverse translations (az source → {en, ru}).
func addFlash(batch map[string]map[string]string) {
	for k, v := range batch {
		flashMessages[k] = v
	}
}

// trFlash translates an Azerbaijani-sourced flash/error message into lang. It
// returns the input unchanged when lang is Azerbaijani, the string is empty, or
// no reverse translation exists (e.g. a message assembled at runtime), so the
// worst case is the original Azerbaijani text rather than a blank banner.
func trFlash(lang, msg string) string {
	if lang == defaultLang || msg == "" {
		return msg
	}
	if m, ok := flashMessages[msg]; ok {
		if s := m[lang]; s != "" {
			return s
		}
	}
	return msg
}
