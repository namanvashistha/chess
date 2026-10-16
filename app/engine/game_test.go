package engine

import (
	"errors"
	"testing"
)

func playAll(t *testing.T, g *Game, moves ...string) []string {
	t.Helper()
	var sans []string
	for _, m := range moves {
		san, err := g.Play(m)
		if err != nil {
			t.Fatalf("Play(%s): %v", m, err)
		}
		sans = append(sans, san)
	}
	return sans
}

func TestSAN(t *testing.T) {
	cases := []struct {
		name  string
		fen   string
		moves []string
		want  []string
	}{
		{"opening", "", []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1b5"}, []string{"e4", "e5", "Nf3", "Nc6", "Bb5"}},
		{"pawn capture and check", "", []string{"e2e4", "d7d5", "e4d5", "d8d5", "b1c3", "d5e5"}, []string{"e4", "d5", "exd5", "Qxd5", "Nc3", "Qe5+"}},
		{"castling both ways", "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1", []string{"e1g1", "e8c8"}, []string{"O-O", "O-O-O"}},
		{"knights share a rank", "4k3/8/8/8/8/8/8/1N2KN2 w - - 0 1", []string{"b1d2"}, []string{"Nbd2"}},
		{"rooks share a file", "4k3/R7/8/8/8/8/8/R3K3 w - - 0 1", []string{"a1a4"}, []string{"R1a4"}},
		{"promotion with mate", "7k/4P3/6K1/8/8/8/8/8 w - - 0 1", []string{"e7e8q"}, []string{"e8=Q#"}},
		{"underpromotion capture", "3r3k/4P3/8/8/8/8/8/K7 w - - 0 1", []string{"e7d8n"}, []string{"exd8=N"}},
		{"en passant", "4k3/8/8/3pP3/8/8/8/4K3 w - d6 0 1", []string{"e5d6"}, []string{"exd6"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := NewGame()
			if c.fen != "" {
				var err error
				if g, err = NewGameFromFEN(c.fen); err != nil {
					t.Fatal(err)
				}
			}
			got := playAll(t, g, c.moves...)
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("move %d: SAN = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

func TestOutcome(t *testing.T) {
	g := NewGame()
	playAll(t, g, "f2f3", "e7e5", "g2g4", "d8h4")
	if r, why := g.Outcome(); r != ResultBlack || why != ReasonCheckmate {
		t.Errorf("fool's mate: Outcome = %q %q", r, why)
	}
	if _, err := g.Play("a2a3"); !errors.Is(err, ErrIllegalMove) {
		t.Errorf("a move after mate returned %v, want ErrIllegalMove", err)
	}

	stale, _ := NewGameFromFEN("7k/5Q2/6K1/8/8/8/8/8 b - - 0 1")
	if r, why := stale.Outcome(); r != ResultDraw || why != ReasonStalemate {
		t.Errorf("stalemate: Outcome = %q %q", r, why)
	}

	for _, fen := range []string{
		"8/8/4k3/8/8/3K4/8/8 w - - 0 1",    // bare kings
		"8/8/4k3/8/8/3KN3/8/8 w - - 0 1",   // king and knight
		"8/2b5/4k3/8/8/3KB3/8/8 w - - 0 1", // bishops on the same colour
	} {
		g, _ := NewGameFromFEN(fen)
		if r, why := g.Outcome(); r != ResultDraw || why != ReasonInsufficient {
			t.Errorf("%s: Outcome = %q %q, want insufficient material", fen, r, why)
		}
	}
	if g, _ := NewGameFromFEN("8/1b6/4k3/8/8/3KB3/8/8 w - - 0 1"); g != nil {
		if r, _ := g.Outcome(); r != ResultNone {
			t.Errorf("opposite-coloured bishops: Outcome = %q, want the game to go on", r)
		}
	}
}

func TestUndoRestoresPosition(t *testing.T) {
	g := NewGame()
	start := g.FEN()
	playAll(t, g, "e2e4", "c7c5")
	if !g.Undo() || !g.Undo() || g.Undo() {
		t.Fatal("Undo did not take back exactly two moves")
	}
	if g.FEN() != start || len(g.Legal()) != 20 || len(g.SANs()) != 0 {
		t.Errorf("after undo: FEN %q, %d legal moves, %d SANs", g.FEN(), len(g.Legal()), len(g.SANs()))
	}
}

func TestBotMoveIsLegalAtEveryLevel(t *testing.T) {
	for _, level := range []string{LevelEasy, LevelMedium, LevelHard} {
		g := NewGame()
		playAll(t, g, "e2e4", "e7e5")
		uci, ok := BotMove(g, level, NewTranspositionTable(4))
		if !ok || !g.IsLegal(uci) {
			t.Errorf("%s: BotMove = %q, %v", level, uci, ok)
		}
	}
}

func TestBotTakesMateInOne(t *testing.T) {
	// Every level must find a mate in one: Qh5 is not it, Qxf7# is.
	for _, level := range []string{LevelEasy, LevelMedium, LevelHard} {
		g := NewGame()
		playAll(t, g, "e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6")
		if uci, _ := BotMove(g, level, nil); uci != "h5f7" {
			t.Errorf("%s: BotMove = %q, want h5f7", level, uci)
		}
	}
}

func TestCoach(t *testing.T) {
	g := NewGame()
	a, err := g.Assess("e2e4")
	if err != nil || (a.Kind != KindGreat && a.Kind != KindGood) {
		t.Errorf("1. e4 assessed as %+v, %v", a, err)
	}
	playAll(t, g, "e2e4", "e7e5")
	a, _ = g.Assess("d1h5")
	if a.Loss < 0 || a.Title == "" || a.Text == "" {
		t.Errorf("Qh5 assessment = %+v", a)
	}
	g2, _ := NewGameFromFEN("4k3/8/8/3p4/8/8/8/3QK3 w - - 0 1")
	b, _ := g2.Assess("d1d5")
	if b.Kind != KindGreat || b.Text != "You win the pawn. Clean." {
		t.Errorf("Qxd5 = %+v", b)
	}
	g3, _ := NewGameFromFEN("4k3/8/8/4p3/8/8/8/3QK3 w - - 0 1")
	c, _ := g3.Assess("d1d4") // the queen walks into the pawn's capture
	if c.Kind != KindBlunder || c.Text != "Your queen on d4 can now be taken." {
		t.Errorf("Qd4 = %+v", c)
	}
	if loose := g3.Loose("w"); len(loose) != 0 {
		t.Errorf("Loose before the blunder = %v", loose)
	}
	if got := g.Candidates(3); len(got) != 3 || !g.IsLegal(got[0]) {
		t.Errorf("Candidates = %v", got)
	}
	if e := NewGame().Eval(2); e < -60 || e > 80 {
		t.Errorf("start eval = %d", e)
	}
}
