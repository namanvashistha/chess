package live

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chess-engine/app/engine"
	"chess-engine/app/store"
)

// storeTimeout bounds each store call a room makes.
const storeTimeout = 5 * time.Second

// Room is one loaded game. Every field below the channels is owned by the
// room's goroutine and touched nowhere else.
type Room struct {
	id      string
	m       *Manager
	inbox   chan func()
	done    chan struct{} // closed when the room has shut down
	ready   chan struct{} // closed once load has finished
	loadErr error

	rec      store.Game // seats, mode and result; Moves is not kept up to date
	game     *engine.Game
	clients  map[*Client]struct{}
	thinking bool // a bot search is running
	idle     *time.Timer
}

func newRoom(m *Manager, id string) *Room {
	return &Room{id: id, m: m, inbox: make(chan func(), 64), done: make(chan struct{}),
		ready: make(chan struct{}), clients: map[*Client]struct{}{}}
}

// load reads the game from the store, replays it, and starts the room.
func (r *Room) load() {
	ctx, cancel := context.WithTimeout(context.Background(), storeTimeout)
	defer cancel()
	rec, err := r.m.store.Game(ctx, r.id)
	if err == nil {
		r.rec = rec
		r.game, err = replay(rec.Moves)
	}
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			err = ErrNotFound
		}
		r.loadErr = err
		r.m.forget(r)
		close(r.done)
		close(r.ready)
		return
	}
	close(r.ready)
	r.armIdle()
	r.maybeBot()
	r.run()
}

func replay(moves []store.Move) (*engine.Game, error) {
	g := engine.NewGame()
	for i, m := range moves {
		if _, err := g.Play(m.UCI); err != nil {
			return nil, fmt.Errorf("stored move %d (%s) is not legal: %w", i, m.UCI, err)
		}
	}
	return g, nil
}

func (r *Room) run() {
	for f := range r.inbox {
		f()
		if r.shouldStop() {
			r.m.forget(r)
			close(r.done)
			return
		}
	}
}

// shouldStop reports whether the room can unload: the idle timer has fired,
// no one is connected and no bot search is running.
func (r *Room) shouldStop() bool {
	return r.idle == nil && len(r.clients) == 0 && !r.thinking
}

// armIdle starts the countdown to unloading while no one is connected.
func (r *Room) armIdle() {
	if r.idle != nil {
		r.idle.Stop()
	}
	var timer *time.Timer
	timer = time.AfterFunc(r.m.opts.IdleAfter, func() {
		select {
		case r.inbox <- func() {
			if r.idle == timer && len(r.clients) == 0 {
				r.idle = nil // shouldStop now holds once nothing else is pending
			}
		}:
		case <-r.done:
		}
	})
	r.idle = timer
}

func (r *Room) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), storeTimeout)
}

// colorOf returns the viewer's colour: "w", "b", "both", or "" if they do not
// play in this game.
func (r *Room) colorOf(u *store.User) string {
	if u == nil {
		return ""
	}
	white := r.rec.White != nil && r.rec.White.ID == u.ID
	black := r.rec.Black != nil && r.rec.Black.ID == u.ID
	switch {
	case white && black:
		return "both"
	case white:
		return "w"
	case black:
		return "b"
	}
	return ""
}

func (r *Room) view(viewer *store.User) View {
	sans := r.game.SANs()
	moves := make([]store.Move, 0, len(sans))
	for i, uci := range r.game.Moves() {
		moves = append(moves, store.Move{UCI: uci, SAN: sans[i]})
	}
	return View{
		ID: r.rec.ID, Mode: r.rec.Mode, BotLevel: r.rec.BotLevel,
		White: r.rec.White, Black: r.rec.Black,
		Status: r.rec.Status(), Result: r.rec.Result, Reason: r.rec.Reason,
		Moves: moves, FEN: r.game.FEN(), Turn: r.game.Turn(), Check: r.game.InCheck(),
		Legal: r.legal(), You: r.colorOf(viewer),
	}
}

// legal is the legal moves, or none once the game has ended or while a seat
// is open.
func (r *Room) legal() []string {
	if r.rec.Status() != store.StatusActive {
		return []string{}
	}
	return r.game.Legal()
}

// attach adds a connected client and sends it the game.
func (r *Room) attach(c *Client) {
	r.clients[c] = struct{}{}
	if r.idle != nil {
		r.idle.Stop()
		r.idle = nil
	}
	c.queue(snapshotMsg{T: typeSnapshot, Game: r.view(c.user)})
}

func (r *Room) detach(c *Client) {
	if _, ok := r.clients[c]; !ok {
		return
	}
	delete(r.clients, c)
	if len(r.clients) == 0 {
		r.armIdle()
	}
}

func (r *Room) closeClients() {
	for c := range r.clients {
		c.close()
	}
}

func (r *Room) broadcast(msg any) {
	for c := range r.clients {
		c.queue(msg)
	}
}

func (r *Room) fail(c *Client, code, message string) {
	c.queue(errorMsg{T: typeError, Code: code, Message: message})
}

// handle processes one message from a client.
func (r *Room) handle(c *Client, in inbound) {
	if in.T != typeMove {
		r.fail(c, CodeBadMessage, "unknown message type")
		return
	}
	color := r.colorOf(c.user)
	if color == "" {
		r.fail(c, CodeNotYourTurn, "you are watching this game")
		return
	}
	if color == "both" {
		color = r.game.Turn()
	}
	r.play(c, color, in.Ply, in.UCI)
}

// play applies a move by the given colour. c is the client that sent it, or
// nil for the bot. Every rule is checked here, in one place.
func (r *Room) play(c *Client, color string, ply int, uci string) {
	reject := func(code, message string) {
		if c != nil {
			r.fail(c, code, message)
			if code == CodeStale || code == CodeSaveFailed {
				c.queue(snapshotMsg{T: typeSnapshot, Game: r.view(c.user)})
			}
		}
	}
	switch {
	case r.rec.Result != "":
		reject(CodeGameOver, "the game is over")
		return
	case r.rec.Status() != store.StatusActive:
		reject(CodeNotYourTurn, "waiting for an opponent")
		return
	case color != r.game.Turn():
		reject(CodeNotYourTurn, "it is not your turn")
		return
	case ply != r.game.Ply():
		reject(CodeStale, "that move is for a different position")
		return
	}
	san, err := r.game.Play(uci)
	if err != nil {
		reject(CodeIllegal, "that move is not legal")
		return
	}

	ctx, cancel := r.ctx()
	defer cancel()
	if err := r.m.store.AppendMove(ctx, r.id, ply, store.Move{UCI: uci, SAN: san}); err != nil {
		r.game.Undo()
		r.m.log.Error("save move", "game", r.id, "ply", ply, "err", err)
		reject(CodeSaveFailed, "the move could not be saved")
		return
	}
	if result, reason := r.game.Outcome(); result != engine.ResultNone {
		if err := r.m.store.Finish(ctx, r.id, result, reason); err != nil {
			// The move is saved and the result follows from the moves, so the
			// game still reads as over to everyone; only the record lags.
			r.m.log.Error("save result", "game", r.id, "err", err)
		}
		r.rec.Result, r.rec.Reason = result, reason
	}

	r.broadcast(moveMsg{T: typeMove, Ply: ply, UCI: uci, SAN: san, FEN: r.game.FEN(),
		Turn: r.game.Turn(), Check: r.game.InCheck(), Legal: r.legal(),
		Status: r.rec.Status(), Result: r.rec.Result, Reason: r.rec.Reason})
	r.maybeBot()
}

// join seats a user in the open seat of an online game.
func (r *Room) join(u store.User) (string, error) {
	if c := r.colorOf(&u); c != "" {
		return c, nil
	}
	if r.rec.Mode != store.ModeOnline || r.rec.Result != "" {
		return "", ErrCannotJoin
	}
	color := ""
	switch {
	case r.rec.White == nil:
		color = "w"
	case r.rec.Black == nil:
		color = "b"
	default:
		return "", ErrGameFull
	}
	ctx, cancel := r.ctx()
	defer cancel()
	if err := r.m.store.TakeSeat(ctx, r.id, color, u.ID); err != nil {
		if errors.Is(err, store.ErrSeatTaken) {
			return "", ErrGameFull
		}
		return "", err
	}
	if color == "w" {
		r.rec.White = &u
	} else {
		r.rec.Black = &u
	}
	// Seating changes what everyone may do, so each client gets a fresh view.
	for c := range r.clients {
		c.queue(snapshotMsg{T: typeSnapshot, Game: r.view(c.user)})
	}
	return color, nil
}

// maybeBot starts the bot's search if it is the bot's turn.
func (r *Room) maybeBot() {
	if r.rec.Mode != store.ModeBot || r.rec.Result != "" || r.thinking {
		return
	}
	seat := r.rec.White
	if r.game.Turn() == "b" {
		seat = r.rec.Black
	}
	if seat == nil || !seat.Bot {
		return
	}
	r.thinking = true
	color, ply, level := r.game.Turn(), r.game.Ply(), r.rec.BotLevel
	moves := r.game.Moves()
	go func() {
		start := time.Now()
		uci, ok := r.m.botMove(moves, level)
		if wait := r.m.opts.BotDelay - time.Since(start); wait > 0 {
			time.Sleep(wait)
		}
		r.inbox <- func() {
			r.thinking = false
			if ok {
				r.play(nil, color, ply, uci)
			}
		}
	}()
}

// botMove runs one bot search on a worker slot.
func (m *Manager) botMove(moves []string, level string) (string, bool) {
	table := <-m.tables
	defer func() { m.tables <- table }()
	g, err := engine.ReplayGame(moves)
	if err != nil {
		m.log.Error("bot replay", "err", err)
		return "", false
	}
	return engine.BotMove(g, level, table)
}
