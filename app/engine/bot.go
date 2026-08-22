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

// botMove is a search-level move. promo carries the promotion piece so the
// search can consider underpromotion and so its material evaluation sees the
// piece that actually appears on the board.
type botMove struct {
	src, dst uint64
	promo    string
}

// applyBotMove plays a search move. It goes through the same applier as the
// server, so the search now sees castling, en passant and promotion instead of
// silently playing a different game.
func applyBotMove(gs dao.GameState, m botMove) dao.GameState {
	ns := applyBitboardMove(gs, m.src, m.dst, m.promo)
	ns.Turn = oppTurn(gs.Turn)
	return ns
}

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

	bestScore := -1
	var best []botMove
	for _, m := range moves {
		score := pieceValueAt(gs, m.dst) // captured value (0 if quiet)
		if m.promo != "" {
			score += botPieceValue[m.promo[0]] - botPieceValue['p']
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

	// Seed the repetition path from the moves already played, so the bot does not
	// shuffle a winning position into a threefold draw.
	path := append(SearchHistory(ReplayGameKeys(RecordedMoves(game.Moves))), PositionKey(gs))
	c := &searchCtx{path: path}

	bestScore := -searchInf
	var best []botMove
	for _, m := range moves {
		// Full window at the root so tied-best moves are collected correctly.
		score := -negamax(c, applyBotMove(gs, m), depth-1, -searchInf, searchInf)
		if score > bestScore {
			bestScore = score
			best = []botMove{m}
		} else if score == bestScore {
			best = append(best, m)
		}
	}
	return buildMove(gs, best[rand.Intn(len(best))])
}

// negamax returns the value of gs from the side-to-move's perspective. It shares
// searchCtx with the UCI search so both see repetitions and the fifty-move rule.
func negamax(c *searchCtx, gs dao.GameState, depth, alpha, beta int) int {
	key := PositionKey(gs)
	if c.drawAtNode(gs, key) {
		return 0
	}

	if depth == 0 {
		return quiescence(c, gs, alpha, beta, maxQuiescenceDepth)
	}

	moves := sideToMoveMovesOrdered(gs)
	if len(moves) == 0 {
		if isKingInCheck(gs, gs.Turn == "w") {
			return -mateScore - depth // prefer faster mates
		}
		return 0 // stalemate
	}

	c.path = append(c.path, key)

	best := -searchInf
	for _, m := range moves {
		score := -negamax(c, applyBotMove(gs, m), depth-1, -beta, -alpha)
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

	c.path = c.path[:len(c.path)-1]
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

// sideToMoveMoves lists the legal moves for the side to move, with promotions
// expanded into all four pieces.
//
// The order is deterministic without sorting: appendLegalMoves walks the
// bitboards least-significant-bit first in a fixed piece order, unlike the
// map-based generator whose iteration order Go randomises.
func sideToMoveMoves(gs dao.GameState) []botMove {
	return appendLegalMoves(gs, make([]botMove, 0, 48))
}

// sideToMoveMovesOrdered orders captures first, most valuable victim captured by
// least valuable attacker (MVV-LVA), so alpha-beta prunes more.
//
// SliceStable, not Slice: an unstable sort over equally-scored moves reorders
// them unpredictably, which reintroduces the non-determinism that the ordered
// generator exists to remove.
func sideToMoveMovesOrdered(gs dao.GameState) []botMove {
	moves := sideToMoveMoves(gs)
	sort.SliceStable(moves, func(i, j int) bool {
		return captureScore(gs, moves[i]) > captureScore(gs, moves[j])
	})
	return moves
}

// captureScore ranks a move for ordering. Winning captures first (queen taken by
// a pawn beats queen taken by a queen), then promotions, then quiet moves.
func captureScore(gs dao.GameState, m botMove) int {
	score := 0
	if victim := pieceValueAt(gs, m.dst); victim != 0 {
		// Victim dominates; the attacker only breaks ties, hence the factor.
		score = 100000 + victim*16 - pieceValueAt(gs, m.src)
	}
	if m.promo != "" {
		score += 90000 + botPieceValue[m.promo[0]]
	}
	return score
}

// promoteOrderedMoves moves the transposition-table suggestion to the front,
// followed by the killers, preserving the existing MVV-LVA order otherwise.
//
// The table move is the best move found for this exact position at a shallower
// depth, so it is overwhelmingly likely to be best again. Searching it first is
// most of what makes iterative deepening pay for itself.
func promoteOrderedMoves(moves []botMove, ttMove botMove, hasTT bool, killers [2]botMove) {
	front := 0
	promote := func(target botMove) {
		for i := front; i < len(moves); i++ {
			if moves[i] == target {
				moves[front], moves[i] = moves[i], moves[front]
				front++
				return
			}
		}
	}
	if hasTT {
		promote(ttMove)
	}
	for _, k := range killers {
		if k.src != 0 {
			promote(k)
		}
	}
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
	move := botMoveToDTO(gs, m)
	if move.Piece == "" {
		return nil
	}
	return &move
}

func oppTurn(turn string) string {
	if turn == "w" {
		return "b"
	}
	return "w"
}
