package engine

// InCheck reports whether the side to move is in check.
func InCheck(gs GameState) bool {
	return isKingInCheck(gs, gs.Turn == "w")
}
