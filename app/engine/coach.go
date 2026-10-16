package engine

import (
	"math/bits"
	"sort"
)

// The coach judges a move the way a teacher would: how much it gave away
// compared with the best move, and why, in one plain sentence. It lives in the
// engine so every client says the same thing about the same move.

// Move quality, from best to worst.
const (
	KindGreat      = "great"
	KindGood       = "good"
	KindInaccuracy = "inaccuracy"
	KindMistake    = "mistake"
	KindBlunder    = "blunder"
)

// Assessment is the coach's verdict on one move.
type Assessment struct {
	Kind  string // one of the Kind constants
	Title string // "Great move", "Blunder", ...
	Text  string // one sentence: why
	Loss  int    // centipawns given away against the best move, 0 to 1000
	Best  string // the best move in UCI, for "show me"
}

var kindTitles = map[string]string{
	KindGreat: "Great move", KindGood: "Good move", KindInaccuracy: "Inaccuracy",
	KindMistake: "Mistake", KindBlunder: "Blunder",
}

var pieceNames = map[byte]string{'p': "pawn", 'n': "knight", 'b': "bishop", 'r': "rook", 'q': "queen", 'k': "king"}

// cp turns a search score into bounded centipawns, so a mate counts as a large
// but finite number and losses stay comparable.
func cp(score int) int {
	if isMateScore(score) {
		return 10000
	}
	if isMateScore(-score) {
		return -10000
	}
	return score
}

// scoreAfter is the value of playing m, from the mover's point of view.
func scoreAfter(gs GameState, history []uint64, m Move, depth int) int {
	child := ApplyMove(gs, m)
	r := Search(child, SearchOptions{MaxDepth: depth, History: append(history, PositionKey(gs))}, nil)
	if !r.HasBest {
		if InCheck(child) {
			return 10000 // the move mates
		}
		return 0
	}
	return -cp(r.Score)
}

// Assess judges uci, a legal move in the current position, before it is played.
func (g *Game) Assess(uci string) (Assessment, error) {
	if !g.IsLegal(uci) {
		return Assessment{}, ErrIllegalMove
	}
	gs, history := g.Position(), g.History()
	best := Search(gs, SearchOptions{MaxDepth: 3, History: history}, nil)
	move, _ := ParseUCIMove(gs, uci)
	played := scoreAfter(gs, history, move, 2)

	loss := cp(best.Score) - played
	loss = max(0, min(1000, loss))
	a := Assessment{Loss: loss}
	if best.HasBest {
		a.Best = MoveToUCI(best.Best)
	}
	// Bands are in centipawns given away. Hanging a minor piece (about 300)
	// is a blunder: that is how a player would describe it.
	switch {
	case loss <= 15 || a.Best == uci:
		a.Kind = KindGreat
	case loss <= 50:
		a.Kind = KindGood
	case loss <= 110:
		a.Kind = KindInaccuracy
	case loss <= 230:
		a.Kind = KindMistake
	default:
		a.Kind = KindBlunder
	}
	a.Title = kindTitles[a.Kind]
	a.Text = g.reason(a.Kind, uci, a.Best)
	return a, nil
}

// reason explains a verdict in one sentence.
func (g *Game) reason(kind, uci, best string) string {
	gs := g.Position()
	from, to := PositionToIndex(uci[0:2]), PositionToIndex(uci[2:4])
	piece := pieceAt(gs, from) | 0x20 // lowercase
	captured := pieceAt(gs, to) | 0x20
	move, _ := ParseUCIMove(gs, uci)
	child := ApplyMove(gs, move)

	if kind == KindGreat || kind == KindGood {
		switch {
		case InCheck(child) && len(GenerateLegalMoveList(child)) == 0:
			return "Checkmate."
		case pieceAt(gs, to) != 0:
			return "You win the " + pieceNames[captured] + ". Clean."
		case piece == 'k' && abs(to%8-from%8) == 2:
			return "King tucked away and the rook brought into play."
		case InCheck(child):
			return "Check, and it keeps the pressure on."
		case (piece == 'n' || piece == 'b') && (from/8 == 0 || from/8 == 7):
			return "Develops the " + pieceNames[piece] + " toward the centre."
		case piece == 'p' && (to%8 == 3 || to%8 == 4):
			return "Claims the centre."
		case kind == KindGreat:
			return "Exactly what the engine would play."
		}
		return "A solid, sensible move."
	}

	// A bad move: say what it allows, if it hangs something, else name the
	// better move.
	reply := Search(child, SearchOptions{MaxDepth: 2}, nil)
	if reply.HasBest {
		dst := PositionToIndex(reply.Best.Destination)
		if v := pieceAt(child, dst) | 0x20; v == 'n' || v == 'b' || v == 'r' || v == 'q' {
			return "Your " + pieceNames[v] + " on " + reply.Best.Destination + " can now be taken."
		}
	}
	if best != "" {
		return sanFor(gs, best, g.legal, false, false) + " was stronger here."
	}
	return "There was a better move."
}

// Describe says in a few words what a move does, as a commentator would of
// the opponent's move: "takes your bishop on a6", "castles", "gives check".
func (g *Game) Describe(uci string) string {
	if !g.IsLegal(uci) {
		return ""
	}
	gs := g.Position()
	from, to := PositionToIndex(uci[0:2]), PositionToIndex(uci[2:4])
	piece := pieceAt(gs, from) | 0x20
	move, _ := ParseUCIMove(gs, uci)
	child := ApplyMove(gs, move)
	var s string
	switch {
	case pieceAt(gs, to) != 0:
		s = "takes your " + pieceNames[pieceAt(gs, to)|0x20] + " on " + uci[2:4]
	case piece == 'k' && abs(to%8-from%8) == 2:
		s = "castles"
	case len(uci) == 5:
		s = "promotes a pawn"
	default:
		s = "moves the " + pieceNames[piece] + " to " + uci[2:4]
	}
	if InCheck(child) {
		if len(GenerateLegalMoveList(child)) == 0 {
			return s + ". Checkmate"
		}
		return s + ", with check"
	}
	return s
}

// Explain says in a few words why uci is a good idea, for "show me".
func (g *Game) Explain(uci string) string {
	if !g.IsLegal(uci) {
		return ""
	}
	gs := g.Position()
	to := PositionToIndex(uci[2:4])
	move, _ := ParseUCIMove(gs, uci)
	child := ApplyMove(gs, move)
	from := PositionToIndex(uci[0:2])
	switch {
	case InCheck(child) && len(GenerateLegalMoveList(child)) == 0:
		return "it is checkmate."
	case pieceAt(gs, to) != 0:
		return "it wins the " + pieceNames[pieceAt(gs, to)|0x20] + "."
	case pieceAt(gs, from)|0x20 == 'k' && abs(to%8-from%8) == 2:
		return "castle to keep your king safe."
	case InCheck(child):
		return "it gives check."
	}
	return "it improves your position the most."
}

// Candidates returns up to n moves the side to move would consider, best
// first, scored with a shallow search. It is what the app draws while the
// engine "thinks".
func (g *Game) Candidates(n int) []string {
	gs, history := g.Position(), g.History()
	type scored struct {
		uci   string
		score int
	}
	list := make([]scored, 0, len(g.legal))
	for _, uci := range g.legal {
		m, err := ParseUCIMove(gs, uci)
		if err != nil {
			continue
		}
		list = append(list, scored{uci, scoreAfter(gs, history, m, 1)})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	out := make([]string, 0, n)
	for i := 0; i < len(list) && i < n; i++ {
		out = append(out, list[i].uci)
	}
	return out
}

// Eval returns the position's score in centipawns from White's point of view,
// with mates as ±10000.
func (g *Game) Eval(depth int) int {
	gs := g.Position()
	r := Search(gs, SearchOptions{MaxDepth: depth, History: g.History()}, nil)
	score := cp(r.Score)
	if !r.HasBest {
		score = 0
		if InCheck(gs) {
			score = -10000
		}
	}
	if gs.Turn == "b" {
		score = -score
	}
	return score
}

// Loose returns the squares of color's pieces (not pawns or the king) that the
// opponent attacks and nothing defends.
func (g *Game) Loose(color string) []string {
	gs := g.Position()
	white := color == "w"
	own := gs.BlackBitboard
	if white {
		own = gs.WhiteBitboard
	}
	candidates := own &^ (gs.PawnBitboard | gs.KingBitboard)
	var out []string
	for candidates != 0 {
		i := bits.TrailingZeros64(candidates)
		candidates &= candidates - 1
		bit := uint64(1) << uint(i)
		if isSquareAttacked(gs, bit, !white) && !defended(gs, i, white) {
			out = append(out, squareName(i))
		}
	}
	return out
}

// defended reports whether color has a piece guarding square i. The attack
// test looks outward from the square, so the piece standing on it does not
// count as its own defender.
func defended(gs GameState, i int, white bool) bool {
	return isSquareAttacked(gs, uint64(1)<<uint(i), white)
}

// Watchers returns the pieces bearing on the piece that stands on square:
// guards are its own side's pieces defending it, attackers the opponent's.
// Both are square names. An empty or unknown square has neither.
func (g *Game) Watchers(square string) (guards, attackers []string) {
	if len(square) != 2 || square[0] < 'a' || square[0] > 'h' || square[1] < '1' || square[1] > '8' {
		return nil, nil
	}
	i := int(square[0]-'a') + 8*int(square[1]-'1')
	gs := g.Position()
	bit := uint64(1) << uint(i)
	occupied := gs.WhiteBitboard | gs.BlackBitboard
	if occupied&bit == 0 {
		return nil, nil
	}
	own := gs.BlackBitboard
	if gs.WhiteBitboard&bit != 0 {
		own = gs.WhiteBitboard
	}
	for att := attackersTo(gs, bit, occupied) &^ bit; att != 0; att &= att - 1 {
		j := bits.TrailingZeros64(att)
		if own&(uint64(1)<<uint(j)) != 0 {
			guards = append(guards, squareName(j))
		} else {
			attackers = append(attackers, squareName(j))
		}
	}
	return guards, attackers
}
