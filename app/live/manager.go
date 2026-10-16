package live

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"chess-engine/app/engine"
	"chess-engine/app/store"
)

// Errors the manager returns to the HTTP layer.
var (
	ErrNotFound   = errors.New("game not found")
	ErrCannotJoin = errors.New("this game cannot be joined")
	ErrGameFull   = errors.New("this game is full")
)

// Options tunes a Manager. Zero values pick the defaults.
type Options struct {
	// IdleAfter is how long a room with no one connected stays loaded.
	IdleAfter time.Duration
	// BotDelay is the least time the bot takes to reply, so its moves don't
	// land before the player's own piece has finished moving.
	BotDelay time.Duration
	// BotWorkers caps how many bot searches run at once.
	BotWorkers int
}

// Manager owns the rooms of the games in progress.
type Manager struct {
	store store.Store
	log   *slog.Logger
	opts  Options

	mu    sync.Mutex
	rooms map[string]*Room

	// tables is a pool of transposition tables, one per bot worker; taking one
	// is how a search gets a worker slot.
	tables chan *engine.TranspositionTable
}

// NewManager returns a Manager over s.
func NewManager(s store.Store, log *slog.Logger, opts Options) *Manager {
	if opts.IdleAfter == 0 {
		opts.IdleAfter = 2 * time.Minute
	}
	if opts.BotDelay == 0 {
		opts.BotDelay = 400 * time.Millisecond
	}
	if opts.BotWorkers == 0 {
		opts.BotWorkers = max(1, runtime.NumCPU()-1)
	}
	m := &Manager{store: s, log: log, opts: opts, rooms: map[string]*Room{},
		tables: make(chan *engine.TranspositionTable, opts.BotWorkers)}
	for range opts.BotWorkers {
		m.tables <- engine.NewTranspositionTable(16)
	}
	return m
}

// do runs f on the game's room goroutine and waits for it. It loads the room if
// needed, and retries once if the room shut down while f was queued.
func (m *Manager) do(ctx context.Context, gameID string, f func(r *Room)) error {
	for attempt := 0; attempt < 2; attempt++ {
		r, err := m.room(ctx, gameID)
		if err != nil {
			return err
		}
		ran := make(chan struct{})
		select {
		case r.inbox <- func() { f(r); close(ran) }:
		case <-r.done:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-ran:
			return nil
		case <-r.done:
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return errors.New("game room unavailable")
}

// room returns the loaded room for a game, loading it from the store if this
// is the first request since it went idle.
func (m *Manager) room(ctx context.Context, gameID string) (*Room, error) {
	m.mu.Lock()
	r, ok := m.rooms[gameID]
	if !ok {
		r = newRoom(m, gameID)
		m.rooms[gameID] = r
		go r.load()
	}
	m.mu.Unlock()

	select {
	case <-r.ready:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if r.loadErr != nil {
		return nil, r.loadErr
	}
	return r, nil
}

// forget removes a room that has shut down, unless it was already replaced.
func (m *Manager) forget(r *Room) {
	m.mu.Lock()
	if m.rooms[r.id] == r {
		delete(m.rooms, r.id)
	}
	m.mu.Unlock()
}

// Snapshot returns the game as viewer sees it; viewer may be nil.
func (m *Manager) Snapshot(ctx context.Context, gameID string, viewer *store.User) (View, error) {
	var v View
	err := m.do(ctx, gameID, func(r *Room) { v = r.view(viewer) })
	return v, err
}

// Join seats user in the game's open seat and returns their colour. Joining a
// game one already plays in returns that colour.
func (m *Manager) Join(ctx context.Context, gameID string, user store.User) (string, error) {
	var color string
	var joinErr error
	err := m.do(ctx, gameID, func(r *Room) { color, joinErr = r.join(user) })
	if err != nil {
		return "", err
	}
	return color, joinErr
}

// Rooms reports how many games are loaded, for the health endpoint.
func (m *Manager) Rooms() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rooms)
}

// Shutdown closes every connection, which lets each room go idle.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		rooms = append(rooms, r)
	}
	m.mu.Unlock()
	for _, r := range rooms {
		select {
		case r.inbox <- r.closeClients:
		case <-r.done:
		}
	}
}
