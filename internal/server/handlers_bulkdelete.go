package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) handleClientsBulkDelete(w http.ResponseWriter, r *http.Request) {
	// Deleting a client cascades its authorizations, groups, targets, templates,
	// pages and campaigns (ON DELETE CASCADE) — the confirm dialog warns of this.
	s.bulkDelete(w, r, "/clients", "client", s.store.DeleteClient)
}

func (s *Server) handleTemplatesBulkDelete(w http.ResponseWriter, r *http.Request) {
	s.bulkDelete(w, r, "/templates", "template", s.store.DeleteTemplate)
}

func (s *Server) handlePagesBulkDelete(w http.ResponseWriter, r *http.Request) {
	s.bulkDelete(w, r, "/pages", "page", s.store.DeleteLandingPage)
}

// bulkDelete deletes the "ids" selected on a list page via del, counting
// successes and failures (a template or landing page still referenced by a
// campaign cannot be deleted — a foreign-key restriction), then flashes the
// outcome and redirects back.
func (s *Server) bulkDelete(w http.ResponseWriter, r *http.Request, redirect, kind string, del func(context.Context, string) error) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return
	}
	var ok, fail int
	for _, id := range r.Form["ids"] {
		if strings.TrimSpace(id) == "" {
			continue
		}
		if err := del(r.Context(), id); err != nil {
			fail++
			continue
		}
		ok++
	}
	if ok > 0 {
		s.audit(r, kind+".delete", kind, "", fmt.Sprintf("%d deleted", ok))
	}
	if fail > 0 {
		s.flashErr(r, s.tf(r, "flash.deleted_some_inuse", ok, fail))
	} else {
		s.flashOK(r, s.tf(r, "flash.deleted_n", ok))
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}
