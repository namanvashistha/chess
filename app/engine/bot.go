package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"math/bits"
	"math/rand"
	"sort"
)

// Centipawn piece values keyed by the lowercase FEN letter (used by the greedy
// chooser); the search uses the same scale via pieceValueAt/evalMaterial.
var botPieceValue = map[byte]int{
	'p': 100, 'n': 320, 'b': 330, 'r': 500, 'q': 900, 'k': 0,
}

const (
	searchInf   = 1 << 30
	mateScore   = 1 << 20
	mediumDepth = 2
	hardDepth   = 4
)

var defFiles = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
var defRanks = []string{"1", "2", "3", "4", "5", "6", "7", "8"}

type botMove struct{ src, dst uint64 }

// ChooseBotMove dispatches to a move-selection strategy based on the game's
// stored difficulty. Returns nil when there are no legal moves.
//
//	easy   - greedy 1-ply (grabs material, hangs pieces)
//	medium - shallow alpha-beta search (won't hang to an immediate recapture)
//	hard   - deeper alpha-beta search
func ChooseBotMove(game *dao.ChessGame) *dto.Move {
	switch game.BotLevel {
	case "medium":
		return ChooseSearchMove(game, mediumDepth)
	case "hard":
		return ChooseSearchMove(game, hardDepth)
	default: // "easy" and anything unset
		return ChooseGreedyMove(game)
	}
}

// ChooseGreedyMove picks the move that grabs the most material right now
// (rewarding promotions), random tiebreak. 1-ply, no lookahead, so it will
// happily trade into defended pieces. Returns nil when there are no legal moves.
func ChooseGreedyMove(game *dao.ChessGame) *dto.Move {
	gs := game.State
	moves := sideToMoveMoves(gs)
	if len(moves) == 0 {
		return nil
	}
	board := ConvertGameStateToMap(gs)

	bestScore := -1
	var best []botMove
	for _, m := range moves {
		score := pieceValueAt(gs, m.dst) // captured value (0 if quiet)
		// Pawn reaching the last rank promotes (engine defaults to queen).
		src := bitToSquare(m.src, defFiles, defRanks)
		dst := bitToSquare(m.dst, defFiles, defRanks)
		piece := board[src]
		if (piece == "P" && len(dst) == 2 && dst[1] == '8') ||
			(piece == "p" && len(dst) == 2 && dst[1] == '1') {
			score += botPieceValue['q'] - botPieceValue['p']
		}
		if score > bestScore {
			bestScore = score
			best = []botMove{m}
		} else if score == bestScore {
			best = append(best, m)
		}
	}
	return buildMove(gs, best[rand.Intn(len(best))])
}

// ChooseSearchMove runs an alpha-beta material search to the given depth. It
// sees recaptures, so unlike the greedy bot it won't hang pieces by trading
// into a defended square. Returns nil when there are no legal moves.
func ChooseSearchMove(game *dao.ChessGame, depth int) *dto.Move {
	gs := game.State
	moves := sideToMoveMovesOrdered(gs)
	if len(moves) == 0 {
		return nil
	}

	bestScore := -searchInf
	var best []botMove
	for _, m := range moves {
		ns := simulateMove(gs, m.src, m.dst)
		ns.Turn = oppTurn(gs.Turn)
		// Full window at the root so tied-best moves are collected correctly.
		score := -negamax(ns, depth-1, -searchInf, searchInf)
		if score > bestScore {
			bestScore = score
			best = []botMove{m}
		} else if score == bestScore {
			best = append(best, m)
		}
	}
	return buildMove(gs, best[rand.Intn(len(best))])
}

// negamax returns the value of gs from the side-to-move's perspective.
func negamax(gs dao.GameState, depth, alpha, beta int) int {
	if depth == 0 {
		score := evalMaterial(gs)
		if gs.Turn == "b" {
			score = -score
		}
		return score
	}

	moves := sideToMoveMovesOrdered(gs)
	if len(moves) == 0 {
		if isKingInCheck(gs, gs.Turn == "w") {
			return -mateScore - depth // prefer slower mates / faster wins
		}
		return 0 // stalemate
	}

	best := -searchInf
	for _, m := range moves {
		ns := simulateMove(gs, m.src, m.dst)
		ns.Turn = oppTurn(gs.Turn)
		score := -negamax(ns, depth-1, -beta, -alpha)
		if score > best {
			best = score
		}
		if best > alpha {
			alpha = best
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

// evalMaterial scores the position in centipawns from White's perspective.
func evalMaterial(gs dao.GameState) int {
	w := 100*bits.OnesCount64(gs.PawnBitboard&gs.WhiteBitboard) +
		320*bits.OnesCount64(gs.KnightBitboard&gs.WhiteBitboard) +
		330*bits.OnesCount64(gs.BishopBitboard&gs.WhiteBitboard) +
		500*bits.OnesCount64(gs.RookBitboard&gs.WhiteBitboard) +
		900*bits.OnesCount64(gs.QueenBitboard&gs.WhiteBitboard)
	b := 100*bits.OnesCount64(gs.PawnBitboard&gs.BlackBitboard) +
		320*bits.OnesCount64(gs.KnightBitboard&gs.BlackBitboard) +
		330*bits.OnesCount64(gs.BishopBitboard&gs.BlackBitboard) +
		500*bits.OnesCount64(gs.RookBitboard&gs.BlackBitboard) +
		900*bits.OnesCount64(gs.QueenBitboard&gs.BlackBitboard)
	return w - b
}

// sideToMoveMoves expands the legal moves for the side to move into a flat list.
// GenerateLegalMovesForAllPositions returns moves for both colours, so we filter
// to the side to move by piece colour.
func sideToMoveMoves(gs dao.GameState) []botMove {
	all, _ := GenerateLegalMovesForAllPositions(gs)
	colorBB := gs.WhiteBitboard
	if gs.Turn == "b" {
		colorBB = gs.BlackBitboard
	}
	var out []botMove
	for src, dsts := range all {
		if src&colorBB == 0 {
			continue
		}
		for d := dsts; d != 0; d &= d - 1 {
			out = append(out, botMove{src: src, dst: d & -d})
		}
	}
	return out
}

// sideToMoveMovesOrdered orders captures first (most valuable victim first) so
// alpha-beta prunes more.
func sideToMoveMovesOrdered(gs dao.GameState) []botMove {
	moves := sideToMoveMoves(gs)
	sort.Slice(moves, func(i, j int) bool {
		return pieceValueAt(gs, moves[i].dst) > pieceValueAt(gs, moves[j].dst)
	})
	return moves
}

func pieceValueAt(gs dao.GameState, bit uint64) int {
	switch {
	case gs.PawnBitboard&bit != 0:
		return 100
	case gs.KnightBitboard&bit != 0:
		return 320
	case gs.BishopBitboard&bit != 0:
		return 330
	case gs.RookBitboard&bit != 0:
		return 500
	case gs.QueenBitboard&bit != 0:
		return 900
	}
	return 0
}

func buildMove(gs dao.GameState, m botMove) *dto.Move {
	source := bitToSquare(m.src, defFiles, defRanks)
	dest := bitToSquare(m.dst, defFiles, defRanks)
	piece := ConvertGameStateToMap(gs)[source]
	if piece == "" {
		return nil
	}
	return &dto.Move{Piece: piece, Source: source, Destination: dest}
}

func oppTurn(turn string) string {
	if turn == "w" {
		return "b"
	}
	return "w"
}
