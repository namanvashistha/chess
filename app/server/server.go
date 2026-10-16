// Package server is the HTTP side: the JSON API, the WebSocket endpoint, and
// the web app's static files.
package server

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"chess-engine/app/live"
	"chess-engine/app/store"

	"github.com/gorilla/websocket"
)

// Config wires the server to its dependencies.
type Config struct {
	Store   store.Store
	Live    *live.Manager
	Log     *slog.Logger
	WebDir  string // the built web app (web/build)
	Static  string // piece images and other legacy assets (app/static)
	Origins []string
	// TrustProxy reads the client address from X-Forwarded-For. Set it only
	// behind a proxy that overwrites that header.
	TrustProxy bool
}

// Server handles every HTTP request.
type Server struct {
	cfg      Config
	mux      *http.ServeMux
	upgrader websocket.Upgrader
	limits   *limiter
}

// New returns the server's handler.
func New(cfg Config) http.Handler {
	s := &Server{cfg: cfg, mux: http.NewServeMux(), limits: newLimiter()}
	s.upgrader = websocket.Upgrader{CheckOrigin: s.originAllowed}

	s.mux.HandleFunc("GET /health", s.health)

	s.mux.HandleFunc("POST /api/session", s.limit("session", 10, s.createSession))
	s.mux.HandleFunc("GET /api/me", s.me)
	s.mux.HandleFunc("PATCH /api/me", s.rename)
	s.mux.HandleFunc("DELETE /api/me", s.deleteMe)

	s.mux.HandleFunc("GET /api/lobby", s.lobby)
	s.mux.HandleFunc("POST /api/games", s.limit("games", 30, s.createGame))
	s.mux.HandleFunc("GET /api/games/{id}", s.getGame)
	s.mux.HandleFunc("POST /api/games/{id}/join", s.joinGame)
	s.mux.HandleFunc("GET /api/games/{id}/pgn", s.pgn)

	s.mux.HandleFunc("GET /ws/games/{id}", s.socket)

	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.Static))))
	s.mux.HandleFunc("GET /", s.webApp)

	return s.recoverer(s.logRequests(s.checkOrigin(s.mux)))
}

// webApp serves the built web app. Paths that are not files fall back to
// index.html, so the client-side router can handle /game/{id} and friends.
func (s *Server) webApp(w http.ResponseWriter, r *http.Request) {
	clean := filepath.Clean("/" + r.URL.Path)
	if strings.HasPrefix(clean, "/api/") || strings.HasPrefix(clean, "/ws/") {
		writeError(w, http.StatusNotFound, "not_found", "no such endpoint")
		return
	}
	path := filepath.Join(s.cfg.WebDir, clean)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		if strings.HasPrefix(clean, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		http.ServeFile(w, r, path)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filepath.Join(s.cfg.WebDir, "index.html"))
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.cfg.Store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "database": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "rooms": s.cfg.Live.Rooms()})
}

// originAllowed reports whether a browser request comes from this site or an
// allowed origin. Requests without an Origin header come from non-browser
// clients, which carry no ambient cookies to abuse.
func (s *Server) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, o := range s.cfg.Origins {
		if o == "*" || strings.EqualFold(o, origin) {
			return true
		}
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// checkOrigin rejects cross-site requests that change state. Session cookies
// are SameSite=Lax, so this is a second line of defence against CSRF.
func (s *Server) checkOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if !s.originAllowed(r) {
				writeError(w, http.StatusForbidden, "bad_origin", "cross-site request refused")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Hijack hands the connection to the WebSocket upgrader, which asserts
// http.Hijacker directly.
func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.status = http.StatusSwitchingProtocols
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		if r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/_app/") || strings.HasPrefix(r.URL.Path, "/static/") {
			return
		}
		s.cfg.Log.Info("request", "method", r.Method, "path", r.URL.Path, "status", sw.status,
			"ms", time.Since(start).Milliseconds())
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				s.cfg.Log.Error("panic", "path", r.URL.Path, "panic", v)
				writeError(w, http.StatusInternalServerError, "internal", "something went wrong")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends {"error": {"code": ..., "message": ...}}. The code is for
// programs; the message is for people.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

// decode reads a small JSON body into v.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "the request body is not valid JSON for this endpoint")
		return false
	}
	return true
}
