package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"strings"
	"testing"
)

func TestPositionKeyDistinguishesState(t *testing.T) {
	start := StartState()

	// Same pieces, different side to move.
	other := start
	other.Turn = "b"
	if PositionKey(start) == PositionKey(other) {
		t.Error("side to move must change the key")
	}

	// Same pieces, different castling rights.
	other = start
	other.CastlingRights = "Kq"
	if PositionKey(start) == PositionKey(other) {
		t.Error("castling rights must change the key")
	}

	// Identical states must hash identically, and the table must be stable
	// across calls (it is seeded from a constant, never from time or crypto).
	if PositionKey(start) != PositionKey(StartState()) {
		t.Error("identical positions must produce identical keys")
	}
}

// A knight out and back returns to the start position, which must hash the same.
func TestPositionKeyRepeatsAfterRoundTrip(t *testing.T) {
	gs := StartState()
	before := PositionKey(gs)

	for _, uci := range []string{"g1f3", "g8f6", "f3g1", "f6g8"} {
		m, err := ParseUCIMove(gs, uci)
		if err != nil {
			t.Fatalf("ParseUCIMove(%q): %v", uci, err)
		}
		gs = ApplyMove(gs, m)
	}

	if PositionKey(gs) != before {
		t.Error("position after Nf3 Nf6 Ng1 Ng8 should hash the same as the start")
	}
}

func TestHalfmoveClock(t *testing.T) {
	gs := StartState()
	if gs.HalfmoveClock != 0 {
		t.Fatalf("start clock = %d, want 0", gs.HalfmoveClock)
	}

	// A knight move increments.
	gs = ApplyMove(gs, mustUCI(t, gs, "g1f3"))
	if gs.HalfmoveClock != 1 {
		t.Errorf("after Nf3 clock = %d, want 1", gs.HalfmoveClock)
	}

	// A pawn move resets.
	gs = ApplyMove(gs, mustUCI(t, gs, "e7e5"))
	if gs.HalfmoveClock != 0 {
		t.Errorf("after a pawn move clock = %d, want 0", gs.HalfmoveClock)
	}

	// A capture resets: Nxe5.
	gs = ApplyMove(gs, mustUCI(t, gs, "f3e5"))
	if gs.HalfmoveClock != 0 {
		t.Errorf("after a capture clock = %d, want 0", gs.HalfmoveClock)
	}

	// And it round-trips through FEN.
	gs.HalfmoveClock = 37
	fen := ToFEN(gs)
	if !strings.Contains(fen, " 37 ") {
		t.Errorf("ToFEN did not emit the halfmove clock: %s", fen)
	}
	parsed, err := ParseFEN(fen)
	if err != nil {
		t.Fatalf("ParseFEN(%q): %v", fen, err)
	}
	if parsed.HalfmoveClock != 37 {
		t.Errorf("round trip clock = %d, want 37", parsed.HalfmoveClock)
	}
}

func TestDrawStatusThreefold(t *testing.T) {
	// Nf3 Nf6 Ng1 Ng8 twice returns to the start position for the third time.
	moves := []string{
		"Ng1f3", "ng8f6", "Nf3g1", "nf6g8",
		"Ng1f3", "ng8f6", "Nf3g1", "nf6g8",
	}
	keys := ReplayGameKeys(moves)
	if got := len(keys); got != len(moves)+1 {
		t.Fatalf("replayed %d keys, want %d", got, len(moves)+1)
	}

	gs := StartState()
	if got := DrawStatus(gs, keys); got != DrawByRepetition {
		t.Errorf("DrawStatus = %q, want %q", got, DrawByRepetition)
	}

	// Two occurrences is not yet a draw.
	if got := DrawStatus(gs, keys[:5]); got != "" {
		t.Errorf("DrawStatus after one round trip = %q, want no draw", got)
	}
}

func TestDrawStatusFiftyMove(t *testing.T) {
	gs := StartState()
	gs.HalfmoveClock = 99
	if got := DrawStatus(gs, nil); got != "" {
		t.Errorf("at 99 plies DrawStatus = %q, want no draw", got)
	}
	gs.HalfmoveClock = 100
	if got := DrawStatus(gs, nil); got != DrawByFiftyMove {
		t.Errorf("at 100 plies DrawStatus = %q, want %q", got, DrawByFiftyMove)
	}
}

// A recorded underpromotion must replay as the piece that was actually chosen.
func TestRecordedMoveRoundTripsPromotion(t *testing.T) {
	gs, err := ParseFEN("8/4P3/8/8/8/8/8/k6K w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	ns := ApplyMove(gs, mustUCI(t, gs, "e7e8n"))
	if ns.LastMove != "Pe7e8n" {
		t.Fatalf("LastMove = %q, want %q", ns.LastMove, "Pe7e8n")
	}

	parsed, ok := ParseRecordedMove(ns.LastMove)
	if !ok {
		t.Fatal("ParseRecordedMove failed")
	}
	if parsed.Promotion != "n" {
		t.Errorf("promotion = %q, want \"n\" (a queen here would replay a different game)", parsed.Promotion)
	}
	replayed := ApplyMove(gs, parsed)
	if PositionKey(replayed) != PositionKey(ns) {
		t.Error("replaying the recorded move produced a different position")
	}
}

// The search must score a position it can only repeat as a draw rather than as
// the material it is holding: this is the mechanism that stops a winning engine
// from shuffling into a threefold.
func TestSearchScoresForcedRepetitionAsDraw(t *testing.T) {
	// Bare kings: nothing but repetition is available, so the score must be 0.
	gs, err := ParseFEN("8/8/4k3/8/8/4K3/8/8 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	res := Search(gs, SearchOptions{MaxDepth: 4}, nil)
	if !res.HasBest {
		t.Fatal("expected a move")
	}
	if res.Score != 0 {
		t.Errorf("score = %d, want 0 for a dead-drawn position", res.Score)
	}
}

func mustUCI(t *testing.T, gs dao.GameState, uci string) dto.Move {
	t.Helper()
	m, err := ParseUCIMove(gs, uci)
	if err != nil {
		t.Fatalf("ParseUCIMove(%q): %v", uci, err)
	}
	return m
}
