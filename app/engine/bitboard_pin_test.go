package engine

import (
	"chess-engine/app/domain/dao"
	"testing"
)

// White Ke1, Be2; Black Re8 (pins the bishop on the e-file), Black Ka8.
// The bishop is absolutely pinned: every bishop move leaves the e-file and
// exposes the white king, so it must have ZERO legal moves.
func TestPinnedBishopHasNoLegalMoves(t *testing.T) {
	gs := dao.GameState{
		WhiteBitboard:  (1 << 4) | (1 << 12),
		BlackBitboard:  (1 << 60) | (1 << 56),
		KingBitboard:   (1 << 4) | (1 << 56),
		BishopBitboard: (1 << 12),
		RookBitboard:   (1 << 60),
		Turn:           "w",
	}
	legal, _ := GenerateLegalMovesForAllPositions(gs)
	bishop := uint64(1) << 12
	if legal[bishop] != 0 {
		t.Fatalf("pinned bishop must have no legal moves; got %d squares", popcount(legal[bishop]))
	}
}

func popcount(x uint64) int {
	n := 0
	for x != 0 {
		n++
		x &= x - 1
	}
	return n
}

// White Ke1, Pe2; Black Qe8 (pins pawn on e-file), Black Nd3, Black Ka8.
// The pawn may push e3/e4 (stays on the file) but capturing d3 (exd3) would
// leave the e-file and expose the king -> illegal. The capture must be excluded.
func TestPinnedPawnCannotCaptureOffFile(t *testing.T) {
	gs := dao.GameState{
		WhiteBitboard:  (1 << 4) | (1 << 12),
		BlackBitboard:  (1 << 60) | (1 << 19) | (1 << 56),
		KingBitboard:   (1 << 4) | (1 << 56),
		QueenBitboard:  (1 << 60),
		KnightBitboard: (1 << 19),
		PawnBitboard:   (1 << 12),
		Turn:           "w",
	}
	legal, _ := GenerateLegalMovesForAllPositions(gs)
	pawn := uint64(1) << 12
	illegalCapture := uint64(1) << 19 // d3
	if legal[pawn]&illegalCapture != 0 {
		t.Fatalf("pinned pawn must not be allowed to capture off-file (exd3); legal=%b", legal[pawn])
	}
}

// White Ke1, Ne4 (knight blocks the e-file); Black Re8, Black Ka8.
// The knight is pinned: ANY knight move uncovers Re8 -> Ke1 (discovered check),
// so the knight must have ZERO legal moves.
func TestKnightPinnedByDiscoveredCheck(t *testing.T) {
	gs := dao.GameState{
		WhiteBitboard:  (1 << 4) | (1 << 28),
		BlackBitboard:  (1 << 60) | (1 << 56),
		KingBitboard:   (1 << 4) | (1 << 56),
		KnightBitboard: (1 << 28),
		RookBitboard:   (1 << 60),
		Turn:           "w",
	}
	legal, _ := GenerateLegalMovesForAllPositions(gs)
	knight := uint64(1) << 28
	if legal[knight] != 0 {
		t.Fatalf("pinned knight must have no legal moves (discovered check); got %d squares: %b", popcount(legal[knight]), legal[knight])
	}
}
