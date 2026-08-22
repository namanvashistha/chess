package engine

import "chess-engine/app/domain/dao"

// Single-square attack detection.
//
// Legality checking used to answer "is my king in check?" by generating every
// pseudo-legal move for both colours and scanning the result. That ran once per
// candidate move, so producing one legal move list cost roughly 560 full move
// generations -- 7.8ms on Kiwipete, and ~96% of all search time.
//
// The question only ever concerns one square, so it only needs the pieces that
// can reach that square. Each test is a handful of table lookups and at most two
// ray traces.

var (
	bishopDirections = []int{-9, -7, 7, 9}
	rookDirections   = []int{-8, 8, -1, 1}
)

const (
	fileAMask = uint64(0x0101010101010101)
	fileHMask = uint64(0x8080808080808080)
)

// pawnAttacks returns every square attacked by the given set of pawns. The file
// masks stop a capture wrapping around the board edge.
//
// This cannot use WhitePawnAttackBitboard / BlackPawnAttackBitboard: despite the
// names, those tables hold the single forward PUSH square, not the diagonal
// captures, so using them reported a king as checked by a pawn directly in
// front of it.
func pawnAttacks(pawns uint64, white bool) uint64 {
	if white {
		return ((pawns &^ fileAMask) << 7) | ((pawns &^ fileHMask) << 9)
	}
	return ((pawns &^ fileAMask) >> 9) | ((pawns &^ fileHMask) >> 7)
}

// isSquareAttacked reports whether square is attacked by the given colour.
// square must be a single set bit.
func isSquareAttacked(gs dao.GameState, square uint64, byWhite bool) bool {
	attackers := gs.BlackBitboard
	if byWhite {
		attackers = gs.WhiteBitboard
	}
	if attackers == 0 {
		return false
	}

	if pawns := gs.PawnBitboard & attackers; pawns != 0 {
		if pawnAttacks(pawns, byWhite)&square != 0 {
			return true
		}
	}

	if KnightAttackBitboard[square]&gs.KnightBitboard&attackers != 0 {
		return true
	}
	// A king guards its neighbours whether or not it may legally move there.
	if KingAttackBitboard[square]&gs.KingBitboard&attackers != 0 {
		return true
	}

	occupied := gs.WhiteBitboard | gs.BlackBitboard

	if diagonal := (gs.BishopBitboard | gs.QueenBitboard) & attackers; diagonal != 0 {
		reach := removeBlockedMoves(square, BishopAttackBitboard[square], occupied, bishopDirections)
		if reach&diagonal != 0 {
			return true
		}
	}

	if orthogonal := (gs.RookBitboard | gs.QueenBitboard) & attackers; orthogonal != 0 {
		reach := removeBlockedMoves(square, RookAttackBitboard[square], occupied, rookDirections)
		if reach&orthogonal != 0 {
			return true
		}
	}

	return false
}

// kingSquare returns the given colour's king, or 0 if it has none (which only
// happens in hand-built test positions).
func kingSquare(gs dao.GameState, white bool) uint64 {
	if white {
		return gs.KingBitboard & gs.WhiteBitboard
	}
	return gs.KingBitboard & gs.BlackBitboard
}
