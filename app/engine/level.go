package engine

import (
	"math/rand/v2"
	"time"
)

// Bot difficulty levels.
const (
	LevelBeginner = "beginner"
	LevelEasy     = "easy"
	LevelMedium   = "medium"
	LevelHard     = "hard"
)

// ValidLevel reports whether level is one of the Level constants.
func ValidLevel(level string) bool {
	return level == LevelBeginner || level == LevelEasy || level == LevelMedium || level == LevelHard
}

// BotMove picks a move for the side to move at the given level, in UCI. It
// returns false when the game is over.
//
// Every level uses the real search. Easy and medium score each root move with a
// shallow search and then choose at random among the moves close to the best,
// so they make human-looking inaccuracies rather than blunders at random.
// Beginner does the same with a window wide enough to drop a minor piece or
// miss one left hanging, while still taking anything bigger, like a free
// queen. Hard plays the search's best move within a time budget.
func BotMove(g *Game, level string, table *TranspositionTable) (string, bool) {
	if result, _ := g.Outcome(); result != ResultNone {
		return "", false
	}
	switch level {
	case LevelBeginner:
		return pickNear(g, 1, 350), true
	case LevelEasy:
		return pickNear(g, 1, 150), true
	case LevelMedium:
		return pickNear(g, 3, 40), true
	default:
		r := Search(g.Position(), SearchOptions{
			MaxDepth: 64,
			MoveTime: time.Second,
			History:  g.History(),
			Table:    table,
		}, nil)
		if !r.HasBest {
			return g.legal[0], true
		}
		return MoveToUCI(r.Best), true
	}
}

// pickNear scores every legal move with a depth-limited search and returns a
// random one among those within window centipawns of the best.
func pickNear(g *Game, depth, window int) string {
	type scored struct {
		uci   string
		score int
	}
	gs := g.Position()
	history := append(g.History(), PositionKey(gs))
	moves := make([]scored, 0, len(g.legal))
	best := -searchInf
	for _, uci := range g.legal {
		m, err := ParseUCIMove(gs, uci)
		if err != nil {
			continue
		}
		child := ApplyMove(gs, m)
		r := Search(child, SearchOptions{MaxDepth: depth, History: history}, nil)
		score := -r.Score
		if !r.HasBest { // the move ends the game
			score = 0
			if InCheck(child) {
				score = mateScore
			}
		}
		moves = append(moves, scored{uci, score})
		best = max(best, score)
	}
	var near []string
	for _, m := range moves {
		if m.score >= best-window {
			near = append(near, m.uci)
		}
	}
	return near[rand.IntN(len(near))]
}
