// Package server exposes Tallywell's local web UI over loopback only. It gates
// the app pages behind unlock, issues a session cookie after unlock, and
// auto-locks after a period of inactivity.
package server

import (
	"crypto/rand"
	"encoding/hex"
	"embed"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/tallywell/tallywell/internal/app"
)

//go:embed web/templates/*.html web/static/*
var webFS embed.FS

// DefaultAutoLock is the default idle timeout before the app auto-locks.
const DefaultAutoLock = 15 * time.Minute

const sessionCookie = "tw_session"

// Server holds the app core, parsed templates, and session state.
type Server struct {
	app       *app.App
	tmpl      *template.Template
	mux       *http.ServeMux
	autoLock  time.Duration
	now       func() time.Time

	mu         sync.Mutex
	sessionID  string
	lastActive time.Time
}

// New builds a Server over the given app core.
func New(a *app.App, autoLock time.Duration) (*Server, error) {
	if autoLock <= 0 {
		autoLock = DefaultAutoLock
	}
	tmpl, err := template.New("").Funcs(templateFuncs()).ParseFS(webFS, "web/templates/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{
		app:      a,
		tmpl:     tmpl,
		autoLock: autoLock,
		now:      time.Now,
	}
	s.routes()
	return s, nil
}

// Handler returns the HTTP handler (loopback enforcement happens at the listener
// in main, by binding 127.0.0.1).
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux = http.NewServeMux()
	s.mux.Handle("GET /static/", http.FileServer(http.FS(webFS)))

	s.mux.HandleFunc("GET /setup", s.handleSetupForm)
	s.mux.HandleFunc("POST /setup", s.handleSetup)
	s.mux.HandleFunc("GET /unlock", s.handleUnlockForm)
	s.mux.HandleFunc("POST /unlock", s.handleUnlock)
	s.mux.HandleFunc("POST /lock", s.handleLock)

	s.mux.HandleFunc("GET /{$}", s.guard(s.handleDashboard))
	s.mux.HandleFunc("GET /sessions", s.guard(s.handleSessions))
	s.mux.HandleFunc("POST /sessions", s.guard(s.handleAddSession))
	s.mux.HandleFunc("GET /rates", s.guard(s.handleRates))
	s.mux.HandleFunc("POST /rates", s.guard(s.handleAddRate))
	s.mux.HandleFunc("GET /settings", s.guard(s.handleSettings))
	s.mux.HandleFunc("POST /settings/practice", s.guard(s.handleAddPractice))
	s.mux.HandleFunc("POST /settings/payer", s.guard(s.handleAddPayer))
	s.mux.HandleFunc("GET /import", s.guard(s.handleImport))
	s.mux.HandleFunc("GET /export", s.guard(s.handleExport))
}

// guard wraps a handler so it only runs when the app is unlocked with a valid
// session; otherwise it redirects to setup or unlock. It also enforces the
// idle auto-lock and refreshes the activity timestamp.
func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch s.app.Phase() {
		case app.PhaseNeedsSetup:
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		case app.PhaseLocked:
			http.Redirect(w, r, "/unlock", http.StatusSeeOther)
			return
		}
		if !s.validSession(r) {
			http.Redirect(w, r, "/unlock", http.StatusSeeOther)
			return
		}
		if s.idleExpired() {
			_ = s.app.Lock()
			s.clearSession()
			http.Redirect(w, r, "/unlock?timeout=1", http.StatusSeeOther)
			return
		}
		s.touch()
		h(w, r)
	}
}

// --- session helpers ---

func (s *Server) startSession(w http.ResponseWriter) error {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	id := hex.EncodeToString(buf)
	s.mu.Lock()
	s.sessionID = id
	s.lastActive = s.now()
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return nil
}

func (s *Server) validSession(r *http.Request) bool {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionID != "" && c.Value == s.sessionID
}

func (s *Server) clearSession() {
	s.mu.Lock()
	s.sessionID = ""
	s.mu.Unlock()
}

func (s *Server) idleExpired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.now().Sub(s.lastActive) > s.autoLock
}

func (s *Server) touch() {
	s.mu.Lock()
	s.lastActive = s.now()
	s.mu.Unlock()
}
