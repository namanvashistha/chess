package engine

// Draw detection by threefold repetition and the fifty-move rule.
//
// Neither existed before: GameState carried no move history and no halfmove
// clock, so a side with a winning position would shuffle pieces until the game
// was adjudicated a draw, and the engine had no way to know it had thrown the
// win away.

// fiftyMovePlies is the halfmove count at which the fifty-move rule applies:
// fifty moves by each side without a capture or a pawn move.
const fiftyMovePlies = 100

// Draw status values returned by DrawStatus. They double as Game outcome
// reasons.
const (
	DrawByRepetition = ReasonRepetition
	DrawByFiftyMove  = ReasonFiftyMove
)

// DrawStatus reports whether the position reached at the end of keys is drawn.
// keys must end with the current position's key.
func DrawStatus(gs GameState, keys []uint64) string {
	if len(keys) > 0 {
		current := keys[len(keys)-1]
		seen := 0
		for _, k := range keys {
			if k == current {
				seen++
			}
		}
		if seen >= 3 {
			return DrawByRepetition
		}
	}
	if gs.HalfmoveClock >= fiftyMovePlies {
		return DrawByFiftyMove
	}
	return ""
}

// SearchHistory converts a full key list into the value for
// SearchOptions.History: every position except the last, which is the position
// the search will start from.
func SearchHistory(keys []uint64) []uint64 {
	if len(keys) == 0 {
		return nil
	}
	return keys[:len(keys)-1]
}
