package server

import (
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGroupDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	group, err := s.store.GetGroup(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	client, _ := s.store.GetClient(r.Context(), group.ClientID)
	targets, _ := s.store.ListTargets(r.Context(), id)
	s.render(w, r, http.StatusOK, "group_detail.html", map[string]any{
		"Group":   group,
		"Client":  client,
		"Targets": targets,
	})
}

func (s *Server) handleTargetAdd(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if email == "" || !strings.Contains(email, "@") {
		http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
		return
	}
	if err := s.store.AddTarget(r.Context(), id,
		email, strings.TrimSpace(r.FormValue("first_name")),
		strings.TrimSpace(r.FormValue("last_name")), strings.TrimSpace(r.FormValue("position"))); err != nil {
		s.flashErr(r, "Hədəf əlavə olunmadı.")
	} else {
		s.flashOK(r, "Hədəf əlavə edildi.")
	}
	http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
}

// handleTargetImport ingests a CSV with header: email,first_name,last_name,position
const maxImportTargets = 100_000

func (s *Server) handleTargetImport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Cap the whole request body so a huge upload cannot exhaust process memory.
	r.Body = http.MaxBytesReader(w, r.Body, 16<<20)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	file, _, err := r.FormFile("csv")
	if err != nil {
		http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
		return
	}
	defer file.Close()

	cr := csv.NewReader(file)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true

	var batch [][4]string
	total := 0
	inserted := 0
	first := true
	for {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		if len(rec) == 0 {
			continue
		}
		email := strings.TrimSpace(strings.ToLower(rec[0]))
		// Skip a header row if present.
		if first {
			first = false
			if email == "email" || !strings.Contains(email, "@") {
				continue
			}
		}
		if !strings.Contains(email, "@") {
			continue
		}
		var fn, ln, pos string
		if len(rec) > 1 {
			fn = strings.TrimSpace(rec[1])
		}
		if len(rec) > 2 {
			ln = strings.TrimSpace(rec[2])
		}
		if len(rec) > 3 {
			pos = strings.TrimSpace(rec[3])
		}
		batch = append(batch, [4]string{email, fn, ln, pos})
		total++
		// Flush in bounded chunks so memory stays flat regardless of file size.
		if len(batch) >= 5000 {
			n, _ := s.store.AddTargetsBatch(r.Context(), id, batch)
			inserted += n
			batch = batch[:0]
		}
		if total >= maxImportTargets {
			break
		}
	}

	if len(batch) > 0 {
		n, _ := s.store.AddTargetsBatch(r.Context(), id, batch)
		inserted += n
	}
	if total > 0 {
		if skipped := total - inserted; skipped > 0 {
			s.flashOK(r, s.tf(r, "flash.targets_added_dup", inserted, skipped))
		} else {
			s.flashOK(r, s.tf(r, "flash.targets_added", inserted))
		}
	} else {
		s.flashErr(r, "Heç bir sətir import olunmadı — formatı (email,first_name,last_name,position) yoxlayın.")
	}
	http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
}

func (s *Server) handleGroupUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		s.flashErr(r, "Qrup adı boş ola bilməz.")
		http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
		return
	}
	if err := s.store.UpdateGroup(r.Context(), id, name); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "group.update", "group", id, name)
	s.flashOK(r, "Qrup adı yeniləndi.")
	http.Redirect(w, r, "/groups/"+id, http.StatusSeeOther)
}

func (s *Server) handleGroupDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	g, err := s.store.GetGroup(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := s.store.CountResultsForGroup(r.Context(), id)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		s.flashErr(r, "Qrup silinmədi — hədəfləri bir kampaniyada istifadə olunub.")
		http.Redirect(w, r, "/clients/"+g.ClientID, http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteGroup(r.Context(), id); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "group.delete", "group", id, g.Name)
	s.flashOK(r, "Qrup silindi.")
	http.Redirect(w, r, "/clients/"+g.ClientID, http.StatusSeeOther)
}

func (s *Server) handleTargetUpdate(w http.ResponseWriter, r *http.Request) {
	tid := chi.URLParam(r, "tid")
	t, err := s.store.GetTarget(r.Context(), tid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	if email == "" || !strings.Contains(email, "@") {
		s.flashErr(r, "Düzgün email daxil edin.")
		http.Redirect(w, r, "/groups/"+t.GroupID, http.StatusSeeOther)
		return
	}
	if err := s.store.UpdateTarget(r.Context(), tid, email,
		strings.TrimSpace(r.FormValue("first_name")),
		strings.TrimSpace(r.FormValue("last_name")),
		strings.TrimSpace(r.FormValue("position"))); err != nil {
		s.flashErr(r, "Hədəf yenilənmədi — bu email qrupda artıq ola bilər.")
		http.Redirect(w, r, "/groups/"+t.GroupID, http.StatusSeeOther)
		return
	}
	s.audit(r, "target.update", "target", tid, email)
	s.flashOK(r, "Hədəf yeniləndi.")
	http.Redirect(w, r, "/groups/"+t.GroupID, http.StatusSeeOther)
}

func (s *Server) handleTargetDelete(w http.ResponseWriter, r *http.Request) {
	tid := chi.URLParam(r, "tid")
	t, err := s.store.GetTarget(r.Context(), tid)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	n, err := s.store.CountResultsForTarget(r.Context(), tid)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if n > 0 {
		s.flashErr(r, "Hədəf silinmədi — bir kampaniyada istifadə olunub.")
		http.Redirect(w, r, "/groups/"+t.GroupID, http.StatusSeeOther)
		return
	}
	if err := s.store.DeleteTarget(r.Context(), tid); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.audit(r, "target.delete", "target", tid, t.Email)
	s.flashOK(r, "Hədəf silindi.")
	http.Redirect(w, r, "/groups/"+t.GroupID, http.StatusSeeOther)
}
