package store

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Memory is an in-process Store for tests. It enforces the same rules as
// Postgres: one move per ply, open seats only, one result per game.
type Memory struct {
	mu     sync.Mutex
	nextID int64
	users  map[int64]*memUser
	games  map[string]*Game
	order  []string // game ids in creation order
}

type memUser struct {
	User
	tokenHash string
}

// NewMemory returns an empty Memory store.
func NewMemory() *Memory {
	return &Memory{users: map[int64]*memUser{}, games: map[string]*Game{}}
}

func (m *Memory) Ping(context.Context) error { return nil }
func (m *Memory) Close()                     {}

func (m *Memory) CreateUser(_ context.Context, name string, tokenHash []byte) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	u := &memUser{User: User{ID: m.nextID, Name: name, CreatedAt: time.Now()}, tokenHash: string(tokenHash)}
	m.users[u.ID] = u
	return u.User, nil
}

func (m *Memory) UserByTokenHash(_ context.Context, tokenHash []byte) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.tokenHash != "" && u.tokenHash == string(tokenHash) {
			return u.User, nil
		}
	}
	return User{}, ErrNotFound
}

func (m *Memory) RenameUser(_ context.Context, id int64, name string) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok || u.Bot {
		return User{}, ErrNotFound
	}
	u.Name = name
	return u.User, nil
}

func (m *Memory) DeleteUser(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok || u.Bot {
		return ErrNotFound
	}
	u.Name, u.tokenHash = DeletedName, ""
	return nil
}

func (m *Memory) BotUser(context.Context) (User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Bot {
			return u.User, nil
		}
	}
	m.nextID++
	u := &memUser{User: User{ID: m.nextID, Name: BotName, Bot: true, CreatedAt: time.Now()}}
	m.users[u.ID] = u
	return u.User, nil
}

func (m *Memory) CreateGame(_ context.Context, g NewGame) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, dup := m.games[g.ID]; dup {
		return fmt.Errorf("game %s already exists", g.ID)
	}
	m.games[g.ID] = &Game{ID: g.ID, Mode: g.Mode, BotLevel: g.BotLevel, Listed: g.Listed,
		White: m.userRef(g.WhiteID), Black: m.userRef(g.BlackID), CreatedAt: time.Now()}
	m.order = append(m.order, g.ID)
	return nil
}

// userRef returns the seat value for a user id; 0 is an open seat.
func (m *Memory) userRef(id int64) *User {
	if id == 0 {
		return nil
	}
	return &User{ID: id}
}

// resolve fills a stored game's seats with the users' current names.
func (m *Memory) resolve(g Game) Game {
	for _, seat := range []**User{&g.White, &g.Black} {
		if *seat != nil {
			u := m.users[(*seat).ID].User
			*seat = &u
		}
	}
	g.Moves = append([]Move(nil), g.Moves...)
	return g
}

func (m *Memory) Game(_ context.Context, id string) (Game, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.games[id]
	if !ok {
		return Game{}, ErrNotFound
	}
	return m.resolve(*g), nil
}

func (m *Memory) TakeSeat(_ context.Context, gameID, color string, userID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.games[gameID]
	if !ok {
		return ErrNotFound
	}
	seat := map[string]**User{"w": &g.White, "b": &g.Black}[color]
	if seat == nil {
		return fmt.Errorf("bad color %q", color)
	}
	if *seat != nil || g.Result != "" {
		return ErrSeatTaken
	}
	*seat = m.userRef(userID)
	return nil
}

func (m *Memory) AppendMove(_ context.Context, gameID string, ply int, mv Move) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.games[gameID]
	if !ok || g.Result != "" || len(g.Moves) != ply {
		return ErrConflict
	}
	g.Moves = append(g.Moves, mv)
	return nil
}

func (m *Memory) Finish(_ context.Context, gameID, result, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	g, ok := m.games[gameID]
	if !ok || g.Result != "" {
		return ErrConflict
	}
	now := time.Now()
	g.Result, g.Reason, g.EndedAt = result, reason, &now
	return nil
}

func (m *Memory) Lobby(_ context.Context, userID int64, limit int) (Lobby, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	l := Lobby{Open: []Summary{}, Live: []Summary{}, Mine: []Summary{}}
	for k := len(m.order) - 1; k >= 0; k-- { // newest first
		g := m.resolve(*m.games[m.order[k]])
		mine := (g.White != nil && g.White.ID == userID) || (g.Black != nil && g.Black.ID == userID)
		s := summarize(g, len(g.Moves))
		switch {
		case userID != 0 && mine:
			if len(l.Mine) < limit {
				l.Mine = append(l.Mine, s)
			}
		case !g.Listed:
		case g.Status() == StatusWaiting && g.Mode == ModeOnline && time.Since(g.CreatedAt) < WaitingTTL:
			if len(l.Open) < limit {
				l.Open = append(l.Open, s)
			}
		case g.Status() == StatusActive:
			if len(l.Live) < limit {
				l.Live = append(l.Live, s)
			}
		}
	}
	return l, nil
}
