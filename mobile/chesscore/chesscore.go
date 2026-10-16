// Package chesscore is the engine's API for the iOS app, shaped for gomobile:
// it exposes only strings, ints and bools, so `gomobile bind` can generate a
// plain Objective-C/Swift interface. The rules live in engine.Game; this adds
// locking, because Swift calls in from more than one thread.
//
// Squares are algebraic ("e4") and moves are UCI ("e2e4", "e7e8q").
package chesscore

import (
	"strings"
	"sync"
	"time"

	"chess-engine/app/engine"
)

// Game status values returned by Game.Status: "" while the game is going,
// otherwise the reason it ended.
const (
	StatusPlaying      = ""
	StatusCheckmate    = engine.ReasonCheckmate
	StatusStalemate    = engine.ReasonStalemate
	StatusRepetition   = engine.ReasonRepetition
	StatusFiftyMove    = engine.ReasonFiftyMove
	StatusInsufficient = engine.ReasonInsufficient
)

// Game is one game in progress.
type Game struct {
	mu    sync.Mutex
	g     *engine.Game
	table *engine.TranspositionTable
	stop  chan struct{} // closed by StopSearch; replaced by each Search
}

// NewGame returns a game at the standard starting position.
func NewGame() *Game {
	return wrap(engine.NewGame())
}

// NewGameFromFEN returns a game starting at the given FEN position.
func NewGameFromFEN(fen string) (*Game, error) {
	g, err := engine.NewGameFromFEN(fen)
	if err != nil {
		return nil, err
	}
	return wrap(g), nil
}

func wrap(g *engine.Game) *Game {
	return &Game{g: g, table: engine.NewTranspositionTable(16)}
}

// FEN returns the current position as FEN.
func (g *Game) FEN() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.FEN()
}

// Board returns the current position as 64 characters, a8 to h1 in reading
// order: FEN piece letters for pieces and '.' for empty squares.
func (g *Game) Board() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Board()
}

// Turn returns "w" or "b" for the side to move.
func (g *Game) Turn() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Turn()
}

// InCheck reports whether the side to move is in check.
func (g *Game) InCheck() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.InCheck()
}

// LegalMoves returns the legal moves for the side to move as space-separated
// UCI strings. Promotions appear once per piece choice ("e7e8q e7e8r ...").
func (g *Game) LegalMoves() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.g.Legal(), " ")
}

// Play makes a move given in UCI and returns it in SAN. It returns an error if
// the move is not legal in the current position.
func (g *Game) Play(uci string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Play(uci)
}

// Undo takes back the last move. It returns false when there is nothing to undo.
func (g *Game) Undo() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Undo()
}

// MoveCount returns the number of half-moves played.
func (g *Game) MoveCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Ply()
}

// Moves returns the moves played so far as space-separated UCI strings.
func (g *Game) Moves() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.g.Moves(), " ")
}

// SANs returns the moves played so far as space-separated SAN strings.
func (g *Game) SANs() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.g.SANs(), " ")
}

// Status reports why the game ended, or StatusPlaying while it goes on.
func (g *Game) Status() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, reason := g.g.Outcome()
	return reason
}

// Result reports the result in PGN notation ("1-0", "0-1", "1/2-1/2"), or ""
// while the game goes on.
func (g *Game) Result() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	result, _ := g.g.Outcome()
	return result
}

// BotMove picks a move at a difficulty level ("easy", "medium" or "hard") and
// returns it in UCI without playing it, or "" when the game is over. It blocks,
// so call it off the main thread.
func (g *Game) BotMove(level string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	uci, _ := engine.BotMove(g.g, level, g.table)
	return uci
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
	gs := g.g.Position()
	opts := engine.SearchOptions{
		MaxDepth: maxDepth,
		MoveTime: time.Duration(moveTimeMs) * time.Millisecond,
		History:  g.g.History(),
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

// Assessment is the coach's verdict on a move; see engine.Assessment.
type Assessment struct {
	Kind  string // "great", "good", "inaccuracy", "mistake" or "blunder"
	Title string
	Text  string
	Loss  int
	Best  string // UCI
}

// Assess judges a legal move before it is played. It blocks for a moment, so
// call it off the main thread.
func (g *Game) Assess(uci string) (*Assessment, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a, err := g.g.Assess(uci)
	if err != nil {
		return nil, err
	}
	return &Assessment{Kind: a.Kind, Title: a.Title, Text: a.Text, Loss: a.Loss, Best: a.Best}, nil
}

// Explain says in a few words why a move is good, for "show me".
func (g *Game) Explain(uci string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Explain(uci)
}

// SAN returns a legal move in standard algebraic notation without playing it.
func (g *Game) SAN(uci string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.g.IsLegal(uci) {
		return ""
	}
	san, _ := g.g.Play(uci)
	g.g.Undo()
	return san
}

// Candidates returns up to n moves the side to move would consider, best
// first, space-separated UCI.
func (g *Game) Candidates(n int) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.g.Candidates(n), " ")
}

// Eval returns the score in centipawns from White's point of view; mates are
// ±10000.
func (g *Game) Eval(depth int) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.g.Eval(depth)
}

// Loose returns the squares of color's undefended, attacked pieces,
// space-separated.
func (g *Game) Loose(color string) string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.g.Loose(color), " ")
}
