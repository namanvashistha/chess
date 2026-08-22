package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ProcessMove(game *dao.ChessGame, move dto.Move, user dao.User) (string, error) {

	if user.ID != game.WhiteUser.ID && user.ID != game.BlackUser.ID {
		log.Errorf("Invalid move: user %d is not in the game", user.ID)
		return "", fmt.Errorf("invalid move: user %d is not in the game", user.ID)
	}

	if game.State.Turn == "w" && user.ID != game.WhiteUser.ID {
		log.Errorf("Invalid move: user %d is not white", user.ID)
		return "", fmt.Errorf("invalid move: user %d is not white", user.ID)
	} else if game.State.Turn == "b" && user.ID != game.BlackUser.ID {
		log.Errorf("Invalid move: user %d is not black", user.ID)
		return "", fmt.Errorf("invalid move: user %d is not black", user.ID)
	}

	if game.State.Turn == "w" && !(int(move.Piece[0]) >= 65 && int(move.Piece[0]) <= 90) {
		log.Errorf("Invalid move: user %d is not white", user.ID)
		return "", fmt.Errorf("invalid move: user %d is not white", user.ID)
	}
	if game.State.Turn == "b" && !(int(move.Piece[0]) >= 97 && int(move.Piece[0]) <= 122) {
		log.Errorf("Invalid move: user %d is not black", user.ID)
		return "", fmt.Errorf("invalid move: user %d is not black", user.ID)
	}

	sourceIdx := PositionToIndex(move.Source)
	destinationIdx := PositionToIndex(move.Destination)

	if !IsValidMove(*game, (1 << sourceIdx), (1 << destinationIdx)) {
		log.Errorf("Invalid move: move is not valid")
		return "", fmt.Errorf("invalid move: move is not valid")
	}

	game.State = ApplyMove(game.State, move)
	return game.State.LastMove, nil
}

// ApplyMove mutates a bare GameState by playing move, with no user/turn/legality
// validation (callers that need those checks should use ProcessMove). It returns
// the resulting state. Handles captures, castling (rook relocation + rights),
// en passant (set/clear + capture), and promotion. The promotion target honors
// move.Promotion ("q"|"r"|"b"|"n", case-insensitive); empty defaults to queen.
func ApplyMove(state dao.GameState, move dto.Move) dao.GameState {
	src := uint64(1) << uint(PositionToIndex(move.Source))
	dst := uint64(1) << uint(PositionToIndex(move.Destination))

	// The moving piece is taken from the board, not from move.Piece: a caller
	// that mislabels the piece can no longer move something else.
	ns := applyBitboardMove(state, src, dst, move.Promotion)
	// The promotion letter is part of the record: without it, replaying a game's
	// moves to detect repetition would turn every underpromotion into a queen and
	// reconstruct the wrong position.
	ns.LastMove = move.Piece + move.Source + move.Destination + strings.ToLower(move.Promotion)
	ns.Turn = ToggleTurn(state.Turn)
	return ns
}

func IsValidMove(game dao.ChessGame, piece uint64, destination uint64) bool {
	legalMoves, _ := GenerateLegalMovesForAllPositions(game.State)
	return legalMoves[piece]&destination != 0
}

func ToggleTurn(currentTurn string) string {
	if currentTurn == "w" {
		return "b"
	}
	return "w"
}
