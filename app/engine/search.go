package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"time"
)

// SearchOptions controls a UCI search. A search runs iterative deepening and
// stops at the first of: MaxDepth reached, MoveTime elapsed, or Stop signalled.
type SearchOptions struct {
	MaxDepth int           // hard depth cap; <= 0 means rely on time/Stop (capped at 64)
	MoveTime time.Duration // wall-clock budget; 0 means no time limit
	Infinite bool          // search until Stop regardless of MaxDepth/MoveTime
	Stop     <-chan struct{}
}

// SearchResult is the outcome of (a completed iteration of) a search.
type SearchResult struct {
	Best    dto.Move   // best move found; valid only when HasBest is true
	HasBest bool       // false when the side to move has no legal moves
	Score   int        // centipawns from the side-to-move's perspective
	Mate    int        // mate distance in moves (+ we mate, - we are mated); 0 if not a mate
	Depth   int        // depth of the last completed iteration
	Nodes   int        // cumulative nodes searched
	PV      []dto.Move // principal variation, starting with Best
	Elapsed time.Duration
}

const maxSearchDepth = 64

// Search runs an iterative-deepening alpha-beta search and returns the result of
// the deepest fully completed iteration. If info is non-nil it is called once per
// completed depth, enabling streaming UCI "info" lines. A partially searched
// depth (cut short by time/Stop) is discarded.
func Search(gs dao.GameState, opts SearchOptions, info func(SearchResult)) SearchResult {
	start := time.Now()

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 || maxDepth > maxSearchDepth {
		maxDepth = maxSearchDepth
	}
	if opts.Infinite {
		maxDepth = maxSearchDepth
	}

	var deadline time.Time
	if opts.MoveTime > 0 && !opts.Infinite {
		deadline = start.Add(opts.MoveTime)
	}

	stopped := func() bool {
		if opts.Stop != nil {
			select {
			case <-opts.Stop:
				return true
			default:
			}
		}
		return !deadline.IsZero() && time.Now().After(deadline)
	}

	var result SearchResult
	rootMoves := sideToMoveMovesOrdered(gs)
	if len(rootMoves) == 0 {
		return result // checkmate or stalemate: no move to make
	}
	// Guarantee a legal move even if the very first iteration is interrupted.
	result.Best = botMoveToDTO(gs, rootMoves[0])
	result.HasBest = true

	for depth := 1; depth <= maxDepth; depth++ {
		var nodes int
		aborted := false
		best := -searchInf
		var bestMove dto.Move
		var bestPV []dto.Move
		alpha := -searchInf

		for _, m := range rootMoves {
			if stopped() {
				aborted = true
				break
			}
			ns := simulateMove(gs, m.src, m.dst)
			ns.Turn = oppTurn(gs.Turn)
			var childPV []dto.Move
			score := -negamaxPV(ns, depth-1, -searchInf, -alpha, &nodes, &childPV, deadline, opts.Stop, &aborted)
			if aborted {
				break
			}
			if score > best {
				best = score
				bestMove = botMoveToDTO(gs, m)
				bestPV = append([]dto.Move{bestMove}, childPV...)
				if best > alpha {
					alpha = best
				}
			}
		}

		if aborted {
			break // discard this incomplete depth, keep the previous result
		}

		result.Best = bestMove
		result.HasBest = true
		result.Score = best
		result.Depth = depth
		result.Nodes += nodes
		result.PV = bestPV
		result.Elapsed = time.Since(start)
		if isMateScore(best) {
			mateMoves := (len(bestPV) + 1) / 2
			if best < 0 {
				mateMoves = -mateMoves
			}
			result.Mate = mateMoves
		} else {
			result.Mate = 0
		}

		if info != nil {
			info(result)
		}

		if result.Mate != 0 || stopped() {
			break // solved, or out of time
		}
	}

	return result
}

// negamaxPV is negamax with node counting, principal-variation collection, and
// cooperative abort on time/Stop. It mirrors negamax (simulateMove + material
// eval) so move selection matches the existing bot.
func negamaxPV(gs dao.GameState, depth, alpha, beta int, nodes *int, pv *[]dto.Move, deadline time.Time, stop <-chan struct{}, aborted *bool) int {
	*nodes++

	// Check for time/Stop periodically to keep the overhead negligible.
	if *nodes&1023 == 0 && abortRequested(deadline, stop) {
		*aborted = true
		return 0
	}

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
			return -mateScore - depth // prefer faster mates
		}
		return 0 // stalemate
	}

	best := -searchInf
	var bestChildPV []dto.Move
	for _, m := range moves {
		ns := simulateMove(gs, m.src, m.dst)
		ns.Turn = oppTurn(gs.Turn)
		var childPV []dto.Move
		score := -negamaxPV(ns, depth-1, -beta, -alpha, nodes, &childPV, deadline, stop, aborted)
		if *aborted {
			return best
		}
		if score > best {
			best = score
			mv := botMoveToDTO(gs, m)
			bestChildPV = append([]dto.Move{mv}, childPV...)
		}
		if best > alpha {
			alpha = best
		}
		if alpha >= beta {
			break
		}
	}
	*pv = bestChildPV
	return best
}

func abortRequested(deadline time.Time, stop <-chan struct{}) bool {
	if stop != nil {
		select {
		case <-stop:
			return true
		default:
		}
	}
	return !deadline.IsZero() && time.Now().After(deadline)
}

// botMoveToDTO builds a dto.Move from a bitboard move, deriving the piece letter
// from the source square. Promoting pawn moves default to a queen (matching the
// search), with the promotion letter set so the move round-trips through UCI.
func botMoveToDTO(gs dao.GameState, m botMove) dto.Move {
	src := bitToSquare(m.src, defFiles, defRanks)
	dst := bitToSquare(m.dst, defFiles, defRanks)
	piece := getPieceCode(m.src, gs.WhiteBitboard&m.src != 0, gs)
	move := dto.Move{Piece: piece, Source: src, Destination: dst}
	if (piece == "P" && len(dst) == 2 && dst[1] == '8') ||
		(piece == "p" && len(dst) == 2 && dst[1] == '1') {
		move.Promotion = "q"
	}
	return move
}

func isMateScore(s int) bool {
	if s < 0 {
		s = -s
	}
	return s > mateScore-maxSearchDepth*2
}
