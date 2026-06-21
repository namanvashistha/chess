package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"math/rand"
)

// Centipawn-ish piece values keyed by the lowercase FEN letter.
var botPieceValue = map[byte]int{
	'p': 1, 'n': 3, 'b': 3, 'r': 5, 'q': 9, 'k': 0,
}

// ChooseGreedyMove picks a move for the side to move using a 1-ply greedy
// heuristic: grab the most valuable hanging-or-not capture available (and
// reward promotions), breaking ties at random so the bot isn't deterministic.
// It does not look ahead, so it will happily trade into defended pieces; that's
// the intended strength level. Returns nil when there are no legal moves
// (checkmate or stalemate).
func ChooseGreedyMove(game *dao.ChessGame) *dto.Move {
	legal, _ := GenerateLegalMovesForAllPositions(game.State)
	if len(legal) == 0 {
		return nil
	}

	movesBySquare := ConvertLegalMovesToMap(legal)
	board := ConvertGameStateToMap(game.State)
	whiteToMove := game.State.Turn == "w"

	bestScore := -1
	var best []dto.Move

	for source, targets := range movesBySquare {
		piece := board[source]
		if piece == "" {
			continue
		}
		// GenerateLegalMovesForAllPositions returns moves for BOTH colours; only
		// consider the side to move, or the engine rejects the move as the wrong
		// colour.
		isWhitePiece := piece[0] >= 'A' && piece[0] <= 'Z'
		if isWhitePiece != whiteToMove {
			continue
		}
		for _, target := range targets {
			score := 0
			if captured := board[target]; captured != "" {
				score += 100 * botPieceValue[toLowerByte(captured[0])]
			}
			// Pawn reaching the last rank promotes (engine defaults to queen).
			if len(target) == 2 {
				if (piece == "P" && target[1] == '8') || (piece == "p" && target[1] == '1') {
					score += 100 * (botPieceValue['q'] - botPieceValue['p'])
				}
			}

			move := dto.Move{Piece: piece, Source: source, Destination: target}
			if score > bestScore {
				bestScore = score
				best = []dto.Move{move}
			} else if score == bestScore {
				best = append(best, move)
			}
		}
	}

	if len(best) == 0 {
		return nil
	}
	chosen := best[rand.Intn(len(best))]
	return &chosen
}

func toLowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
