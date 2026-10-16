package engine

import (
	"math/bits"
	"sort"
)

// Centipawn piece values keyed by the lowercase FEN letter (used by the greedy
// chooser); the search uses the same scale via pieceValueAt/evalMaterial.
var botPieceValue = map[byte]int{
	'p': 100, 'n': 320, 'b': 330, 'r': 500, 'q': 900, 'k': 0,
}

const (
	searchInf = 1 << 30
	mateScore = 1 << 20
)

var defFiles = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
var defRanks = []string{"1", "2", "3", "4", "5", "6", "7", "8"}

// botMove is a search-level move. promo carries the promotion piece so the
// search can consider underpromotion and so its material evaluation sees the
// piece that actually appears on the board.
type botMove struct {
	src, dst uint64
	promo    string
}

// applyBotMove plays a search move. It goes through the same applier as the
// server, so the search now sees castling, en passant and promotion instead of
// silently playing a different game.
func applyBotMove(gs GameState, m botMove) GameState {
	ns := applyBitboardMove(gs, m.src, m.dst, m.promo)
	ns.Turn = oppTurn(gs.Turn)
	return ns
}

// sideToMoveMoves lists the legal moves for the side to move, with promotions
// expanded into all four pieces.
//
// The order is deterministic without sorting: appendLegalMoves walks the
// bitboards least-significant-bit first in a fixed piece order, unlike the
// map-based generator whose iteration order Go randomises.
func sideToMoveMoves(gs GameState) []botMove {
	return appendLegalMoves(gs, make([]botMove, 0, 48))
}

// sideToMoveMovesOrdered orders captures first, most valuable victim captured by
// least valuable attacker (MVV-LVA), so alpha-beta prunes more.
//
// SliceStable, not Slice: an unstable sort over equally-scored moves reorders
// them unpredictably, which reintroduces the non-determinism that the ordered
// generator exists to remove.
func sideToMoveMovesOrdered(gs GameState) []botMove {
	moves := sideToMoveMoves(gs)
	sort.SliceStable(moves, func(i, j int) bool {
		return captureScore(gs, moves[i]) > captureScore(gs, moves[j])
	})
	return moves
}

// captureScore ranks a move for ordering. Winning captures first (queen taken by
// a pawn beats queen taken by a queen), then promotions, then quiet moves.
func captureScore(gs GameState, m botMove) int {
	score := 0
	if victim := pieceValueAt(gs, m.dst); victim != 0 {
		// Victim dominates; the attacker only breaks ties, hence the factor.
		score = 100000 + victim*16 - pieceValueAt(gs, m.src)
	}
	if m.promo != "" {
		score += 90000 + botPieceValue[m.promo[0]]
	}
	return score
}

func pieceValueAt(gs GameState, bit uint64) int {
	switch {
	case gs.PawnBitboard&bit != 0:
		return 100
	case gs.KnightBitboard&bit != 0:
		return 320
	case gs.BishopBitboard&bit != 0:
		return 330
	case gs.RookBitboard&bit != 0:
		return 500
	case gs.QueenBitboard&bit != 0:
		return 900
	}
	return 0
}

// hasNonPawnMaterial reports whether the side to move has any piece other than
// pawns and the king. Null-move pruning is unsound without one.
func hasNonPawnMaterial(gs GameState) bool {
	own := gs.WhiteBitboard
	if gs.Turn == "b" {
		own = gs.BlackBitboard
	}
	return own&(gs.KnightBitboard|gs.BishopBitboard|gs.RookBitboard|gs.QueenBitboard) != 0
}

func oppTurn(turn string) string {
	if turn == "w" {
		return "b"
	}
	return "w"
}

// evalMaterial scores the position in centipawns from White's perspective.
func evalMaterial(gs GameState) int {
	w := 100*bits.OnesCount64(gs.PawnBitboard&gs.WhiteBitboard) +
		320*bits.OnesCount64(gs.KnightBitboard&gs.WhiteBitboard) +
		330*bits.OnesCount64(gs.BishopBitboard&gs.WhiteBitboard) +
		500*bits.OnesCount64(gs.RookBitboard&gs.WhiteBitboard) +
		900*bits.OnesCount64(gs.QueenBitboard&gs.WhiteBitboard)
	b := 100*bits.OnesCount64(gs.PawnBitboard&gs.BlackBitboard) +
		320*bits.OnesCount64(gs.KnightBitboard&gs.BlackBitboard) +
		330*bits.OnesCount64(gs.BishopBitboard&gs.BlackBitboard) +
		500*bits.OnesCount64(gs.RookBitboard&gs.BlackBitboard) +
		900*bits.OnesCount64(gs.QueenBitboard&gs.BlackBitboard)
	return w - b
}
