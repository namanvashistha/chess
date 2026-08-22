package engine

import (
	"chess-engine/app/domain/dao"
	"sort"
)

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
// promoting pawn move expanded into its four choices.
//
// The order is deterministic (by source square, then destination, then promotion
// piece). The underlying generator returns a Go map, whose iteration order is
// randomised, which previously made the engine pick a different move every run
// from the same position and made any search regression impossible to reproduce.
func GenerateLegalMoveList(gs dao.GameState) []LegalMove {
	all, _ := GenerateLegalMovesForAllPositions(gs)
	return expandPromotions(gs, FilterMovesByTurn(all, gs))
}

func expandPromotions(gs dao.GameState, moves map[uint64]uint64) []LegalMove {
	out := make([]LegalMove, 0, 48)
	for src, dsts := range moves {
		isPawn := gs.PawnBitboard&src != 0
		isWhite := gs.WhiteBitboard&src != 0
		for d := dsts; d != 0; d &= d - 1 {
			dst := d & -d
			if isPawn && isPromotionSquare(dst, isWhite) {
				for _, promo := range promotionChoices {
					out = append(out, LegalMove{Src: src, Dst: dst, Promotion: promo})
				}
				continue
			}
			out = append(out, LegalMove{Src: src, Dst: dst})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Src != out[j].Src {
			return out[i].Src < out[j].Src
		}
		if out[i].Dst != out[j].Dst {
			return out[i].Dst < out[j].Dst
		}
		return out[i].Promotion < out[j].Promotion
	})
	return out
}
