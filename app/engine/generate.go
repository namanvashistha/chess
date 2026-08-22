package engine

import "chess-engine/app/domain/dao"

// A direct legal move generator for the search.
//
// The original generator's currency is map[uint64]uint64 keyed by a piece's
// bitboard, and it builds three of them per position. At ~30k nodes a search
// that was 1.5GB of allocation and left the garbage collector holding half the
// CPU. This produces a flat slice instead, with no map anywhere, and appends
// into a caller-supplied buffer so a search can reuse one per ply.
//
// GenerateLegalMovesForAllPositions is still the server/UI path, since the
// client wants moves grouped by square. TestGeneratorsAgree asserts the two
// produce identical move sets.

const (
	rank2Mask = uint64(0x000000000000FF00)
	rank3Mask = uint64(0x0000000000FF0000)
	rank6Mask = uint64(0x0000FF0000000000)
	rank7Mask = uint64(0x00FF000000000000)
)

// appendLegalMoves appends every legal move for the side to move to out and
// returns the extended slice. Promotions are expanded into all four pieces.
func appendLegalMoves(gs dao.GameState, out []botMove) []botMove {
	white := gs.Turn != "b"

	own, enemy := gs.BlackBitboard, gs.WhiteBitboard
	if white {
		own, enemy = gs.WhiteBitboard, gs.BlackBitboard
	}
	occupied := own | enemy
	empty := ^occupied

	// Pawns.
	for pawns := gs.PawnBitboard & own; pawns != 0; pawns &= pawns - 1 {
		from := pawns & -pawns
		var targets uint64

		if white {
			if push := from << 8; push&empty != 0 {
				targets |= push
				if from&rank2Mask != 0 && (from<<16)&empty != 0 {
					targets |= from << 16
				}
			}
			targets |= pawnAttacks(from, true) & enemy
			targets |= pawnAttacks(from, true) & gs.EnPassant & rank6Mask
		} else {
			if push := from >> 8; push&empty != 0 {
				targets |= push
				if from&rank7Mask != 0 && (from>>16)&empty != 0 {
					targets |= from >> 16
				}
			}
			targets |= pawnAttacks(from, false) & enemy
			targets |= pawnAttacks(from, false) & gs.EnPassant & rank3Mask
		}

		out = appendPieceMoves(gs, out, from, targets, true, white)
	}

	for knights := gs.KnightBitboard & own; knights != 0; knights &= knights - 1 {
		from := knights & -knights
		out = appendPieceMoves(gs, out, from, KnightAttackBitboard[from]&^own, false, white)
	}

	for bishops := gs.BishopBitboard & own; bishops != 0; bishops &= bishops - 1 {
		from := bishops & -bishops
		reach := removeBlockedMoves(from, BishopAttackBitboard[from], occupied, bishopDirections)
		out = appendPieceMoves(gs, out, from, reach&^own, false, white)
	}

	for rooks := gs.RookBitboard & own; rooks != 0; rooks &= rooks - 1 {
		from := rooks & -rooks
		reach := removeBlockedMoves(from, RookAttackBitboard[from], occupied, rookDirections)
		out = appendPieceMoves(gs, out, from, reach&^own, false, white)
	}

	for queens := gs.QueenBitboard & own; queens != 0; queens &= queens - 1 {
		from := queens & -queens
		reach := removeBlockedMoves(from, BishopAttackBitboard[from], occupied, bishopDirections) |
			removeBlockedMoves(from, RookAttackBitboard[from], occupied, rookDirections)
		out = appendPieceMoves(gs, out, from, reach&^own, false, white)
	}

	if king := gs.KingBitboard & own; king != 0 {
		out = appendPieceMoves(gs, out, king, KingAttackBitboard[king]&^own, false, white)
		out = appendCastlingMoves(gs, out, king, occupied, white)
	}

	return out
}

// appendPieceMoves filters a target bitboard down to the moves that leave the
// mover's own king safe, expanding promotions.
func appendPieceMoves(gs dao.GameState, out []botMove, from, targets uint64, isPawn, white bool) []botMove {
	for t := targets; t != 0; t &= t - 1 {
		to := t & -t
		if !leavesKingSafe(gs, from, to, white) {
			continue
		}
		if isPawn && isPromotionSquare(to, white) {
			for _, promo := range promotionChoices {
				out = append(out, botMove{src: from, dst: to, promo: promo})
			}
			continue
		}
		out = append(out, botMove{src: from, dst: to})
	}
	return out
}

// leavesKingSafe plays the move and asks whether the mover's king is attacked.
// The promotion piece cannot change the answer, so it is left as a queen.
func leavesKingSafe(gs dao.GameState, from, to uint64, white bool) bool {
	next := applyBitboardMove(gs, from, to, "q")
	king := kingSquare(next, white)
	if king == 0 {
		return true // hand-built test position with no king
	}
	return !isSquareAttacked(next, king, !white)
}

// appendCastlingMoves adds castling, which needs the king's path to be both
// empty and unattacked -- a condition the generic per-move safety filter cannot
// express, since it only ever sees the final square.
func appendCastlingMoves(gs dao.GameState, out []botMove, king, occupied uint64, white bool) []botMove {
	if white {
		if king != sqE1 {
			return out
		}
		if containsByte(gs.CastlingRights, 'K') && occupied&(sqF1|sqG1) == 0 &&
			!anyAttacked(gs, false, sqE1, sqF1, sqG1) {
			out = append(out, botMove{src: sqE1, dst: sqG1})
		}
		if containsByte(gs.CastlingRights, 'Q') && occupied&(sqD1|sqC1|squareBit("b1")) == 0 &&
			!anyAttacked(gs, false, sqE1, sqD1, sqC1) {
			out = append(out, botMove{src: sqE1, dst: sqC1})
		}
		return out
	}

	if king != sqE8 {
		return out
	}
	if containsByte(gs.CastlingRights, 'k') && occupied&(sqF8|sqG8) == 0 &&
		!anyAttacked(gs, true, sqE8, sqF8, sqG8) {
		out = append(out, botMove{src: sqE8, dst: sqG8})
	}
	if containsByte(gs.CastlingRights, 'q') && occupied&(sqD8|sqC8|squareBit("b8")) == 0 &&
		!anyAttacked(gs, true, sqE8, sqD8, sqC8) {
		out = append(out, botMove{src: sqE8, dst: sqC8})
	}
	return out
}

func anyAttacked(gs dao.GameState, byWhite bool, squares ...uint64) bool {
	for _, sq := range squares {
		if isSquareAttacked(gs, sq, byWhite) {
			return true
		}
	}
	return false
}

func containsByte(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}
