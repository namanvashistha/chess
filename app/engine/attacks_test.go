package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"testing"
)

// isKingInCheckSlow is the original implementation: generate every pseudo-legal
// move for both colours and scan the result. It is kept solely as the reference
// that the fast single-square detector is checked against.
func isKingInCheckSlow(gs dao.GameState, isWhiteKing bool) bool {
	pseudo, _ := GenerateInitialMoves(gs)
	in, _ := CheckIfKingIsInCheck(gs, pseudo, isWhiteKing)
	return in
}

// isSquareAttacked replaced a full both-colour move generation per legality
// check. This walks every position reachable in four plies from CPW position 3
// -- chosen because it is the pin and en-passant heavy one -- and asserts the
// fast detector agrees with the exhaustive one for both colours at every node.
//
// It caught a real bug: the first version read WhitePawnAttackBitboard, which
// despite its name holds the forward push square rather than the diagonal
// captures, so a king was reported in check from a pawn standing directly in
// front of it.
func TestFastCheckDetectionMatchesExhaustive(t *testing.T) {
	gs := mustFEN(t, "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1")
	found := 0

	var walk func(gs dao.GameState, depth int, line string)
	walk = func(gs dao.GameState, depth int, line string) {
		if found > 2 {
			return
		}
		for _, white := range []bool{true, false} {
			fast := isKingInCheck(gs, white)
			slow := isKingInCheckSlow(gs, white)
			if fast != slow {
				found++
				t.Errorf("MISMATCH white=%v fast=%v slow=%v\n  fen:  %s\n  line: %s",
					white, fast, slow, ToFEN(gs), line)
				return
			}
		}
		if depth == 0 {
			return
		}
		for _, m := range GenerateLegalMoveList(gs) {
			mv := dto.Move{
				Piece:       getPieceCode(m.Src, gs.WhiteBitboard&m.Src != 0, gs),
				Source:      bitToSquare(m.Src, defFiles, defRanks),
				Destination: bitToSquare(m.Dst, defFiles, defRanks),
				Promotion:   m.Promotion,
			}
			walk(ApplyMove(gs, mv), depth-1, line+" "+mv.Source+mv.Destination+mv.Promotion)
		}
	}
	walk(gs, 4, "")
	if found == 0 {
		t.Log("fast and exhaustive check detection agree on every reachable position")
	}
}
