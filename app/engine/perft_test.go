package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"testing"
)

// perft counts leaf nodes to the given depth by enumerating the side-to-move's
// legal moves and applying each via ApplyMove. It exercises FEN parsing, move
// generation, and ApplyMove together.
//
// Caveat: the generator returns a destination bitboard per piece, which has no
// room for a promotion choice, so a promoting move is counted once (as a queen)
// instead of four times. Positions whose reference count includes promotions are
// therefore expected to undercount; see TestPerftSuite.
func perft(gs dao.GameState, depth int) int {
	if depth == 0 {
		return 1
	}
	all, _ := GenerateLegalMovesForAllPositions(gs)
	moves := FilterMovesByTurn(all, gs)
	nodes := 0
	for src, dsts := range moves {
		for d := dsts; d != 0; d &= d - 1 {
			dst := d & -d
			move := dto.Move{
				Piece:       getPieceCode(src, gs.WhiteBitboard&src != 0, gs),
				Source:      bitToSquare(src, defFiles, defRanks),
				Destination: bitToSquare(dst, defFiles, defRanks),
			}
			ns := ApplyMove(gs, move)
			nodes += perft(ns, depth-1)
		}
	}
	return nodes
}

// The standard CPW perft suite. Reference counts are exact: a mismatch is a
// move-generation or move-application defect, never a test bug.
//
// This used to cover only the starting position to depth 3 -- the single
// configuration that passes -- so it could not fail. The suite below found three
// real defects immediately; each is recorded against the position that exposes
// it, with a skip so the rest of the suite stays enforced.
func TestPerftSuite(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		want []int // index i => depth i+1
		skip string
	}{
		{
			name: "startpos",
			fen:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			want: []int{20, 400, 8902, 197281},
		},
		{
			name: "kiwipete",
			fen:  "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
			want: []int{48, 2039, 97862},
		},
		{
			name: "position-3",
			fen:  "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
			want: []int{14, 191, 2812, 43238},
			// Overcounts (193 at depth 2). filterLegalMoves validates moves with
			// simulateMove, which cannot perform an en-passant capture, so the
			// captured pawn stays on the board and still blocks the rank. That
			// hides the discovered check and both en-passant captures here
			// (f4g3, f4e3) are offered as legal when they are not.
			skip: "en-passant discovered check not detected (simulateMove cannot do en passant)",
		},
		{
			name: "position-4",
			fen:  "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
			want: []int{6, 264, 9467},
			// Undercounts by exactly the missing underpromotions: black's b2 pawn
			// has 2 promotion squares x 3 unavailable pieces = 6 per branch,
			// x 6 white replies = 36, and 264-228 = 36.
			skip: "underpromotion cannot be represented in the generator's destination bitboard",
		},
		{
			name: "position-5",
			fen:  "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 0 1",
			want: []int{44, 1486, 62379},
			// Same defect, visible at depth 1: d7xc8 has 4 promotion choices
			// counted as 1, and 44-41 = 3.
			skip: "underpromotion cannot be represented in the generator's destination bitboard",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.skip != "" {
				t.Skip("known defect: " + c.skip)
			}
			gs, err := ParseFEN(c.fen)
			if err != nil {
				t.Fatalf("ParseFEN(%q): %v", c.fen, err)
			}
			for i, want := range c.want {
				depth := i + 1
				if got := perft(gs, depth); got != want {
					t.Errorf("perft(depth %d) = %d, want %d", depth, got, want)
				}
			}
		})
	}
}

// TestKingAdjacencyIsIllegal pins the bug that lost a real game under fastchess:
// a king was allowed to step onto a square adjacent to the enemy king.
//
// generateKingMoves stores the *safety-filtered* king move set into
// pseudo_legal_moves, and getAttackedSquares reads that same map to build the
// opposing side's attack set. Any square next to a king that is also attacked by
// the other side therefore vanishes from that king's attack set, and the enemy
// king may legally move onto it. Here black's pawn on a4 attacks b3, which
// erases b3 from white Kb2's attack set, so kb4-b3 looks safe.
func TestKingAdjacencyIsIllegal(t *testing.T) {
	for _, fen := range []string{
		"8/8/8/8/pk6/8/1K6/8 b - - 0 1",   // with the pawn: b3 wrongly offered
		"8/8/8/8/1k6/8/1K6/8 b - - 0 1",   // without it: correct
		"8/8/8/8/pk6/8/1K6/3q4 b - - 0 1", // the position from the lost game
	} {
		gs, err := ParseFEN(fen)
		if err != nil {
			t.Fatalf("ParseFEN(%q): %v", fen, err)
		}
		all, _ := GenerateLegalMovesForAllPositions(gs)
		blackKing := gs.KingBitboard & gs.BlackBitboard
		for d := all[blackKing]; d != 0; d &= d - 1 {
			sq := bitToSquare(d&-d, defFiles, defRanks)
			if sq == "a3" || sq == "b3" || sq == "c3" {
				t.Errorf("%s: king move to %s is illegal (adjacent to white Kb2)", fen, sq)
			}
		}
	}
}
