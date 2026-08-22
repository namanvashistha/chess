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
	// History holds the Zobrist keys of positions that already occurred in the
	// game, oldest first, excluding the position being searched. Without it the
	// search cannot see a repetition that reaches back before the root, so an
	// engine with a winning position happily shuffles into a threefold draw.
	History []uint64
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

	// The root position itself is never scored as a draw -- a move still has to
	// be returned -- but it goes on the path so a child returning to it counts as
	// a repetition.
	rootPath := make([]uint64, 0, len(opts.History)+maxDepth+1)
	rootPath = append(rootPath, opts.History...)
	rootPath = append(rootPath, PositionKey(gs))

	for depth := 1; depth <= maxDepth; depth++ {
		c := &searchCtx{deadline: deadline, stop: opts.Stop, path: rootPath}
		best := -searchInf
		var bestMove dto.Move
		var bestPV []dto.Move
		alpha := -searchInf

		for _, m := range rootMoves {
			if stopped() {
				c.aborted = true
				break
			}
			var childPV []dto.Move
			score := -negamaxPV(c, applyBotMove(gs, m), depth-1, -searchInf, -alpha, &childPV)
			if c.aborted {
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

		if c.aborted {
			break // discard this incomplete depth, keep the previous result
		}

		result.Best = bestMove
		result.HasBest = true
		result.Score = best
		result.Depth = depth
		result.Nodes += c.nodes
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

// searchCtx is the mutable state threaded through a search: node count, the
// abort conditions, and the repetition path.
//
// It replaces a nine-parameter recursive signature, and it is what carries the
// position path needed for repetition detection.
type searchCtx struct {
	nodes    int
	deadline time.Time
	stop     <-chan struct{}
	aborted  bool
	// path holds the position keys from the start of the game down to the
	// current node. Pre-root game history is seeded from SearchOptions.History.
	path []uint64
}

// repeats reports whether key already appears on the current path.
//
// A single earlier occurrence is treated as a draw, not two. This is the
// standard convention: a side that can reach the same position once can reach it
// again, so the first repetition inside the tree already means the game is
// drawable, and scoring it as such is what makes the engine avoid it when
// winning and steer into it when losing.
func (c *searchCtx) repeats(key uint64) bool {
	for _, k := range c.path {
		if k == key {
			return true
		}
	}
	return false
}

// drawAtNode reports whether the node is a draw by repetition or the fifty-move
// rule.
//
// Caveat: this is checked before move generation, so a checkmate delivered on
// the exact ply the fifty-move counter expires is scored as a draw rather than a
// mate. Detecting that would cost a full move generation at every node; the
// position is vanishingly rare and every engine makes this trade.
func (c *searchCtx) drawAtNode(gs dao.GameState, key uint64) bool {
	return c.repeats(key) || gs.HalfmoveClock >= 100
}

// negamaxPV is negamax with node counting, principal-variation collection,
// repetition/fifty-move draw detection, and cooperative abort on time/Stop.
func negamaxPV(c *searchCtx, gs dao.GameState, depth, alpha, beta int, pv *[]dto.Move) int {
	c.nodes++

	// Check for time/Stop periodically to keep the overhead negligible.
	if c.nodes&1023 == 0 && abortRequested(c.deadline, c.stop) {
		c.aborted = true
		return 0
	}

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
	var bestChildPV []dto.Move
	for _, m := range moves {
		var childPV []dto.Move
		score := -negamaxPV(c, applyBotMove(gs, m), depth-1, -beta, -alpha, &childPV)
		if c.aborted {
			c.path = c.path[:len(c.path)-1]
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

	c.path = c.path[:len(c.path)-1]
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

// botMoveToDTO builds a dto.Move from a search move, deriving the piece letter
// from the source square. The promotion piece comes from the move itself, so an
// underpromotion the search chose survives the round trip through UCI instead of
// being rewritten as a queen.
func botMoveToDTO(gs dao.GameState, m botMove) dto.Move {
	return dto.Move{
		Piece:       getPieceCode(m.src, gs.WhiteBitboard&m.src != 0, gs),
		Source:      bitToSquare(m.src, defFiles, defRanks),
		Destination: bitToSquare(m.dst, defFiles, defRanks),
		Promotion:   m.promo,
	}
}

func isMateScore(s int) bool {
	if s < 0 {
		s = -s
	}
	return s > mateScore-maxSearchDepth*2
}
