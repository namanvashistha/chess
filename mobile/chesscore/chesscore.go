// Package chesscore is the engine's API for the iOS app, shaped for gomobile:
// it exposes only strings, ints and bools, so `gomobile bind` can generate a
// plain Objective-C/Swift interface without wrapping dao or dto types.
//
// Squares are algebraic ("e4") and moves are UCI ("e2e4", "e7e8q").
package chesscore

import (
	"strings"
	"sync"
	"time"

	"chess-engine/app/domain/dao"
	"chess-engine/app/engine"
)

// Game status values returned by Game.Status.
const (
	StatusPlaying    = ""
	StatusCheckmate  = "checkmate"
	StatusStalemate  = "stalemate"
	StatusRepetition = engine.DrawByRepetition
	StatusFiftyMove  = engine.DrawByFiftyMove
)

// Game is one game in progress: the current position plus enough history to
// undo moves and to detect repetition.
type Game struct {
	mu     sync.Mutex
	states []dao.GameState // states[0] is the start; the last is the current position
	moves  []string        // UCI moves; len(moves) == len(states)-1
	keys   []uint64        // Zobrist key of each state, aligned with states
	table  *engine.TranspositionTable
	stop   chan struct{} // closed by StopSearch; replaced by each Search
}

// NewGame returns a game at the standard starting position.
func NewGame() *Game {
	return newGame(engine.StartState())
}

// NewGameFromFEN returns a game starting at the given FEN position.
func NewGameFromFEN(fen string) (*Game, error) {
	gs, err := engine.ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	return newGame(gs), nil
}

func newGame(gs dao.GameState) *Game {
	return &Game{
		states: []dao.GameState{gs},
		keys:   []uint64{engine.PositionKey(gs)},
		table:  engine.NewTranspositionTable(16),
	}
}

func (g *Game) current() dao.GameState { return g.states[len(g.states)-1] }

// FEN returns the current position as FEN.
func (g *Game) FEN() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return engine.ToFEN(g.current())
}

// Board returns the current position as 64 characters, a8 to h1 in reading
// order: FEN piece letters for pieces and '.' for empty squares.
func (g *Game) Board() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	placement := strings.Fields(engine.ToFEN(g.current()))[0]
	var sb strings.Builder
	sb.Grow(64)
	for _, r := range placement {
		switch {
		case r == '/':
		case r >= '1' && r <= '8':
			sb.WriteString(strings.Repeat(".", int(r-'0')))
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Turn returns "w" or "b" for the side to move.
func (g *Game) Turn() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.current().Turn
}

// InCheck reports whether the side to move is in check.
func (g *Game) InCheck() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return engine.InCheck(g.current())
}

// LegalMoves returns the legal moves for the side to move as space-separated
// UCI strings. Promotions appear once per piece choice ("e7e8q e7e8r ...").
func (g *Game) LegalMoves() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(legalUCI(g.current()), " ")
}

func legalUCI(gs dao.GameState) []string {
	list := engine.GenerateLegalMoveList(gs)
	out := make([]string, 0, len(list))
	for _, m := range list {
		out = append(out, bitSquare(m.Src)+bitSquare(m.Dst)+m.Promotion)
	}
	return out
}

func bitSquare(bit uint64) string {
	i := 0
	for bit > 1 {
		bit >>= 1
		i++
	}
	return string([]byte{byte('a' + i%8), byte('1' + i/8)})
}

// Play makes a move given in UCI. It returns an error if the move is not legal
// in the current position.
func (g *Game) Play(uci string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	gs := g.current()
	uci = strings.ToLower(strings.TrimSpace(uci))
	if !contains(legalUCI(gs), uci) {
		return &MoveError{Move: uci}
	}
	move, err := engine.ParseUCIMove(gs, uci)
	if err != nil {
		return err
	}
	next := engine.ApplyMove(gs, move)
	g.states = append(g.states, next)
	g.moves = append(g.moves, uci)
	g.keys = append(g.keys, engine.PositionKey(next))
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// MoveError reports a move that is not legal in the current position.
type MoveError struct {
	Move string
}

func (e *MoveError) Error() string { return "illegal move " + e.Move }

// Undo takes back the last move. It returns false when there is nothing to undo.
func (g *Game) Undo() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.moves) == 0 {
		return false
	}
	n := len(g.states) - 1
	g.states = g.states[:n]
	g.keys = g.keys[:n]
	g.moves = g.moves[:n-1]
	return true
}

// MoveCount returns the number of half-moves played.
func (g *Game) MoveCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.moves)
}

// Moves returns the moves played so far as space-separated UCI strings.
func (g *Game) Moves() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.moves, " ")
}

// Status reports whether the game is over: one of the Status constants.
func (g *Game) Status() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	gs := g.current()
	if len(engine.GenerateLegalMoveList(gs)) == 0 {
		if engine.InCheck(gs) {
			return StatusCheckmate
		}
		return StatusStalemate
	}
	return engine.DrawStatus(gs, g.keys)
}

// SearchResult is the engine's answer for one position.
type SearchResult struct {
	BestMove string // UCI; empty when the side to move has no legal moves
	Score    int    // centipawns from White's point of view
	Mate     int    // moves to mate from White's point of view (+ White mates); 0 if none
	Depth    int    // deepest completed iteration
	Nodes    int
	PV       string // principal variation, space-separated UCI
}

// Search finds the best move for the side to move. It stops at maxDepth, after
// moveTimeMs milliseconds, or when StopSearch is called, whichever is first;
// pass 0 to leave either limit off. It blocks, so call it off the main thread.
func (g *Game) Search(maxDepth, moveTimeMs int) *SearchResult {
	g.mu.Lock()
	gs := g.current()
	opts := engine.SearchOptions{
		MaxDepth: maxDepth,
		MoveTime: time.Duration(moveTimeMs) * time.Millisecond,
		History:  engine.SearchHistory(g.keys),
		Table:    g.table,
	}
	if maxDepth <= 0 && moveTimeMs <= 0 {
		opts.MaxDepth = 6
	}
	stop := make(chan struct{})
	g.stop = stop
	opts.Stop = stop
	g.mu.Unlock()

	r := engine.Search(gs, opts, nil)

	sign := 1
	if gs.Turn == "b" {
		sign = -1
	}
	out := &SearchResult{Score: sign * r.Score, Mate: sign * r.Mate, Depth: r.Depth, Nodes: r.Nodes}
	if r.HasBest {
		out.BestMove = engine.MoveToUCI(r.Best)
	}
	pv := make([]string, 0, len(r.PV))
	for _, m := range r.PV {
		pv = append(pv, engine.MoveToUCI(m))
	}
	out.PV = strings.Join(pv, " ")
	return out
}

// StopSearch ends a running Search early; it returns the best move found so far.
func (g *Game) StopSearch() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.stop != nil {
		close(g.stop)
		g.stop = nil
	}
}
