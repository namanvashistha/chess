package engine

import "testing"

func TestStaticExchangeEvaluation(t *testing.T) {
	cases := []struct {
		name     string
		fen      string
		from, to string
		want     int
	}{
		// Queen takes an undefended pawn: wins a pawn outright.
		{"free pawn", "4k3/8/8/3p4/8/8/8/3QK3 w - - 0 1", "d1", "d5", 100},
		// Queen takes a pawn defended by a pawn: wins 100, loses 900.
		{"queen grabs a defended pawn", "4k3/8/2p5/3p4/8/8/8/3QK3 w - - 0 1", "d1", "d5", -800},
		// Pawn takes a pawn defended by a pawn: even trade.
		{"pawn takes defended pawn", "4k3/8/2p5/3p4/4P3/8/8/4K3 w - - 0 1", "e4", "d5", 0},
		// Rook takes an undefended knight: wins the piece.
		{"undefended knight", "4k3/8/3n4/8/8/3R4/3R4/4K3 w - - 0 1", "d3", "d6", 320},
		// Rook takes a knight defended by a pawn, second rook behind it:
		// +320 knight, -500 rook, +100 the second rook recaptures the pawn = -80.
		// Negative, so quiescence skips it.
		{"defended knight costs the exchange", "4k3/2p5/3n4/8/8/3R4/3R4/4K3 w - - 0 1", "d3", "d6", -80},
	}

	for _, c := range cases {
		gs, err := ParseFEN(c.fen)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		got := see(gs, squareBit(c.from), squareBit(c.to))
		if got != c.want {
			t.Errorf("%s: see(%s%s) = %d, want %d", c.name, c.from, c.to, got, c.want)
		}
	}
}
