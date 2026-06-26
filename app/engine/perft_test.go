package engine

import (
	"chess-engine/app/domain/dto"
	"chess-engine/app/domain/dao"
	"testing"
)

// perft counts leaf nodes to the given depth by enumerating the side-to-move's
// legal moves and applying each via ApplyMove. It exercises FEN parsing, move
// generation, and ApplyMove together. (Promotions are counted once, as a queen,
// so only use it at depths where no promotion occurs.)
func perft(gs dao.GameState, depth int) int {
	if depth == 0 {
		return 1
	}
	all, _ := GenerateLegalMovesForAllPositions(gs)
	moves := FilterMovesByTurn(all, gs)
	nodes := 0
	for src, dsts := range moves {
		for d := dsts; d != 0; d &= d - 1 {
			dst := d & -d
			move := dto.Move{
				Piece:       getPieceCode(src, gs.WhiteBitboard&src != 0, gs),
				Source:      bitToSquare(src, defFiles, defRanks),
				Destination: bitToSquare(dst, defFiles, defRanks),
			}
			ns := ApplyMove(gs, move)
			nodes += perft(ns, depth-1)
		}
	}
	return nodes
}

// Standard perft reference counts for the starting position (CPW). A mismatch
// indicates a move-generation or move-application defect, not a test bug.
func TestPerftStartPosition(t *testing.T) {
	want := map[int]int{1: 20, 2: 400, 3: 8902}
	gs := StartState()
	for depth := 1; depth <= 3; depth++ {
		if got := perft(gs, depth); got != want[depth] {
			t.Errorf("perft(start, %d) = %d, want %d", depth, got, want[depth])
		}
	}
}
