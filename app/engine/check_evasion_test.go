package engine

import (
	"testing"
)

// Position BEFORE black's suspect ...c5: white Bb5+ checks the black king on e8
// along b5-c6-d7-e8 (both intermediate squares empty). Black is to move and IN
// CHECK, so the ONLY legal moves are check evasions (block on c6/d7, capture the
// bishop, or move the king). ...c7-c5 does NOT address the check and must be
// excluded. If it appears in the legal moves, the engine has a check-evasion bug.
func TestInCheckMustNotAllowIrrelevantPawnPush(t *testing.T) {
	gs, err := ParseFEN("rnbqkb1r/ppp1pppp/3p1n2/1B6/4P3/2N5/PPPP1PPP/R1BQK1NR b KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}

	if !isKingInCheck(gs, false) {
		t.Fatal("setup error: black king should be in check from Bb5")
	}

	all, status := GenerateLegalMovesForAllPositions(gs)
	legal := FilterMovesByTurn(all, gs)
	asMap := ConvertLegalMovesToMap(legal)

	t.Logf("status = %q", status)
	t.Logf("c7 legal moves = %v", asMap["c7"])

	for _, dst := range asMap["c7"] {
		if dst == "c5" || dst == "c6" {
			// c6 WOULD be a legal block; c5 never is. Report both for clarity.
			if dst == "c5" {
				t.Errorf("BUG: engine allows ...c7c5 while king is in check (does not resolve check)")
			}
		}
	}

	// Count total legal moves: a correct engine has only a handful of evasions here.
	total := 0
	for _, dsts := range legal {
		total += popcount(dsts)
	}
	t.Logf("total legal moves for black in check = %d", total)
}
