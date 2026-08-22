package engine

import "testing"

func TestPassedPawnDetection(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		// score from White's perspective must be at least this much better than
		// the same material without the passer.
		wantWhiteBetter bool
	}{
		// White c6 pawn: no black pawn on b, c or d ahead of it. Passed.
		{"white passer on the sixth", "4k3/8/2P5/8/8/8/p7/4K3 w - - 0 1", true},
		// Same pawn, but a black pawn on d7 stops it. Not passed.
		{"blocked by an adjacent pawn", "4k3/3p4/2P5/8/8/8/p7/4K3 w - - 0 1", false},
	}

	var scores []int
	for _, c := range cases {
		gs, err := ParseFEN(c.fen)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		scores = append(scores, passedPawnScore(gs))
		t.Logf("%-28s passedPawnScore = %d", c.name, passedPawnScore(gs))
	}

	if scores[0] <= scores[1] {
		t.Errorf("a passed pawn must score better than a blocked one: %d vs %d", scores[0], scores[1])
	}
}

// A pawn on the seventh must be worth far more than one on the third.
func TestPassedPawnScalesWithRank(t *testing.T) {
	near, _ := ParseFEN("4k3/2P5/8/8/8/8/8/4K3 w - - 0 1")
	far, _ := ParseFEN("4k3/8/8/8/8/2P5/8/4K3 w - - 0 1")
	if passedPawnScore(near) <= passedPawnScore(far) {
		t.Errorf("seventh-rank passer (%d) should beat third-rank passer (%d)",
			passedPawnScore(near), passedPawnScore(far))
	}
}
