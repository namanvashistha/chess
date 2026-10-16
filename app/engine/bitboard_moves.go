package engine

// ApplyMove plays move on state with no legality check (Game.Play checks) and
// returns the resulting state. Handles captures, castling (rook relocation + rights),
// en passant (set/clear + capture), and promotion. The promotion target honors
// move.Promotion ("q"|"r"|"b"|"n", case-insensitive); empty defaults to queen.
func ApplyMove(state GameState, move Move) GameState {
	src := uint64(1) << uint(PositionToIndex(move.Source))
	dst := uint64(1) << uint(PositionToIndex(move.Destination))

	// The moving piece is taken from the board, not from move.Piece: a caller
	// that mislabels the piece can no longer move something else.
	ns := applyBitboardMove(state, src, dst, move.Promotion)
	ns.Turn = ToggleTurn(state.Turn)
	return ns
}

func ToggleTurn(currentTurn string) string {
	if currentTurn == "w" {
		return "b"
	}
	return "w"
}
