package engine

import "testing"

// A beginner misses small things but still takes a queen left en prise.
func TestBeginnerTakesAFreeQueen(t *testing.T) {
	for i := 0; i < 20; i++ {
		g, err := NewGameFromFEN("4k3/8/8/3p4/4Q3/8/8/4K3 b - - 0 1")
		if err != nil {
			t.Fatal(err)
		}
		if m, ok := BotMove(g, LevelBeginner, nil); !ok || m != "d5e4" {
			t.Fatalf("beginner played %q, want d5e4", m)
		}
	}
}

func TestBeginnerPlaysLegalMoves(t *testing.T) {
	g := NewGame()
	for i := 0; i < 40; i++ {
		m, ok := BotMove(g, LevelBeginner, nil)
		if !ok {
			return
		}
		if _, err := g.Play(m); err != nil {
			t.Fatalf("move %d: %s: %v", i, m, err)
		}
	}
}
