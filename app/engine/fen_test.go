package engine

import (
	"chess-engine/app/domain/dao"
	"testing"
)

// StartState must match the hardcoded starting bitboards used by the game server.
func TestStartStateBitboards(t *testing.T) {
	gs := StartState()
	want := dao.GameState{
		WhiteBitboard:  0xFFFF,
		BlackBitboard:  0xFFFF000000000000,
		PawnBitboard:   0x00FF00000000FF00,
		RookBitboard:   0x8100000000000081,
		KnightBitboard: 0x4200000000000042,
		BishopBitboard: 0x2400000000000024,
		QueenBitboard:  0x0800000000000008,
		KingBitboard:   0x1000000000000010,
		CastlingRights: "KQkq",
		Turn:           "w",
	}
	if gs.WhiteBitboard != want.WhiteBitboard || gs.BlackBitboard != want.BlackBitboard ||
		gs.PawnBitboard != want.PawnBitboard || gs.RookBitboard != want.RookBitboard ||
		gs.KnightBitboard != want.KnightBitboard || gs.BishopBitboard != want.BishopBitboard ||
		gs.QueenBitboard != want.QueenBitboard || gs.KingBitboard != want.KingBitboard ||
		gs.CastlingRights != want.CastlingRights || gs.Turn != want.Turn {
		t.Fatalf("StartState mismatch:\n got %+v\nwant %+v", gs, want)
	}
}

func TestFENRoundTrip(t *testing.T) {
	cases := []string{
		StartFEN,
		"rnbqkbnr/pp1ppppp/8/2p5/4P3/8/PPPP1PPP/RNBQKBNR w KQkq c6 0 1",        // en passant target c6 (black just pushed)
		"rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",        // en passant target e3 (white just pushed)
		"r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",                                 // castling rights all sides
		"8/8/8/8/8/8/8/4K2k w - - 0 1",                                         // sparse, no castling/ep
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1", // Kiwipete
	}
	for _, fen := range cases {
		gs, err := ParseFEN(fen)
		if err != nil {
			t.Fatalf("ParseFEN(%q) error: %v", fen, err)
		}
		if got := ToFEN(gs); got != fen {
			t.Errorf("round trip mismatch:\n in:  %q\n out: %q", fen, got)
		}
	}
}

// The engine encodes en passant as two bits: the target square plus the pushed
// pawn's square (which overlaps the mover's color so the generator's guard fires).
func TestFENEnPassantDoubleBit(t *testing.T) {
	// White just played e2-e4; black to move; target e3, pushed pawn on e4.
	gs, err := ParseFEN("rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1")
	if err != nil {
		t.Fatal(err)
	}
	e3 := uint64(1) << uint(PositionToIndex("e3"))
	e4 := uint64(1) << uint(PositionToIndex("e4"))
	if gs.EnPassant != (e3 | e4) {
		t.Fatalf("EnPassant = %#x, want %#x (e3|e4)", gs.EnPassant, e3|e4)
	}
	if gs.EnPassant&gs.WhiteBitboard == 0 {
		t.Fatal("EnPassant must overlap White (the pushed pawn) so black's EP guard fires")
	}
}

func TestParseFENErrors(t *testing.T) {
	bad := []string{
		"",
		"only/three w - -",
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP w KQkq - 0 1", // 7 ranks
		"rnbqkbnr/pppppppp/8/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", // 9 ranks
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNX w KQkq - 0 1",   // bad piece X
		StartFEN[:len(StartFEN)-len("w KQkq - 0 1")] + "x KQkq - 0 1", // bad side
	}
	for _, fen := range bad {
		if _, err := ParseFEN(fen); err == nil {
			t.Errorf("ParseFEN(%q) expected error, got nil", fen)
		}
	}
}
