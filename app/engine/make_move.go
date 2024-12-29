package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// MakeMove handles the movement of pieces on the chessboard and validates the move.
func MakeMove(game *dao.ChessGame, move dto.Move, user dao.User) error {
	var board map[string]string
	if err := json.Unmarshal(game.ChessState.Board, &board); err != nil {
		log.Errorf("Failed to unmarshal board: %v", err)
		return fmt.Errorf("failed to unmarshal board: %w", err)
	}
	sourcePiece := move.Piece
	// destinationPiece := board[move.DestinationSquare]

	// Log the move
	log.Infof("Attempting to move %s from %s to %s", sourcePiece, move.Source, move.Destination)
	log.Info(board)
	if sourcePiece != board[move.Source] {
		log.Error("Invalid move source piece not matching. please refresh Error")
		return fmt.Errorf("invalid move source piece not matching")
	}

	// Validate source square
	if sourcePiece == "---" {
		log.Errorf("Invalid move: no piece at source square %s", move.Source)
		return fmt.Errorf("invalid move: no piece at source square %s", move.Source)
	}

	if user.ID != game.WhiteUser.ID && user.ID != game.BlackUser.ID {
		log.Errorf("Invalid move: user %d is not in the game", user.ID)
		return fmt.Errorf("invalid move: user %d is not in the game", user.ID)
	}

	if game.ChessState.Turn == "white" && user.ID != game.WhiteUser.ID {
		log.Errorf("Invalid move: user %d is not white player", user.ID)
		return fmt.Errorf("invalid move: user %d is not white player", user.ID)
	}

	if game.ChessState.Turn == "black" && user.ID != game.BlackUser.ID {
		log.Errorf("Invalid move: user %d is not black player", user.ID)
		return fmt.Errorf("invalid move: user %d is not black player", user.ID)
	}

	// Check if it's the correct player's turn
	currentTurn := game.ChessState.Turn
	pieceColor := string(sourcePiece[0:1]) // Assume the first character indicates color (e.g., 'w' or 'b')
	if (currentTurn == "white" && pieceColor != "w") || (currentTurn == "black" && pieceColor != "b") {
		log.Errorf("Invalid move: it's %s's turn", currentTurn)
		return fmt.Errorf("invalid move: it's %s's turn", currentTurn)
	}

	// Validate move legality
	allowedMoves := GetAllowedMoves(*game)
	if !contains(allowedMoves[move.Piece], move.Destination) {
		log.Errorf("Invalid move: %s to %s is not allowed", move.Source, move.Destination)
		return fmt.Errorf("invalid move: %s to %s is not allowed", move.Source, move.Destination)
	}

	// Handle special moves (e.g., promotion, castling, en passant)
	// if isPawnPromotion(sourcePiece, move.DestinationSquare) {
	// 	game.ChessState.Board[move.DestinationSquare] = promotePawn(sourcePiece, move.PromotionPiece)
	// } else {
	// 	// Perform the move
	// 	game.ChessState.Board[move.DestinationSquare] = sourcePiece
	// }
	board[move.Destination] = sourcePiece
	board[move.Source] = "---"

	// Capture piece if necessary
	// if destinationPiece != "---" {
	// 	log.Infof("Piece captured: %s", destinationPiece)
	// 	game.ChessState.CapturedPieces = append(game.ChessState.CapturedPieces, destinationPiece)
	// }

	// Update game state
	// game.ChessState.MoveHistory = append(game.ChessState.MoveHistory, move)
	// game.ChessState.CurrentTurn = switchTurn(currentTurn)
	log.Info(move.Source, move.Destination)
	// log.Info(board)
	updatedBoard, err := json.Marshal(board)
	if err != nil {
		log.Errorf("failed to marshal board: %v", err)
		return fmt.Errorf("failed to marshal board: %w", err)
	}
	game.ChessState.Board = json.RawMessage(updatedBoard)
	game.ChessState.Turn = switchTurn(currentTurn)
	log.Infof("Move successful: %s moved to %s", sourcePiece, move.Destination)
	return nil
}

// Helper to check if a slice contains a specific value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// Helper to switch turn
func switchTurn(currentTurn string) string {
	if currentTurn == "white" {
		return "black"
	}
	return "white"
}

// Check if the move is a pawn promotion
func isPawnPromotion(piece string, destinationSquare string) bool {
	// Assume the last rank for white is 8 and for black is 1
	if piece == "wP" && destinationSquare[1] == '8' {
		return true
	} else if piece == "bP" && destinationSquare[1] == '1' {
		return true
	}
	return false
}

// Promote pawn to the specified piece
func promotePawn(pawn string, promotionPiece string) string {
	// Validate promotionPiece (e.g., must be "wQ", "wR", etc.)
	if promotionPiece != "" {
		return promotionPiece
	}
	// Default to queen if no promotion piece specified
	return pawn[:1] + "Q" // Retain the color and promote to queen
}
