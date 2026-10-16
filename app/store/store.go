// Package store keeps users, games and moves. A game is its seats plus its
// list of moves; positions are never stored, only derived by replaying moves.
//
// Postgres is the real implementation. Memory implements the same interface
// for tests and has the same semantics, including the conflict errors.
package store

import (
	"context"
	"errors"
	"time"
)

// Errors a Store returns. Callers compare with errors.Is.
var (
	ErrNotFound  = errors.New("not found")
	ErrSeatTaken = errors.New("seat already taken")
	// ErrConflict means a move for that ply already exists or the game has
	// ended: someone else got there first.
	ErrConflict = errors.New("conflict")
)

// Game modes.
const (
	ModeOnline = "online" // two people on two devices
	ModeBot    = "bot"    // a person against the engine
	ModeLocal  = "local"  // two people sharing one device
)

// Game statuses, derived from seats and result.
const (
	StatusWaiting  = "waiting"  // a seat is open
	StatusActive   = "active"   // both seats taken, no result yet
	StatusFinished = "finished" // there is a result
)

// WaitingTTL is how long a game with an open seat stays in the lobby.
const WaitingTTL = 10 * time.Minute

// User is a player. Users are anonymous: a name and a session token.
type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Bot       bool      `json:"bot"`
	CreatedAt time.Time `json:"-"`
}

// Move is one recorded half-move.
type Move struct {
	UCI string `json:"uci"`
	SAN string `json:"san"`
}

// Game is a game's record: who plays, how it ended, and its moves.
type Game struct {
	ID        string
	Mode      string
	BotLevel  string
	White     *User // nil while the seat is open
	Black     *User
	Listed    bool
	Result    string // "" while the game is going
	Reason    string
	Moves     []Move
	CreatedAt time.Time
	EndedAt   *time.Time
}

// Status derives the game's status from its seats and result.
func (g Game) Status() string {
	switch {
	case g.Result != "":
		return StatusFinished
	case g.White == nil || g.Black == nil:
		return StatusWaiting
	default:
		return StatusActive
	}
}

// NewGame describes a game to create.
type NewGame struct {
	ID       string
	Mode     string
	BotLevel string
	WhiteID  int64 // 0 for an open seat
	BlackID  int64
	Listed   bool
}

// Summary is a game as the lobby lists it, without moves.
type Summary struct {
	ID        string    `json:"id"`
	Mode      string    `json:"mode"`
	White     *User     `json:"white"`
	Black     *User     `json:"black"`
	Status    string    `json:"status"`
	Result    string    `json:"result"`
	Plies     int       `json:"plies"`
	CreatedAt time.Time `json:"created_at"`
}

// Lobby is what the lobby page shows.
type Lobby struct {
	Open []Summary `json:"open"` // listed online games with an open seat
	Live []Summary `json:"live"` // listed games in progress, to watch
	Mine []Summary `json:"mine"` // the viewer's recent games
}

// Store persists users and games.
type Store interface {
	CreateUser(ctx context.Context, name string, tokenHash []byte) (User, error)
	UserByTokenHash(ctx context.Context, tokenHash []byte) (User, error)
	RenameUser(ctx context.Context, id int64, name string) (User, error)
	// DeleteUser removes a user's identity: the name becomes "Deleted player"
	// and the session no longer signs in. Their games stay intact.
	DeleteUser(ctx context.Context, id int64) error
	// BotUser returns the engine's user, creating it the first time.
	BotUser(ctx context.Context) (User, error)

	CreateGame(ctx context.Context, g NewGame) error
	Game(ctx context.Context, id string) (Game, error)
	// TakeSeat seats a user as "w" or "b" if that seat is open.
	TakeSeat(ctx context.Context, gameID, color string, userID int64) error
	// AppendMove records the move at ply, which must be the next one.
	AppendMove(ctx context.Context, gameID string, ply int, m Move) error
	// Finish records a game's result. It fails with ErrConflict if the game has
	// already ended.
	Finish(ctx context.Context, gameID, result, reason string) error
	// Lobby lists games for the lobby; userID 0 leaves Mine empty.
	Lobby(ctx context.Context, userID int64, limit int) (Lobby, error)

	Ping(ctx context.Context) error
	Close()
}

// DeletedName replaces a deleted user's name.
const DeletedName = "Deleted player"

// BotName is the engine's display name.
const BotName = "Engine"
