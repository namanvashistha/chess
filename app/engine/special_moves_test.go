package engine

import (
	"sort"
	"strings"
	"testing"
)

func movesFor(t *testing.T, fen string) []string {
	t.Helper()
	gs, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	var out []string
	for _, m := range GenerateLegalMoveList(gs) {
		out = append(out, bitToSquare(m.Src, defFiles, defRanks)+
			bitToSquare(m.Dst, defFiles, defRanks)+m.Promotion)
	}
	sort.Strings(out)
	return out
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// An en-passant capture that vacates its own rank and removes the blocking pawn
// can expose the capturing side's king. CPW position 3 is the canonical case:
// after g2g4, black's f4xg3 e.p. would leave kh4 in check from Rb4, because the
// capture empties both f4 and g4.
func TestEnPassantDiscoveredCheckIsIllegal(t *testing.T) {
	cases := []struct {
		fen    string
		banned string
	}{
		{"8/2p5/3p4/KP5r/1R3pPk/8/4P3/8 b - g3 0 1", "f4g3"},
		{"8/2p5/3p4/KP5r/1R2Pp1k/8/6P1/8 b - e3 0 1", "f4e3"},
	}
	for _, c := range cases {
		got := movesFor(t, c.fen)
		if contains(got, c.banned) {
			t.Errorf("%s: %s is illegal (discovered check on the 4th rank), got moves %v",
				c.fen, c.banned, got)
		}
	}
}

// The mirror case: an en-passant capture that is legal must still be offered,
// and must actually remove the captured pawn.
func TestEnPassantCaptureWorks(t *testing.T) {
	const fen = "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1"
	got := movesFor(t, fen)
	if !contains(got, "e5d6") {
		t.Fatalf("expected the en-passant capture e5d6 to be legal, got %v", got)
	}

	gs, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	ns := applyBitboardMove(gs, squareBit("e5"), squareBit("d6"), "")
	if ns.PawnBitboard&ns.BlackBitboard&squareBit("d5") != 0 {
		t.Error("captured pawn is still on d5 after en passant")
	}
	if ns.PawnBitboard&ns.WhiteBitboard&squareBit("d6") == 0 {
		t.Error("capturing pawn is not on d6")
	}
	if evalMaterial(ns) != 100 {
		t.Errorf("evalMaterial = %d, want 100 (a pawn up)", evalMaterial(ns))
	}
}

// All four promotion pieces must be generated, and each must actually appear on
// the board. The generator's map[srcBit]dstBitboard could not express the
// choice, so only a queen was ever reachable.
func TestUnderpromotionIsGeneratedAndApplied(t *testing.T) {
	const fen = "8/4P3/8/8/8/8/8/k6K w - - 0 1"
	got := movesFor(t, fen)
	for _, want := range []string{"e7e8q", "e7e8r", "e7e8b", "e7e8n"} {
		if !contains(got, want) {
			t.Errorf("missing promotion %s; got %v", want, got)
		}
	}

	gs, err := ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	e8 := squareBit("e8")
	checks := []struct {
		promo string
		want  string
	}{
		{"q", "queen"}, {"r", "rook"}, {"b", "bishop"}, {"n", "knight"},
	}
	for _, c := range checks {
		ns := applyBitboardMove(gs, squareBit("e7"), e8, c.promo)
		if ns.PawnBitboard&e8 != 0 {
			t.Errorf("promo %q: a pawn is still on e8", c.promo)
		}
		var on uint64
		switch c.promo {
		case "q":
			on = ns.QueenBitboard
		case "r":
			on = ns.RookBitboard
		case "b":
			on = ns.BishopBitboard
		case "n":
			on = ns.KnightBitboard
		}
		if on&e8 == 0 {
			t.Errorf("promo %q: no %s on e8", c.promo, c.want)
		}
	}
}

// Castling must move the rook. simulateMove used to move only the king, leaving
// the rook behind for the whole search tree.
func TestCastlingRelocatesRookAndClearsRights(t *testing.T) {
	gs, err := ParseFEN("r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}

	ns := applyBitboardMove(gs, squareBit("e1"), squareBit("g1"), "")
	if ns.KingBitboard&squareBit("g1") == 0 {
		t.Error("king should be on g1")
	}
	if ns.RookBitboard&squareBit("f1") == 0 {
		t.Error("rook should have moved to f1")
	}
	if ns.RookBitboard&squareBit("h1") != 0 {
		t.Error("rook should no longer be on h1")
	}
	if strings.ContainsAny(ns.CastlingRights, "KQ") {
		t.Errorf("white castling rights should be gone, got %q", ns.CastlingRights)
	}
	if !strings.ContainsAny(ns.CastlingRights, "kq") {
		t.Errorf("black castling rights should survive, got %q", ns.CastlingRights)
	}
}

// A rook captured on its home square ends that side's castling right. This case
// was not handled at all.
func TestCapturingRookClearsCastlingRight(t *testing.T) {
	gs, err := ParseFEN("r3k2r/8/8/8/8/8/8/R3K1nR b KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	// Black knight on g1 takes the rook on h1.
	ns := applyBitboardMove(gs, squareBit("g1"), squareBit("h1"), "")
	if strings.Contains(ns.CastlingRights, "K") {
		t.Errorf("K right should be gone after Rh1 is captured, got %q", ns.CastlingRights)
	}
	if !strings.Contains(ns.CastlingRights, "Q") {
		t.Errorf("Q right should survive, got %q", ns.CastlingRights)
	}
}

// The move list must be reproducible: it is built from a Go map, whose iteration
// order is randomised.
func TestMoveListIsDeterministic(t *testing.T) {
	gs, err := ParseFEN("r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	first := GenerateLegalMoveList(gs)
	for i := 0; i < 20; i++ {
		next := GenerateLegalMoveList(gs)
		if len(next) != len(first) {
			t.Fatalf("run %d: length %d != %d", i, len(next), len(first))
		}
		for k := range first {
			if next[k] != first[k] {
				t.Fatalf("run %d: move %d differs: %+v vs %+v", i, k, next[k], first[k])
			}
		}
	}
}
