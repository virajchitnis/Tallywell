package server

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"time"

	"github.com/tallywell/tallywell/internal/app"
	"github.com/tallywell/tallywell/internal/model"
	"github.com/tallywell/tallywell/internal/reconcile"
	"github.com/tallywell/tallywell/internal/report"
)

// --- setup / unlock / lock ---

func (s *Server) handleSetupForm(w http.ResponseWriter, r *http.Request) {
	if s.app.Phase() != app.PhaseNeedsSetup {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	s.render(w, "setup.html", pageData{})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	pass := r.FormValue("passphrase")
	confirm := r.FormValue("confirm")
	if len(pass) < 8 {
		s.render(w, "setup.html", pageData{Flash: "Choose a passphrase of at least 8 characters."})
		return
	}
	if pass != confirm {
		s.render(w, "setup.html", pageData{Flash: "The two passphrases do not match."})
		return
	}
	if err := s.app.Setup(pass); err != nil {
		s.render(w, "setup.html", pageData{Flash: "Could not set up: " + err.Error()})
		return
	}
	if err := s.startSession(w); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleUnlockForm(w http.ResponseWriter, r *http.Request) {
	switch s.app.Phase() {
	case app.PhaseNeedsSetup:
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	case app.PhaseUnlocked:
		if s.validSession(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}
	flash := ""
	if r.URL.Query().Get("timeout") == "1" {
		flash = "Locked after inactivity. Enter your passphrase to continue."
	}
	s.render(w, "unlock.html", pageData{Flash: flash})
}

func (s *Server) handleUnlock(w http.ResponseWriter, r *http.Request) {
	pass := r.FormValue("passphrase")
	if err := s.app.Unlock(pass); err != nil {
		s.render(w, "unlock.html", pageData{Flash: "Incorrect passphrase."})
		return
	}
	if err := s.startSession(w); err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	_ = s.app.Lock()
	s.clearSession()
	http.Redirect(w, r, "/unlock", http.StatusSeeOther)
}

// --- dashboard ---

type dashboardView struct {
	Summary   reconcile.Summary
	Practices map[string]model.Practice
	Payers    map[string]model.Payer
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	records, _ := st.Records()
	practices, _ := st.Practices()
	payers, _ := st.Payers()
	view := dashboardView{
		Summary: reconcile.BuildSummary(records, practices, payers),
	}
	s.render(w, "dashboard.html", pageData{Active: "dashboard", Content: view})
}

// --- sessions ---

type sessionsView struct {
	Records   []model.Record
	Practices []model.Practice
	Payers    []model.Payer
	PayerName map[string]string
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	records, _ := st.Records()
	practices, _ := st.Practices()
	payers, _ := st.Payers()
	names := map[string]string{}
	for _, p := range payers {
		names[p.ID] = p.Name
	}
	s.render(w, "sessions.html", pageData{Active: "sessions", Content: sessionsView{
		Records: records, Practices: practices, Payers: payers, PayerName: names,
	}})
}

func (s *Server) handleAddSession(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	date, derr := model.ParseDate(r.FormValue("date"))
	if derr != nil {
		http.Redirect(w, r, "/sessions", http.StatusSeeOther)
		return
	}
	payerID := r.FormValue("payer_id")
	service := r.FormValue("service")
	status := model.SessionStatus(r.FormValue("status"))
	if !status.Valid() {
		status = model.StatusCompleted
	}

	// Resolve practice from payer, and expected amount from rates.
	payers, _ := st.Payers()
	practiceID := ""
	for _, p := range payers {
		if p.ID == payerID {
			practiceID = p.PracticeID
		}
	}
	rates, _ := st.Rates()
	expected, _ := model.ExpectedFor(rates, payerID, service)
	if override := r.FormValue("expected"); override != "" {
		if c, err := model.ParseMoney(override); err == nil {
			expected = c
		}
	}

	rec := model.Record{
		ID:         newID(),
		Date:       date,
		ClientID:   r.FormValue("client_id"),
		PracticeID: practiceID,
		PayerID:    payerID,
		Service:    service,
		Status:     status,
		Expected:   expected,
		Source:     "manual",
	}
	if r.FormValue("paid") == "on" {
		rec.Paid = expected
		rec.DatePaid = date
	}
	_ = st.PutRecord(rec)
	http.Redirect(w, r, "/sessions", http.StatusSeeOther)
}

// --- rates ---

type ratesView struct {
	Rates     []model.Rate
	Payers    []model.Payer
	PayerName map[string]string
}

func (s *Server) handleRates(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	rates, _ := st.Rates()
	payers, _ := st.Payers()
	names := map[string]string{}
	for _, p := range payers {
		names[p.ID] = p.Name
	}
	sort.Slice(rates, func(i, j int) bool { return names[rates[i].PayerID] < names[rates[j].PayerID] })
	s.render(w, "rates.html", pageData{Active: "rates", Content: ratesView{Rates: rates, Payers: payers, PayerName: names}})
}

func (s *Server) handleAddRate(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	amount, merr := model.ParseMoney(r.FormValue("amount"))
	if merr != nil || r.FormValue("payer_id") == "" {
		http.Redirect(w, r, "/rates", http.StatusSeeOther)
		return
	}
	_ = st.PutRate(model.Rate{
		ID:      newID(),
		PayerID: r.FormValue("payer_id"),
		Service: r.FormValue("service"),
		Amount:  amount,
	})
	http.Redirect(w, r, "/rates", http.StatusSeeOther)
}

// --- settings (practices & payers) ---

type settingsView struct {
	Practices    []model.Practice
	Payers       []model.Payer
	PracticeName map[string]string
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	practices, _ := st.Practices()
	payers, _ := st.Payers()
	names := map[string]string{}
	for _, p := range practices {
		names[p.ID] = p.Name
	}
	s.render(w, "settings.html", pageData{Active: "settings", Content: settingsView{
		Practices: practices, Payers: payers, PracticeName: names,
	}})
}

func (s *Server) handleAddPractice(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	kind := model.PracticeKind(r.FormValue("kind"))
	if name == "" || (kind != model.PracticeOwn && kind != model.PracticeEmployer) {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	_ = st.PutPractice(model.Practice{ID: newID(), Name: name, Kind: kind})
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func (s *Server) handleAddPayer(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	name := r.FormValue("name")
	practiceID := r.FormValue("practice_id")
	kind := model.PayerKind(r.FormValue("kind"))
	if name == "" || practiceID == "" {
		http.Redirect(w, r, "/settings", http.StatusSeeOther)
		return
	}
	_ = st.PutPayer(model.Payer{
		ID:          newID(),
		Name:        name,
		PracticeID:  practiceID,
		Kind:        kind,
		ImporterKey: r.FormValue("importer_key"),
	})
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// --- quit ---

func (s *Server) handleQuit(w http.ResponseWriter, r *http.Request) {
	s.render(w, "quit.html", pageData{})
	if s.quit != nil {
		go func() {
			time.Sleep(300 * time.Millisecond)
			s.quit()
		}()
	}
}

// --- reset / uninstall ---

type resetView struct {
	DataDir  string
	Platform string // "darwin" | "windows" | "linux"
}

func (s *Server) handleResetForm(w http.ResponseWriter, r *http.Request) {
	s.render(w, "reset.html", pageData{
		Active:  "settings",
		Content: resetView{DataDir: s.app.Dir(), Platform: runtime.GOOS},
	})
}

func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("confirm") != "RESET" {
		s.render(w, "reset.html", pageData{
			Active:  "settings",
			Flash:   "Type RESET in capital letters to confirm.",
			Content: resetView{DataDir: s.app.Dir(), Platform: runtime.GOOS},
		})
		return
	}
	_ = s.app.Reset()
	s.clearSession()
	http.Redirect(w, r, "/setup", http.StatusSeeOther)
}

// --- import (Phase 2 placeholder) / export ---

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	s.render(w, "import.html", pageData{Active: "import"})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	st, err := s.app.Store()
	if err != nil {
		http.Redirect(w, r, "/unlock", http.StatusSeeOther)
		return
	}
	records, _ := st.Records()
	practices, _ := st.Practices()
	payers, _ := st.Payers()
	name := fmt.Sprintf("Tallywell-Export-%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	if err := report.WriteXLSX(w, report.Input{Records: records, Practices: practices, Payers: payers}); err != nil {
		http.Error(w, "export error", http.StatusInternalServerError)
	}
}
