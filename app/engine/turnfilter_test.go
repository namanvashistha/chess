package engine

import (
	"chess-engine/app/domain/dao"
	"testing"
)

func startState(turn string) dao.GameState {
	return dao.GameState{
		WhiteBitboard:  0xFFFF,
		BlackBitboard:  0xFFFF000000000000,
		PawnBitboard:   0x00FF00000000FF00,
		RookBitboard:   0x8100000000000081,
		KnightBitboard: 0x4200000000000042,
		BishopBitboard: 0x2400000000000024,
		QueenBitboard:  0x0800000000000008,
		KingBitboard:   0x1000000000000010,
		Turn:           turn,
	}
}

func countMoves(m map[string][]string) int {
	n := 0
	for _, v := range m {
		n += len(v)
	}
	return n
}

func TestLegalMovesFilteredToSideToMove(t *testing.T) {
	for _, turn := range []string{"w", "b"} {
		gs := startState(turn)
		raw, _ := GenerateLegalMovesForAllPositions(gs)
		filtered := FilterMovesByTurn(raw, gs)
		out := ConvertLegalMovesToMap(filtered)

		// Opening position: exactly 20 legal moves for the side to move.
		if got := countMoves(out); got != 20 {
			t.Errorf("turn=%s: expected 20 moves, got %d (%v)", turn, got, out)
		}

		// Every move source square must belong to the side to move.
		sideMask := gs.WhiteBitboard
		if turn == "b" {
			sideMask = gs.BlackBitboard
		}
		for sq := range out {
			bit := uint64(1) << uint(PositionToIndex(sq))
			if bit&sideMask == 0 {
				t.Errorf("turn=%s: move map contains opponent square %s", turn, sq)
			}
		}
	}
}
