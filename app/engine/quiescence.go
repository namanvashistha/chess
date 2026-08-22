package engine

import "chess-engine/app/domain/dao"

// Quiescence search.
//
// Without it, the search evaluates whatever position it happens to reach at
// depth 0 -- mid-capture-sequence or not. That is the horizon effect: the engine
// grabs a defended piece on the last ply, counts the material, and never sees
// the recapture one ply later. It is why a depth-5 search from the start
// position used to report +1.00 for a line ending in Qxb7, a move that simply
// loses the queen to Bxb7.
//
// The fix is to keep searching noisy moves (captures and promotions) past the
// nominal depth until the position is quiet, and only evaluate then.

// maxQuiescenceDepth bounds the extension. A material-only evaluation with no
// static exchange evaluation will happily explore long losing capture chains, so
// the cap is what stops a pathological position from stalling the search.
const maxQuiescenceDepth = 8

// deltaMargin is the slack in delta pruning: if the standing evaluation plus the
// piece being captured still falls this far short of alpha, the capture cannot
// rescue the position and is skipped.
const deltaMargin = 200

// isNoisyMove reports whether a move changes material -- a capture (including en
// passant) or a promotion. These are the moves quiescence follows.
func isNoisyMove(gs dao.GameState, m botMove) bool {
	if m.promo != "" {
		return true
	}
	if m.dst&(gs.WhiteBitboard|gs.BlackBitboard) != 0 {
		return true
	}
	// En passant: the destination square is empty, so the check above misses it.
	return gs.PawnBitboard&m.src != 0 && m.dst&gs.EnPassant != 0
}

// staticEval scores the position in centipawns from the side to move's view.
func staticEval(gs dao.GameState) int {
	score := evaluate(gs)
	if gs.Turn == "b" {
		return -score
	}
	return score
}

// quiescence searches noisy moves until the position is quiet, then evaluates.
//
// When the side to move is in check it searches every move instead, because
// standing pat in check would let the search assume it can decline to move out
// of one.
func quiescence(c *searchCtx, gs dao.GameState, alpha, beta, qdepth int) int {
	c.nodes++
	if c.nodes&1023 == 0 && abortRequested(c.deadline, c.stop) {
		c.aborted = true
		return 0
	}

	inCheck := isKingInCheck(gs, gs.Turn != "b")

	// Stand pat: the side to move is not obliged to capture, so the static score
	// is a lower bound on what it can achieve. Not available while in check.
	standPat := staticEval(gs)
	if !inCheck {
		if standPat >= beta {
			return standPat
		}
		if standPat > alpha {
			alpha = standPat
		}
	}

	if qdepth <= 0 {
		return standPat
	}

	moves := sideToMoveMovesOrdered(gs)
	if len(moves) == 0 {
		if inCheck {
			return -mateScore - qdepth
		}
		return 0 // stalemate
	}

	for _, m := range moves {
		if !inCheck {
			if !isNoisyMove(gs, m) {
				continue
			}
			// Delta pruning: winning this piece outright still leaves the score
			// short of alpha, so the line cannot matter.
			if standPat+pieceValueAt(gs, m.dst)+deltaMargin < alpha {
				continue
			}
		}

		score := -quiescence(c, applyBotMove(gs, m), -beta, -alpha, qdepth-1)
		if c.aborted {
			return alpha
		}
		if score >= beta {
			return score
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}
