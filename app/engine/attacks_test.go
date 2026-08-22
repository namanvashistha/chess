package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"testing"
)

// isKingInCheckSlow is the original implementation: generate every pseudo-legal
// move for both colours and scan the result. It is kept solely as the reference
// that the fast single-square detector is checked against.
func isKingInCheckSlow(gs dao.GameState, isWhiteKing bool) bool {
	pseudo, _ := GenerateInitialMoves(gs)
	in, _ := CheckIfKingIsInCheck(gs, pseudo, isWhiteKing)
	return in
}

// isSquareAttacked replaced a full both-colour move generation per legality
// check. This walks every position reachable in four plies from CPW position 3
// -- chosen because it is the pin and en-passant heavy one -- and asserts the
// fast detector agrees with the exhaustive one for both colours at every node.
//
// It caught a real bug: the first version read WhitePawnAttackBitboard, which
// despite its name holds the forward push square rather than the diagonal
// captures, so a king was reported in check from a pawn standing directly in
// front of it.
func TestFastCheckDetectionMatchesExhaustive(t *testing.T) {
	gs := mustFEN(t, "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1")
	found := 0

	var walk func(gs dao.GameState, depth int, line string)
	walk = func(gs dao.GameState, depth int, line string) {
		if found > 2 {
			return
		}
		for _, white := range []bool{true, false} {
			fast := isKingInCheck(gs, white)
			slow := isKingInCheckSlow(gs, white)
			if fast != slow {
				found++
				t.Errorf("MISMATCH white=%v fast=%v slow=%v\n  fen:  %s\n  line: %s",
					white, fast, slow, ToFEN(gs), line)
				return
			}
		}
		if depth == 0 {
			return
		}
		for _, m := range GenerateLegalMoveList(gs) {
			mv := dto.Move{
				Piece:       getPieceCode(m.Src, gs.WhiteBitboard&m.Src != 0, gs),
				Source:      bitToSquare(m.Src, defFiles, defRanks),
				Destination: bitToSquare(m.Dst, defFiles, defRanks),
				Promotion:   m.Promotion,
			}
			walk(ApplyMove(gs, mv), depth-1, line+" "+mv.Source+mv.Destination+mv.Promotion)
		}
	}
	walk(gs, 4, "")
	if found == 0 {
		t.Log("fast and exhaustive check detection agree on every reachable position")
	}
}

// The search uses appendLegalMoves (flat slice, no maps) while the server and
// UI still use GenerateLegalMovesForAllPositions (grouped by square). Perft
// validates the first; this asserts the second agrees with it, so the web game
// and the engine can never diverge on what is legal.
func TestGeneratorsAgree(t *testing.T) {
	fens := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 0 1",
		"8/2p5/3p4/KP5r/1R3pPk/8/4P3/8 b - g3 0 1", // en passant available
		"r3k2r/8/8/8/8/8/8/R3K2R b KQkq - 0 1",     // castling both sides
	}

	for _, fen := range fens {
		gs, err := ParseFEN(fen)
		if err != nil {
			t.Fatalf("ParseFEN(%q): %v", fen, err)
		}

		fast := map[string]bool{}
		for _, m := range appendLegalMoves(gs, nil) {
			fast[bitToSquare(m.src, defFiles, defRanks)+bitToSquare(m.dst, defFiles, defRanks)] = true
		}

		grouped, _ := GenerateLegalMovesForAllPositions(gs)
		slow := map[string]bool{}
		for src, dsts := range FilterMovesByTurn(grouped, gs) {
			for d := dsts; d != 0; d &= d - 1 {
				slow[bitToSquare(src, defFiles, defRanks)+bitToSquare(d&-d, defFiles, defRanks)] = true
			}
		}

		for mv := range fast {
			if !slow[mv] {
				t.Errorf("%s: fast generator has %s, map generator does not", fen, mv)
			}
		}
		for mv := range slow {
			if !fast[mv] {
				t.Errorf("%s: map generator has %s, fast generator does not", fen, mv)
			}
		}
	}
}
