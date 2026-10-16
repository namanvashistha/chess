package server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"chess-engine/app/engine"
	"chess-engine/app/live"
	"chess-engine/app/store"
)

const lobbyLimit = 30

func (s *Server) lobby(w http.ResponseWriter, r *http.Request) {
	u, err := s.user(r)
	if err != nil {
		s.internal(w, "load user", err)
		return
	}
	var id int64
	if u != nil {
		id = u.ID
	}
	l, err := s.cfg.Store.Lobby(r.Context(), id, lobbyLimit)
	if err != nil {
		s.internal(w, "lobby", err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// createGameRequest is the body of POST /api/games.
type createGameRequest struct {
	Mode  string `json:"mode"`  // "online", "bot" or "local"
	Color string `json:"color"` // "w", "b" or "random" (the default); ignored for local
	Level string `json:"level"` // bot level: "easy", "medium" (default) or "hard"
	// Listed puts an online game in the lobby. It defaults to true; false makes
	// a private game reachable only by its link.
	Listed *bool `json:"listed"`
}

func (s *Server) createGame(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	var req createGameRequest
	if !decode(w, r, &req) {
		return
	}

	g := store.NewGame{ID: newGameID(), Mode: req.Mode}
	color := req.Color
	switch color {
	case "", "random":
		color = "w"
		if coinFlip() {
			color = "b"
		}
	case "w", "b":
	default:
		writeError(w, http.StatusBadRequest, "bad_color", `color must be "w", "b" or "random"`)
		return
	}
	seat := func(you, other int64) {
		if color == "w" {
			g.WhiteID, g.BlackID = you, other
		} else {
			g.WhiteID, g.BlackID = other, you
		}
	}

	switch req.Mode {
	case store.ModeOnline:
		seat(u.ID, 0)
		g.Listed = req.Listed == nil || *req.Listed
	case store.ModeBot:
		g.BotLevel = req.Level
		if g.BotLevel == "" {
			g.BotLevel = engine.LevelMedium
		}
		if !engine.ValidLevel(g.BotLevel) {
			writeError(w, http.StatusBadRequest, "bad_level", `level must be "easy", "medium" or "hard"`)
			return
		}
		bot, err := s.cfg.Store.BotUser(r.Context())
		if err != nil {
			s.internal(w, "bot user", err)
			return
		}
		seat(u.ID, bot.ID)
	case store.ModeLocal:
		g.WhiteID, g.BlackID = u.ID, u.ID
	default:
		writeError(w, http.StatusBadRequest, "bad_mode", `mode must be "online", "bot" or "local"`)
		return
	}

	if err := s.cfg.Store.CreateGame(r.Context(), g); err != nil {
		s.internal(w, "create game", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": g.ID})
}

func (s *Server) getGame(w http.ResponseWriter, r *http.Request) {
	u, err := s.user(r)
	if err != nil {
		s.internal(w, "load user", err)
		return
	}
	v, err := s.cfg.Live.Snapshot(r.Context(), r.PathValue("id"), u)
	if err != nil {
		s.gameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) joinGame(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	color, err := s.cfg.Live.Join(r.Context(), r.PathValue("id"), *u)
	if err != nil {
		s.gameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"color": color})
}

func (s *Server) gameError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, live.ErrNotFound):
		writeError(w, http.StatusNotFound, "game_not_found", "there is no game at this link")
	case errors.Is(err, live.ErrGameFull):
		writeError(w, http.StatusConflict, "game_full", "both seats are taken")
	case errors.Is(err, live.ErrCannotJoin):
		writeError(w, http.StatusConflict, "cannot_join", "this game cannot be joined")
	case errors.Is(err, context.Canceled):
	default:
		s.internal(w, "game", err)
	}
}

// socket upgrades to the game's WebSocket. Signing in is optional: without a
// session the connection watches.
func (s *Server) socket(w http.ResponseWriter, r *http.Request) {
	u, err := s.user(r)
	if err != nil {
		s.internal(w, "load user", err)
		return
	}
	id := r.PathValue("id")
	// Check the game exists before upgrading, so a bad link gets a plain 404.
	if _, err := s.cfg.Live.Snapshot(r.Context(), id, nil); err != nil {
		s.gameError(w, err)
		return
	}
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // the upgrader has written the error
	}
	if err := s.cfg.Live.Serve(context.Background(), id, conn, u); err != nil {
		s.cfg.Log.Warn("socket", "game", id, "err", err)
	}
}

// pgn serves the game as a PGN file.
func (s *Server) pgn(w http.ResponseWriter, r *http.Request) {
	g, err := s.cfg.Store.Game(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "game_not_found", "there is no game at this link")
		return
	}
	if err != nil {
		s.internal(w, "pgn", err)
		return
	}
	w.Header().Set("Content-Type", "application/x-chess-pgn")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="chess-%s.pgn"`, g.ID))
	_, _ = w.Write([]byte(PGN(g, r.Host)))
}

// PGN renders a game in Portable Game Notation.
func PGN(g store.Game, site string) string {
	name := func(u *store.User) string {
		if u == nil {
			return "?"
		}
		return strings.ReplaceAll(u.Name, `"`, "'")
	}
	result := g.Result
	if result == "" {
		result = "*"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "[Event \"Casual game\"]\n[Site \"%s\"]\n[Date \"%s\"]\n", site, g.CreatedAt.UTC().Format("2006.01.02"))
	fmt.Fprintf(&b, "[White \"%s\"]\n[Black \"%s\"]\n[Result \"%s\"]\n", name(g.White), name(g.Black), result)
	if g.Reason != "" {
		fmt.Fprintf(&b, "[Termination \"%s\"]\n", strings.ReplaceAll(g.Reason, "_", " "))
	}
	b.WriteString("\n")

	line := 0
	write := func(tok string) {
		if line > 0 && line+1+len(tok) > 80 {
			b.WriteString("\n")
			line = 0
		} else if line > 0 {
			b.WriteString(" ")
			line++
		}
		b.WriteString(tok)
		line += len(tok)
	}
	for i, m := range g.Moves {
		if i%2 == 0 {
			write(fmt.Sprintf("%d.", i/2+1))
		}
		write(m.SAN)
	}
	write(result)
	b.WriteString("\n")
	return b.String()
}

const idAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// newGameID returns 10 random characters (about 58 bits). It is the game's
// share link, so it must not be guessable.
func newGameID() string {
	b := make([]byte, 10)
	max := big.NewInt(int64(len(idAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic("crypto/rand failed: " + err.Error())
		}
		b[i] = idAlphabet[n.Int64()]
	}
	return string(b)
}

func coinFlip() bool {
	n, _ := rand.Int(rand.Reader, big.NewInt(2))
	return n.Int64() == 1
}

// limiter is a per-client fixed-window rate limit on expensive endpoints.
type limiter struct {
	mu      sync.Mutex
	windows map[string]*window
	lastGC  time.Time
}

type window struct {
	start time.Time
	count int
}

func newLimiter() *limiter {
	return &limiter{windows: map[string]*window{}}
}

func (l *limiter) allow(key string, perMinute int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if now.Sub(l.lastGC) > 5*time.Minute {
		for k, w := range l.windows {
			if now.Sub(w.start) > time.Minute {
				delete(l.windows, k)
			}
		}
		l.lastGC = now
	}
	w, ok := l.windows[key]
	if !ok || now.Sub(w.start) > time.Minute {
		w = &window{start: now}
		l.windows[key] = w
	}
	w.count++
	return w.count <= perMinute
}

// limit wraps a handler with a per-client limit of perMinute requests.
func (s *Server) limit(name string, perMinute int, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.limits.allow(name+"|"+s.clientIP(r), perMinute) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again in a minute")
			return
		}
		next(w, r)
	}
}

func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if f := r.Header.Get("X-Forwarded-For"); f != "" {
			return strings.TrimSpace(strings.Split(f, ",")[0])
		}
	}
	host := r.RemoteAddr
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	return host
}
