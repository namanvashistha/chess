package engine

import (
	"errors"
	"math/bits"
	"strings"
)

// Game results, in PGN notation.
const (
	ResultNone  = ""
	ResultWhite = "1-0"
	ResultBlack = "0-1"
	ResultDraw  = "1/2-1/2"
)

// Reasons a game ended.
const (
	ReasonCheckmate    = "checkmate"
	ReasonStalemate    = "stalemate"
	ReasonRepetition   = "repetition"
	ReasonFiftyMove    = "fifty_move"
	ReasonInsufficient = "insufficient_material"
)

// ErrIllegalMove is returned by Game.Play for a move that is not legal in the
// current position.
var ErrIllegalMove = errors.New("illegal move")

// Game is a game in progress: the start position, the moves played, and what
// follows from them. It is the one place the rules are applied, shared by the
// server, the UCI binary and the mobile bindings. It is not safe for
// concurrent use.
type Game struct {
	states []GameState // states[0] is the start; the last is the current position
	keys   []uint64    // Zobrist key of each state
	moves  []string    // UCI
	sans   []string
	legal  []string // legal UCI moves in the current position
}

// NewGame returns a game at the standard starting position.
func NewGame() *Game {
	return newGame(StartState())
}

// NewGameFromFEN returns a game starting at the given position.
func NewGameFromFEN(fen string) (*Game, error) {
	gs, err := ParseFEN(fen)
	if err != nil {
		return nil, err
	}
	return newGame(gs), nil
}

func newGame(gs GameState) *Game {
	g := &Game{states: []GameState{gs}, keys: []uint64{PositionKey(gs)}}
	g.legal = legalUCI(gs)
	return g
}

// ReplayGame builds a game from the standard start and a list of UCI moves.
func ReplayGame(moves []string) (*Game, error) {
	g := NewGame()
	for _, m := range moves {
		if _, err := g.Play(m); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// Position returns the current position.
func (g *Game) Position() GameState { return g.states[len(g.states)-1] }

// Turn returns "w" or "b" for the side to move.
func (g *Game) Turn() string { return g.Position().Turn }

// Ply returns the number of half-moves played.
func (g *Game) Ply() int { return len(g.moves) }

// Moves returns the moves played, in UCI.
func (g *Game) Moves() []string { return append([]string(nil), g.moves...) }

// SANs returns the moves played, in standard algebraic notation.
func (g *Game) SANs() []string { return append([]string(nil), g.sans...) }

// Legal returns the legal moves in the current position, in UCI. A promotion
// appears once per piece choice.
func (g *Game) Legal() []string { return append([]string(nil), g.legal...) }

// IsLegal reports whether uci is a legal move in the current position.
func (g *Game) IsLegal(uci string) bool {
	for _, m := range g.legal {
		if m == uci {
			return true
		}
	}
	return false
}

// InCheck reports whether the side to move is in check.
func (g *Game) InCheck() bool { return InCheck(g.Position()) }

// FEN returns the current position as FEN.
func (g *Game) FEN() string { return ToFEN(g.Position()) }

// History returns the Zobrist keys of the positions before the current one,
// oldest first, for SearchOptions.History.
func (g *Game) History() []uint64 { return SearchHistory(g.keys) }

// Play makes a move given in UCI and returns it in SAN. It returns
// ErrIllegalMove, and changes nothing, if the move is not legal or the game is
// already over.
func (g *Game) Play(uci string) (string, error) {
	uci = strings.ToLower(strings.TrimSpace(uci))
	if result, _ := g.Outcome(); result != ResultNone || !g.IsLegal(uci) {
		return "", ErrIllegalMove
	}
	gs := g.Position()
	move, err := ParseUCIMove(gs, uci)
	if err != nil {
		return "", ErrIllegalMove
	}
	before := g.legal
	next := ApplyMove(gs, move)

	g.states = append(g.states, next)
	g.keys = append(g.keys, PositionKey(next))
	g.moves = append(g.moves, uci)
	g.legal = legalUCI(next)
	san := sanFor(gs, uci, before, InCheck(next), len(g.legal) == 0)
	g.sans = append(g.sans, san)
	return san, nil
}

// Undo takes back the last move. It returns false when there is none.
func (g *Game) Undo() bool {
	if len(g.moves) == 0 {
		return false
	}
	n := len(g.states) - 1
	g.states = g.states[:n]
	g.keys = g.keys[:n]
	g.moves = g.moves[:n-1]
	g.sans = g.sans[:n-1]
	g.legal = legalUCI(g.Position())
	return true
}

// Outcome reports how the game ended, or ResultNone while it is still going.
// These are the endings the rules force; resignation and agreed draws are the
// caller's to record.
func (g *Game) Outcome() (result, reason string) {
	gs := g.Position()
	if len(g.legal) == 0 {
		if InCheck(gs) {
			if gs.Turn == "w" {
				return ResultBlack, ReasonCheckmate
			}
			return ResultWhite, ReasonCheckmate
		}
		return ResultDraw, ReasonStalemate
	}
	if draw := DrawStatus(gs, g.keys); draw != "" {
		return ResultDraw, draw
	}
	if insufficientMaterial(gs) {
		return ResultDraw, ReasonInsufficient
	}
	return ResultNone, ""
}

// Board returns the current position as 64 characters, a8 to h1 in reading
// order: FEN piece letters for pieces and '.' for empty squares.
func (g *Game) Board() string {
	gs := g.Position()
	var sb strings.Builder
	sb.Grow(64)
	for rank := 7; rank >= 0; rank-- {
		for file := 0; file < 8; file++ {
			if p := pieceAt(gs, rank*8+file); p != 0 {
				sb.WriteByte(p)
			} else {
				sb.WriteByte('.')
			}
		}
	}
	return sb.String()
}

func legalUCI(gs GameState) []string {
	list := GenerateLegalMoveList(gs)
	out := make([]string, 0, len(list))
	for _, m := range list {
		out = append(out, squareName(bits.TrailingZeros64(m.Src))+squareName(bits.TrailingZeros64(m.Dst))+m.Promotion)
	}
	return out
}

func squareName(i int) string {
	return string([]byte{byte('a' + i%8), byte('1' + i/8)})
}

// pieceAt returns the FEN letter of the piece on square index i (a1 = 0), or 0.
func pieceAt(gs GameState, i int) byte {
	bit := uint64(1) << uint(i)
	if (gs.WhiteBitboard|gs.BlackBitboard)&bit == 0 {
		return 0
	}
	return getPieceCode(bit, gs.WhiteBitboard&bit != 0, gs)[0]
}

// sanFor renders uci in standard algebraic notation, given the position before
// the move and its legal moves.
func sanFor(gs GameState, uci string, legal []string, check, noMoves bool) string {
	from, to := PositionToIndex(uci[0:2]), PositionToIndex(uci[2:4])
	piece := pieceAt(gs, from)
	kind := piece &^ 0x20 // uppercase
	var s string

	switch {
	case kind == 'K' && abs(to%8-from%8) == 2:
		s = "O-O"
		if to%8 == 2 {
			s = "O-O-O"
		}
	case kind == 'P':
		if from%8 != to%8 {
			s = uci[0:1] + "x"
		}
		s += uci[2:4]
		if len(uci) == 5 {
			s += "=" + strings.ToUpper(uci[4:5])
		}
	default:
		// Another piece of the same kind that can reach the same square forces
		// a file, a rank, or both to tell them apart.
		var sameFile, sameRank, rivals bool
		for _, m := range legal {
			f := PositionToIndex(m[0:2])
			if f == from || m[2:4] != uci[2:4] || pieceAt(gs, f) != piece {
				continue
			}
			rivals = true
			sameFile = sameFile || f%8 == from%8
			sameRank = sameRank || f/8 == from/8
		}
		s = string(kind)
		switch {
		case !rivals:
		case !sameFile:
			s += uci[0:1]
		case !sameRank:
			s += uci[1:2]
		default:
			s += uci[0:2]
		}
		if pieceAt(gs, to) != 0 {
			s += "x"
		}
		s += uci[2:4]
	}
	if check && noMoves {
		return s + "#"
	}
	if check {
		return s + "+"
	}
	return s
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// lightSquares has a bit set for every light square (a1 is dark).
const lightSquares uint64 = 0x55AA55AA55AA55AA

// insufficientMaterial reports positions where neither side can ever mate:
// bare kings, a single minor piece, or only bishops that all stand on squares
// of one colour.
func insufficientMaterial(gs GameState) bool {
	if gs.PawnBitboard|gs.RookBitboard|gs.QueenBitboard != 0 {
		return false
	}
	minors := gs.KnightBitboard | gs.BishopBitboard
	if bits.OnesCount64(minors) <= 1 {
		return true
	}
	if gs.KnightBitboard != 0 {
		return false
	}
	return gs.BishopBitboard&lightSquares == 0 || gs.BishopBitboard&^lightSquares == 0
}
