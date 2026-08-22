package engine

import "chess-engine/app/domain/dao"

// LegalMove is a fully specified move. It exists because the generator's native
// output -- map[srcBit]dstBitboard -- has nowhere to record a promotion choice,
// so a promoting move was indistinguishable from a single move and the engine
// could only ever promote to a queen. Underpromotion (a rook to avoid giving
// stalemate, a knight to fork) was unreachable, and perft undercounted every
// promoting position by three moves per promotion square.
type LegalMove struct {
	Src       uint64
	Dst       uint64
	Promotion string // "", or one of "q" "r" "b" "n"
}

// promotionChoices is ordered q, r, b, n so the first entry stays the historical
// default when callers only look at one.
var promotionChoices = [4]string{"q", "r", "b", "n"}

// GenerateLegalMoveList returns every legal move for the side to move, with each
// promoting pawn move expanded into its four choices, in a deterministic order.
func GenerateLegalMoveList(gs dao.GameState) []LegalMove {
	moves := appendLegalMoves(gs, nil)
	out := make([]LegalMove, 0, len(moves))
	for _, m := range moves {
		out = append(out, LegalMove{Src: m.src, Dst: m.dst, Promotion: m.promo})
	}
	return out
}
