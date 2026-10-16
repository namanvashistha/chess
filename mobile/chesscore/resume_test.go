package chesscore

import "testing"

// The iOS app calls these on the main thread while resuming a saved game and
// after each move. None of them may panic: a Go panic takes the app down.
func TestCallsTheAppMakes(t *testing.T) {
	lines := [][]string{
		{"e2e4", "d7d5"},
		{"d2d4", "d7d5", "e2e3", "g8f6"},
		{"e2e4", "b8c6", "f1a6", "b7a6"},
		{"f2f3", "e7e5", "g2g4", "d8h4"}, // ends in mate
	}
	for _, line := range lines {
		g := NewGame()
		for _, m := range line {
			if !g.Play(m) {
				t.Fatalf("Play(%s) refused", m)
			}
			_ = g.Board()
			_ = g.Turn()
			_ = g.Status()
			_ = g.InCheck()
			_ = g.LegalMoves()
			_ = g.Loose("w")
			_ = g.Loose("b")
			for _, sq := range []string{"e4", "d5", "a1", "zz", ""} {
				_ = g.Guards(sq)
				_ = g.Attackers(sq)
			}
		}
		_ = g.Eval(2)
		_ = g.Candidates(4)
		_ = g.BotMove("medium")
		for _, uci := range []string{"a2a3", "g1f3", "a7a6", "zz", ""} {
			_ = g.Describe(uci)
			_ = g.SAN(uci)
			_ = g.Explain(uci)
		}
		j := NewGameFromFEN(g.FEN())
		if j == nil {
			t.Fatalf("NewGameFromFEN(%q) = nil", g.FEN())
		}
		for _, uci := range []string{"a2a3", "a7a6", "zz", ""} {
			_ = j.Assess(uci)
		}
	}
}
