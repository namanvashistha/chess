package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"strings"
)

// Draw detection by threefold repetition and the fifty-move rule.
//
// Neither existed before: GameState carried no move history and no halfmove
// clock, so a side with a winning position would shuffle pieces until the game
// was adjudicated a draw, and the engine had no way to know it had thrown the
// win away.

// fiftyMovePlies is the halfmove count at which the fifty-move rule applies:
// fifty moves by each side without a capture or a pawn move.
const fiftyMovePlies = 100

// Draw status values, returned by DrawStatus and reported alongside the
// checkmate/stalemate statuses from GenerateLegalMovesForAllPositions.
const (
	DrawByRepetition = "draw_repetition"
	DrawByFiftyMove  = "draw_fifty_move"
)

// RecordedMoves extracts the stored move strings from a game's move rows.
func RecordedMoves(moves []dao.GameMove) []string {
	out := make([]string, 0, len(moves))
	for _, m := range moves {
		out = append(out, m.Move)
	}
	return out
}

// ParseRecordedMove parses a stored LastMove/GameMove string, which is the piece
// letter, source square, destination square, and an optional promotion letter.
func ParseRecordedMove(s string) (dto.Move, bool) {
	if len(s) < 5 {
		return dto.Move{}, false
	}
	move := dto.Move{Piece: s[0:1], Source: s[1:3], Destination: s[3:5]}
	if len(s) > 5 {
		move.Promotion = strings.ToLower(s[5:6])
	}
	return move, true
}

// ReplayGameKeys replays a game's recorded moves from the standard starting
// position and returns the Zobrist key of every position that occurred: the
// start position first, then one per move.
//
// An unparseable move stops the replay and returns the keys accumulated so far.
// A short history can only fail to spot a repetition, never invent one.
func ReplayGameKeys(moves []string) []uint64 {
	gs := StartState()
	keys := make([]uint64, 0, len(moves)+1)
	keys = append(keys, PositionKey(gs))

	for _, raw := range moves {
		move, ok := ParseRecordedMove(raw)
		if !ok {
			return keys
		}
		gs = ApplyMove(gs, move)
		keys = append(keys, PositionKey(gs))
	}
	return keys
}

// DrawStatus reports whether the position reached at the end of keys is drawn.
// keys must end with the current position's key, as produced by ReplayGameKeys.
func DrawStatus(gs dao.GameState, keys []uint64) string {
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
