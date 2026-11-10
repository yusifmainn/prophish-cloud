package server

// messages is the UI translation catalog: messages[key][lang] = text.
//
// It is assembled from several i18n_messages_*.go files, each of which registers
// its slice of the catalog from an init() via addMessages. Keys are namespaced
// by screen (e.g. "dash.title", "camp.new") with a few shared namespaces:
//
//	nav.*      sidebar navigation
//	common.*   buttons / labels reused across screens (save, cancel, delete, …)
//	pill.*     campaign + target status badges
//	event.*    tracking-event timeline labels (keyed by event type)
//	action.*   audit-log action labels (keyed by action code)
//	flash.*    one-off toast messages set by handlers
//
// Every key must define az, en and ru. The i18n_coverage_test enforces that and
// also checks that every key referenced by a template or handler exists here.
var messages = map[string]map[string]string{}

// addMessages merges a batch into the catalog. Batches are registered from
// init() functions, which run after the messages var is initialized. Keys are
// namespaced per screen, so batches do not collide.
func addMessages(batch map[string]map[string]string) {
	for k, v := range batch {
		messages[k] = v
	}
}
