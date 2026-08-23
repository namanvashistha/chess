package engine

import "chess-engine/app/domain/dao"

// Static exchange evaluation.
//
// Quiescence follows every capture, including ones that are obviously terrible
// -- a queen taking a pawn defended by a pawn. The search then spends real work
// proving what arithmetic could have said immediately. SEE plays out the
// recaptures on a single square, cheapest attacker first, and returns the
// material balance without searching anything.

// seePieceValue mirrors the search's material scale. The king is given a value
// large enough that no exchange ever wants to trade it.
var seePieceValue = [kindKing + 1]int{0, 100, 320, 330, 500, 900, 20000}

// attackersTo returns every piece of either colour bearing on square.
//
// occupied is passed separately rather than read from gs so the caller can
// remove pieces as an exchange proceeds; sliding attackers hidden behind them
// then appear, which is how x-rays are handled.
func attackersTo(gs dao.GameState, square, occupied uint64) uint64 {
	var att uint64

	// A white pawn attacks `square` exactly when it stands on a square a black
	// pawn placed on `square` would attack, and vice versa.
	att |= pawnAttacks(square, false) & gs.PawnBitboard & gs.WhiteBitboard
	att |= pawnAttacks(square, true) & gs.PawnBitboard & gs.BlackBitboard

	att |= KnightAttackBitboard[square] & gs.KnightBitboard
	att |= KingAttackBitboard[square] & gs.KingBitboard
	att |= removeBlockedMoves(square, BishopAttackBitboard[square], occupied, bishopDirections) &
		(gs.BishopBitboard | gs.QueenBitboard)
	att |= removeBlockedMoves(square, RookAttackBitboard[square], occupied, rookDirections) &
		(gs.RookBitboard | gs.QueenBitboard)

	return att & occupied
}

// leastValuableAttacker returns the cheapest attacker belonging to one colour.
func leastValuableAttacker(gs dao.GameState, attackers uint64, white bool) (uint64, pieceKind) {
	side := gs.BlackBitboard
	if white {
		side = gs.WhiteBitboard
	}
	for _, b := range [...]struct {
		bb   uint64
		kind pieceKind
	}{
		{gs.PawnBitboard, kindPawn},
		{gs.KnightBitboard, kindKnight},
		{gs.BishopBitboard, kindBishop},
		{gs.RookBitboard, kindRook},
		{gs.QueenBitboard, kindQueen},
		{gs.KingBitboard, kindKing},
	} {
		if s := attackers & side & b.bb; s != 0 {
			return s & -s, b.kind
		}
	}
	return 0, kindNone
}

// see returns the centipawns a capture from->to wins (positive) or loses
// (negative), assuming both sides recapture optimally.
//
// It does not understand en passant: the captured pawn is not on the
// destination square, so callers skip SEE for those.
func see(gs dao.GameState, from, to uint64) int {
	var gain [32]int

	occupied := gs.WhiteBitboard | gs.BlackBitboard
	white := gs.WhiteBitboard&from != 0

	gain[0] = seePieceValue[kindAt(gs, to)]
	attackerBit, attackerKind := from, kindAt(gs, from)

	depth := 0
	for {
		depth++
		if depth >= len(gain) {
			break
		}
		gain[depth] = seePieceValue[attackerKind] - gain[depth-1]

		occupied &^= attackerBit
		white = !white

		attackers := attackersTo(gs, to, occupied)
		next, nextKind := leastValuableAttacker(gs, attackers, white)
		if next == 0 {
			break
		}
		// A king may not capture into a square the other side still attacks, so
		// that recapture is not available.
		if nextKind == kindKing {
			opponent := gs.WhiteBitboard
			if white {
				opponent = gs.BlackBitboard
			}
			if attackers&opponent != 0 {
				break
			}
		}
		attackerBit, attackerKind = next, nextKind
	}

	// Fold back: at each step the side to move may decline to recapture, so it
	// takes the better of "stand here" and "continue the exchange".
	for depth > 1 {
		depth--
		best := -gain[depth-1]
		if gain[depth] > best {
			best = gain[depth]
		}
		gain[depth-1] = -best
	}
	return gain[0]
}
