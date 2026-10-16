package engine

import (
	"testing"
)

func TestParseUCIMove(t *testing.T) {
	start := StartState()

	cases := []struct {
		name     string
		state    GameState
		uci      string
		wantPc   string
		wantProm string
	}{
		{"quiet pawn", start, "e2e4", "P", ""},
		{"black knight", func() GameState { g := start; g.Turn = "b"; return g }(), "g8f6", "n", ""},
		{"white castle king", mustFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"), "e1g1", "K", ""},
		{"underpromotion knight", mustFEN(t, "8/4P3/8/8/8/8/8/k6K w - - 0 1"), "e7e8n", "P", "n"},
		{"promotion queen", mustFEN(t, "8/4P3/8/8/8/8/8/k6K w - - 0 1"), "e7e8q", "P", "q"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := ParseUCIMove(c.state, c.uci)
			if err != nil {
				t.Fatalf("ParseUCIMove(%q): %v", c.uci, err)
			}
			if m.Piece != c.wantPc {
				t.Errorf("piece = %q, want %q", m.Piece, c.wantPc)
			}
			if m.Promotion != c.wantProm {
				t.Errorf("promotion = %q, want %q", m.Promotion, c.wantProm)
			}
			if got := MoveToUCI(m); got != c.uci {
				t.Errorf("MoveToUCI round trip = %q, want %q", got, c.uci)
			}
		})
	}
}

func TestParseUCIMoveErrors(t *testing.T) {
	start := StartState()
	bad := []string{"e2", "e2e4e5x", "z9z9", "e3e4" /* empty source */, "e7e8k" /* bad promo */}
	for _, uci := range bad {
		if _, err := ParseUCIMove(start, uci); err == nil {
			t.Errorf("ParseUCIMove(%q) expected error, got nil", uci)
		}
	}
}

func TestApplyMovePromotionUnderpromotion(t *testing.T) {
	gs := mustFEN(t, "8/4P3/8/8/8/8/8/k6K w - - 0 1")
	m, _ := ParseUCIMove(gs, "e7e8n")
	ns := ApplyMove(gs, m)

	e8 := uint64(1) << uint(PositionToIndex("e8"))
	if ns.KnightBitboard&e8 == 0 {
		t.Error("expected a knight on e8 after e7e8n")
	}
	if ns.QueenBitboard&e8 != 0 {
		t.Error("did not expect a queen on e8 after underpromotion")
	}
	if ns.PawnBitboard&e8 != 0 {
		t.Error("pawn should be gone after promotion")
	}
}

func TestApplyMoveCastlingMovesRook(t *testing.T) {
	gs := mustFEN(t, "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")
	m, _ := ParseUCIMove(gs, "e1g1")
	ns := ApplyMove(gs, m)

	g1 := uint64(1) << uint(PositionToIndex("g1"))
	f1 := uint64(1) << uint(PositionToIndex("f1"))
	h1 := uint64(1) << uint(PositionToIndex("h1"))
	if ns.KingBitboard&g1 == 0 {
		t.Error("king should be on g1 after O-O")
	}
	if ns.RookBitboard&f1 == 0 {
		t.Error("rook should have moved to f1 after O-O")
	}
	if ns.RookBitboard&h1 != 0 {
		t.Error("rook should no longer be on h1 after O-O")
	}
}

func mustFEN(t *testing.T, fen string) GameState {
	t.Helper()
	gs, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	return gs
}
