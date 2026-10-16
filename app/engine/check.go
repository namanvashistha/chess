package engine

import "chess-engine/app/domain/dao"

// InCheck reports whether the side to move is in check.
func InCheck(gs dao.GameState) bool {
	return isKingInCheck(gs, gs.Turn == "w")
}
