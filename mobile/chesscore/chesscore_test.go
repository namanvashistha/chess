package chesscore

import (
	"strings"
	"testing"
)

func TestStartPosition(t *testing.T) {
	g := NewGame()
	want := "rnbqkbnrpppppppp................................PPPPPPPPRNBQKBNR"
	if got := g.Board(); got != want {
		t.Errorf("Board() = %q, want %q", got, want)
	}
	if n := len(strings.Fields(g.LegalMoves())); n != 20 {
		t.Errorf("start position has %d legal moves, want 20", n)
	}
	if g.Turn() != "w" || g.Status() != StatusPlaying {
		t.Errorf("Turn() = %q, Status() = %q", g.Turn(), g.Status())
	}
}

func TestPlayAndUndo(t *testing.T) {
	g := NewGame()
	for _, m := range []string{"e2e4", "e7e5", "g1f3"} {
		if err := g.Play(m); err != nil {
			t.Fatalf("Play(%s): %v", m, err)
		}
	}
	if err := g.Play("e1e3"); err == nil {
		t.Error("Play(e1e3) succeeded, want an illegal-move error")
	}
	if g.Moves() != "e2e4 e7e5 g1f3" || g.Turn() != "b" {
		t.Errorf("Moves() = %q, Turn() = %q", g.Moves(), g.Turn())
	}
	if !g.Undo() || g.MoveCount() != 2 || g.Turn() != "w" {
		t.Errorf("after Undo: MoveCount() = %d, Turn() = %q", g.MoveCount(), g.Turn())
	}
}

func TestFoolsMate(t *testing.T) {
	g := NewGame()
	for _, m := range []string{"f2f3", "e7e5", "g2g4", "d8h4"} {
		if err := g.Play(m); err != nil {
			t.Fatalf("Play(%s): %v", m, err)
		}
	}
	if !g.InCheck() || g.Status() != StatusCheckmate {
		t.Errorf("InCheck() = %v, Status() = %q, want checkmate", g.InCheck(), g.Status())
	}
	if r := g.Search(3, 0); r.BestMove != "" {
		t.Errorf("Search on a mated position returned %q", r.BestMove)
	}
}

func TestSearchFindsMateForBlack(t *testing.T) {
	g := NewGame()
	for _, m := range []string{"f2f3", "e7e5", "g2g4"} {
		if err := g.Play(m); err != nil {
			t.Fatal(err)
		}
	}
	r := g.Search(3, 0)
	if r.BestMove != "d8h4" {
		t.Errorf("BestMove = %q, want d8h4", r.BestMove)
	}
	// Black mates, so the score is reported from White's side as negative.
	if r.Mate >= 0 {
		t.Errorf("Mate = %d, want negative (Black mates)", r.Mate)
	}
}
