package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"testing"
)

// perft counts leaf nodes to the given depth. It enumerates the side-to-move's
// legal moves via GenerateLegalMoveList (which expands promotions) and applies
// each with ApplyMove, so it exercises FEN parsing, move generation, promotion
// handling and move application together.
func perft(gs dao.GameState, depth int) int {
	if depth == 0 {
		return 1
	}
	nodes := 0
	for _, m := range GenerateLegalMoveList(gs) {
		move := dto.Move{
			Piece:       getPieceCode(m.Src, gs.WhiteBitboard&m.Src != 0, gs),
			Source:      bitToSquare(m.Src, defFiles, defRanks),
			Destination: bitToSquare(m.Dst, defFiles, defRanks),
			Promotion:   m.Promotion,
		}
		nodes += perft(ApplyMove(gs, move), depth-1)
	}
	return nodes
}

// The standard CPW perft suite. Reference counts are exact: a mismatch is a
// move-generation or move-application defect, never a test bug.
//
// This used to cover only the starting position to depth 3 -- the single
// configuration that passes -- so it could not fail. All five positions are now
// enforced at every depth.
func TestPerftSuite(t *testing.T) {
	cases := []struct {
		name string
		fen  string
		want []int // index i => depth i+1
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
			// Exposed illegal en-passant captures: the legality filter simulated
			// the move without removing the captured pawn, so it kept blocking
			// the 4th rank and hid the discovered check from Rb4.
			name: "position-3",
			fen:  "8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
			want: []int{14, 191, 2812, 43238},
		},
		{
			// Exposed missing underpromotions.
			name: "position-4",
			fen:  "r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1",
			want: []int{6, 264, 9467},
		},
		{
			name: "position-5",
			fen:  "rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 0 1",
			want: []int{44, 1486, 62379},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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
