// Package live runs games in progress. Each game that someone is looking at
// has a room: one goroutine that owns the game's state, applies moves in
// order, writes them to the store and sends them to everyone connected. There
// are no locks around game state and no cache to keep in step with the
// database; the room is the only writer.
//
// The WebSocket protocol is JSON objects with a "t" (type) field.
//
// Client to server:
//
//	{"t":"move","ply":4,"uci":"g1f3"}  play a move; ply is the number of moves
//	                                   already played, so a resent or stale
//	                                   move is rejected instead of applied twice
//
// Server to client:
//
//	snapshot  the whole game, as this client sees it; sent on connect, when
//	          someone takes a seat, and after an error that may have left the
//	          client out of step
//	move      one move and the position after it
//	error     sent only to the client that caused it
package live

import "chess-engine/app/store"

// Message types.
const (
	typeMove     = "move"
	typeSnapshot = "snapshot"
	typeError    = "error"
)

// Error codes in error messages.
const (
	CodeBadMessage  = "bad_message"
	CodeNotYourTurn = "not_your_turn"
	CodeStale       = "stale"
	CodeIllegal     = "illegal"
	CodeGameOver    = "game_over"
	CodeSaveFailed  = "save_failed"
)

// inbound is any message a client sends.
type inbound struct {
	T   string `json:"t"`
	Ply int    `json:"ply"`
	UCI string `json:"uci"`
}

// View is a game as one viewer sees it. You is the viewer's colour: "w", "b",
// "both" in a game they play alone on one device, or "" for a spectator.
type View struct {
	ID       string       `json:"id"`
	Mode     string       `json:"mode"`
	BotLevel string       `json:"bot_level,omitempty"`
	White    *store.User  `json:"white"`
	Black    *store.User  `json:"black"`
	Status   string       `json:"status"`
	Result   string       `json:"result"`
	Reason   string       `json:"reason"`
	Moves    []store.Move `json:"moves"`
	FEN      string       `json:"fen"`
	Turn     string       `json:"turn"`
	Check    bool         `json:"check"`
	Legal    []string     `json:"legal"`
	You      string       `json:"you"`
}

type snapshotMsg struct {
	T    string `json:"t"`
	Game View   `json:"game"`
}

type moveMsg struct {
	T      string   `json:"t"`
	Ply    int      `json:"ply"` // index of this move; the game now has ply+1 moves
	UCI    string   `json:"uci"`
	SAN    string   `json:"san"`
	FEN    string   `json:"fen"`
	Turn   string   `json:"turn"`
	Check  bool     `json:"check"`
	Legal  []string `json:"legal"`
	Status string   `json:"status"`
	Result string   `json:"result"`
	Reason string   `json:"reason"`
}

type errorMsg struct {
	T       string `json:"t"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
