package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chess-engine/app/live"
	"chess-engine/app/store"

	"github.com/gorilla/websocket"
)

// These tests drive the real server over real HTTP and WebSocket connections,
// backed by the in-memory store.

type env struct {
	t     *testing.T
	srv   *httptest.Server
	store *store.Memory
	live  *live.Manager
}

func newEnv(t *testing.T, opts live.Options) *env {
	t.Helper()
	web := t.TempDir()
	if err := os.WriteFile(filepath.Join(web, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if opts.BotDelay == 0 {
		opts.BotDelay = time.Millisecond
	}
	mem := store.NewMemory()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := live.NewManager(mem, log, opts)
	srv := httptest.NewServer(New(Config{Store: mem, Live: m, Log: log, WebDir: web, Static: web}))
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, store: mem, live: m}
}

// player is one browser: a cookie jar and its sockets.
type player struct {
	e    *env
	http *http.Client
	user store.User
}

func (e *env) player() *player {
	jar, _ := cookiejar.New(nil)
	p := &player{e: e, http: &http.Client{Jar: jar}}
	var s sessionResponse
	p.call("POST", "/api/session", nil, http.StatusCreated, &s)
	p.user = s.User
	return p
}

func (p *player) call(method, path string, body any, want int, out any) {
	p.e.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, p.e.srv.URL+path, r)
	req.Header.Set("Origin", p.e.srv.URL)
	res, err := p.http.Do(req)
	if err != nil {
		p.e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		p.e.t.Fatalf("%s %s: status %d, want %d: %s", method, path, res.StatusCode, want, data)
	}
	if out != nil {
		if s, ok := out.(*string); ok {
			*s = string(data)
		} else if err := json.Unmarshal(data, out); err != nil {
			p.e.t.Fatalf("%s %s: %v: %s", method, path, err, data)
		}
	}
}

func (p *player) create(body map[string]any) string {
	var out struct{ ID string }
	p.call("POST", "/api/games", body, http.StatusCreated, &out)
	return out.ID
}

type sock struct {
	t    *testing.T
	conn *websocket.Conn
}

// connect opens the game's socket; p may be nil for a signed-out spectator.
func (e *env) connect(p *player, gameID string) *sock {
	e.t.Helper()
	u := "ws" + strings.TrimPrefix(e.srv.URL, "http") + "/ws/games/" + gameID
	h := http.Header{"Origin": {e.srv.URL}}
	if p != nil {
		base, _ := url.Parse(e.srv.URL)
		for _, c := range p.http.Jar.Cookies(base) {
			h.Add("Cookie", c.String())
		}
	}
	conn, res, err := websocket.DefaultDialer.Dial(u, h)
	if err != nil {
		status := 0
		if res != nil {
			status = res.StatusCode
		}
		e.t.Fatalf("dial %s: %v (status %d)", gameID, err, status)
	}
	e.t.Cleanup(func() { conn.Close() })
	return &sock{t: e.t, conn: conn}
}

type msg struct {
	T       string     `json:"t"`
	Game    live.View  `json:"game"`
	Ply     int        `json:"ply"`
	UCI     string     `json:"uci"`
	SAN     string     `json:"san"`
	Turn    string     `json:"turn"`
	Legal   []string   `json:"legal"`
	Status  string     `json:"status"`
	Result  string     `json:"result"`
	Reason  string     `json:"reason"`
	Code    string     `json:"code"`
	Message string     `json:"message"`
	White   store.User `json:"-"`
}

func (s *sock) read() msg {
	s.t.Helper()
	s.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var m msg
	if err := s.conn.ReadJSON(&m); err != nil {
		s.t.Fatalf("read: %v", err)
	}
	return m
}

func (s *sock) expect(typ string) msg {
	s.t.Helper()
	m := s.read()
	if m.T != typ {
		s.t.Fatalf("got %q message %+v, want %q", m.T, m, typ)
	}
	return m
}

func (s *sock) move(ply int, uci string) {
	s.t.Helper()
	if err := s.conn.WriteJSON(map[string]any{"t": "move", "ply": ply, "uci": uci}); err != nil {
		s.t.Fatal(err)
	}
}

func TestOnlineGameToCheckmate(t *testing.T) {
	e := newEnv(t, live.Options{})
	alice, bob := e.player(), e.player()
	id := alice.create(map[string]any{"mode": "online", "color": "w"})

	a := e.connect(alice, id)
	if snap := a.expect("snapshot"); snap.Game.You != "w" || snap.Game.Status != store.StatusWaiting || len(snap.Game.Legal) != 0 {
		t.Fatalf("creator's first snapshot = %+v", snap.Game)
	}
	// Moving before anyone has joined is refused, and the game stays usable.
	a.move(0, "f2f3")
	if m := a.expect("error"); m.Code != live.CodeNotYourTurn {
		t.Fatalf("early move: %+v", m)
	}

	var lobby store.Lobby
	bob.call("GET", "/api/lobby", nil, http.StatusOK, &lobby)
	if len(lobby.Open) != 1 || lobby.Open[0].ID != id {
		t.Fatalf("bob's lobby open = %+v", lobby.Open)
	}
	var joined struct{ Color string }
	bob.call("POST", "/api/games/"+id+"/join", nil, http.StatusOK, &joined)
	if joined.Color != "b" {
		t.Fatalf("bob joined as %q", joined.Color)
	}
	if snap := a.expect("snapshot"); snap.Game.Status != store.StatusActive || len(snap.Game.Legal) != 20 {
		t.Fatalf("snapshot after join = %+v", snap.Game)
	}

	b := e.connect(bob, id)
	if snap := b.expect("snapshot"); snap.Game.You != "b" || snap.Game.Black == nil || snap.Game.Black.ID != bob.user.ID {
		t.Fatalf("bob's snapshot = %+v", snap.Game)
	}
	watcher := e.connect(nil, id)
	watcher.expect("snapshot")

	plays := []struct {
		s    *sock
		uci  string
		want string
	}{{a, "f2f3", "f3"}, {b, "e7e5", "e5"}, {a, "g2g4", "g4"}, {b, "d8h4", "Qh4#"}}
	for ply, p := range plays {
		p.s.move(ply, p.uci)
		for _, s := range []*sock{a, b, watcher} {
			m := s.expect("move")
			if m.Ply != ply || m.SAN != p.want {
				t.Fatalf("ply %d: got %+v, want %s", ply, m, p.want)
			}
			if ply == 3 && (m.Result != "0-1" || m.Reason != "checkmate" || m.Status != store.StatusFinished || len(m.Legal) != 0) {
				t.Fatalf("mating move: %+v", m)
			}
		}
	}

	a.move(4, "a2a3")
	if m := a.expect("error"); m.Code != live.CodeGameOver {
		t.Fatalf("move after mate: %+v", m)
	}
	var pgn string
	bob.call("GET", "/api/games/"+id+"/pgn", nil, http.StatusOK, &pgn)
	if !strings.Contains(pgn, `[Result "0-1"]`) || !strings.Contains(pgn, "1. f3 e5 2. g4 Qh4# 0-1") {
		t.Errorf("PGN:\n%s", pgn)
	}
	rec, _ := e.store.Game(nil, id)
	if rec.Result != "0-1" || rec.Reason != "checkmate" || len(rec.Moves) != 4 {
		t.Errorf("stored game = %+v", rec)
	}
}

func TestRejectedMoves(t *testing.T) {
	e := newEnv(t, live.Options{})
	alice, bob := e.player(), e.player()
	id := alice.create(map[string]any{"mode": "online", "color": "w"})
	bob.call("POST", "/api/games/"+id+"/join", nil, http.StatusOK, nil)
	a, b := e.connect(alice, id), e.connect(bob, id)
	a.expect("snapshot")
	b.expect("snapshot")

	b.move(0, "e7e5")
	if m := b.expect("error"); m.Code != live.CodeNotYourTurn {
		t.Errorf("black moving first: %+v", m)
	}
	a.move(0, "e2e5")
	if m := a.expect("error"); m.Code != live.CodeIllegal {
		t.Errorf("illegal move: %+v", m)
	}
	a.move(0, "e2e4")
	a.expect("move")
	b.expect("move")

	// A resent move (same ply) is stale: rejected, with a fresh snapshot.
	b.move(0, "e7e5")
	if m := b.expect("error"); m.Code != live.CodeStale {
		t.Errorf("stale move: %+v", m)
	}
	if snap := b.expect("snapshot"); len(snap.Game.Moves) != 1 {
		t.Errorf("resync snapshot has %d moves", len(snap.Game.Moves))
	}

	if err := b.conn.WriteMessage(websocket.TextMessage, []byte("not json")); err != nil {
		t.Fatal(err)
	}
	if m := b.expect("error"); m.Code != live.CodeBadMessage {
		t.Errorf("garbage message: %+v", m)
	}

	watcher := e.connect(nil, id)
	watcher.expect("snapshot")
	watcher.move(1, "e7e5")
	if m := watcher.expect("error"); m.Code != live.CodeNotYourTurn {
		t.Errorf("spectator move: %+v", m)
	}

	carol := e.player()
	carol.call("POST", "/api/games/"+id+"/join", nil, http.StatusConflict, nil)
}

func TestBotReplies(t *testing.T) {
	e := newEnv(t, live.Options{})
	alice := e.player()
	id := alice.create(map[string]any{"mode": "bot", "color": "w", "level": "easy"})
	a := e.connect(alice, id)
	if snap := a.expect("snapshot"); snap.Game.Status != store.StatusActive || snap.Game.Black == nil || !snap.Game.Black.Bot {
		t.Fatalf("bot game snapshot = %+v", snap.Game)
	}
	a.move(0, "e2e4")
	a.expect("move")
	if m := a.expect("move"); m.Ply != 1 || m.Turn != "w" {
		t.Fatalf("bot reply = %+v", m)
	}
	alice.call("POST", "/api/games/"+id+"/join", nil, http.StatusOK, nil) // already seated: fine
	e.player().call("POST", "/api/games/"+id+"/join", nil, http.StatusConflict, nil)
}

func TestBotMovesFirstAsWhite(t *testing.T) {
	e := newEnv(t, live.Options{})
	alice := e.player()
	id := alice.create(map[string]any{"mode": "bot", "color": "b", "level": "hard"})
	a := e.connect(alice, id)
	snap := a.expect("snapshot")
	if len(snap.Game.Moves) == 0 {
		if m := a.expect("move"); m.Ply != 0 {
			t.Fatalf("bot opening move = %+v", m)
		}
	}
}

func TestLocalGameAndRepetition(t *testing.T) {
	e := newEnv(t, live.Options{})
	alice := e.player()
	id := alice.create(map[string]any{"mode": "local"})
	a := e.connect(alice, id)
	if snap := a.expect("snapshot"); snap.Game.You != "both" {
		t.Fatalf("local game: you = %q", snap.Game.You)
	}
	shuffle := []string{"g1f3", "g8f6", "f3g1", "f6g8", "g1f3", "g8f6", "f3g1", "f6g8"}
	var last msg
	for ply, uci := range shuffle {
		a.move(ply, uci)
		last = a.expect("move")
	}
	if last.Result != "1/2-1/2" || last.Reason != "repetition" {
		t.Fatalf("after the third repetition: %+v", last)
	}
}

func TestRoomUnloadsAndReloads(t *testing.T) {
	e := newEnv(t, live.Options{IdleAfter: 30 * time.Millisecond})
	alice := e.player()
	id := alice.create(map[string]any{"mode": "local"})
	a := e.connect(alice, id)
	a.expect("snapshot")
	a.move(0, "d2d4")
	a.expect("move")
	a.conn.Close()

	deadline := time.Now().Add(2 * time.Second)
	for e.live.Rooms() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("room still loaded: %d rooms", e.live.Rooms())
		}
		time.Sleep(10 * time.Millisecond)
	}
	// The game comes back from the store with its moves.
	a = e.connect(alice, id)
	if snap := a.expect("snapshot"); len(snap.Game.Moves) != 1 || snap.Game.Turn != "b" {
		t.Fatalf("reloaded snapshot = %+v", snap.Game)
	}
	a.move(1, "d7d5")
	if m := a.expect("move"); m.SAN != "d5" {
		t.Fatalf("move after reload = %+v", m)
	}
}

func TestAPIEdges(t *testing.T) {
	e := newEnv(t, live.Options{})
	alice := e.player()

	var me store.User
	alice.call("GET", "/api/me", nil, http.StatusOK, &me)
	if me.ID != alice.user.ID {
		t.Errorf("me = %+v", me)
	}
	alice.call("PATCH", "/api/me", map[string]string{"name": "  Ada  "}, http.StatusOK, &me)
	if me.Name != "Ada" {
		t.Errorf("renamed to %q", me.Name)
	}
	alice.call("PATCH", "/api/me", map[string]string{"name": ""}, http.StatusBadRequest, nil)
	alice.call("POST", "/api/games", map[string]any{"mode": "chaos"}, http.StatusBadRequest, nil)
	alice.call("POST", "/api/games", map[string]any{"mode": "bot", "level": "grandmaster"}, http.StatusBadRequest, nil)
	alice.call("GET", "/api/games/nope", nil, http.StatusNotFound, nil)
	alice.call("GET", "/api/nothing", nil, http.StatusNotFound, nil)

	// A cross-site POST is refused even with a valid session cookie.
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/games", strings.NewReader(`{"mode":"local"}`))
	req.Header.Set("Origin", "https://evil.example")
	res, err := alice.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("cross-site POST: status %d, want 403", res.StatusCode)
	}

	// Signed out: reading works, acting does not.
	anon := &player{e: e, http: &http.Client{}}
	anon.call("GET", "/api/lobby", nil, http.StatusOK, nil)
	anon.call("POST", "/api/games", map[string]any{"mode": "local"}, http.StatusUnauthorized, nil)

	// The web app's routes fall back to index.html.
	var page string
	anon.call("GET", "/game/abc", nil, http.StatusOK, &page)
	if !strings.Contains(page, "app") {
		t.Errorf("SPA fallback served %q", page)
	}

	alice.call("DELETE", "/api/me", nil, http.StatusNoContent, nil)
	alice.call("GET", "/api/me", nil, http.StatusUnauthorized, nil)
}
