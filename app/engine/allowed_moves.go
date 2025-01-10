package engine

import (
	"chess-engine/app/domain/dao"
	"fmt"
)

func GetAllowedMoves(game dao.ChessGame) map[string][]string {
	allowedMoves := make(map[string][]string)

	var board map[string]string

	// err := json.Unmarshal([]byte(game.ChessState.Board), &board)
	// if err != nil {
	// 	fmt.Println("Error unmarshaling board:", err)
	// 	return allowedMoves
	// }
	for square, pieceCode := range board {
		// squareColor := data[0]
		// pieceCode := data[1]
		pieceColor := pieceCode[0:1]
		pieceType := pieceCode[1:2]
		// pieceId := pieceCode[2:]

		if pieceType == "R" {
			allowedMoves[pieceCode] = GetAllowedMovesRook(square, pieceColor, board)
		}
		if pieceType == "B" {
			allowedMoves[pieceCode] = GetAllowedMovesBishop(square, pieceColor, board)
		}
		if pieceType == "Q" {
			allowedMoves[pieceCode] = GetAllowedMovesQueen(square, pieceColor, board)
		}
		if pieceType == "K" {
			allowedMoves[pieceCode] = GetAllowedMovesKing(square, pieceColor, board)
		}
		if pieceType == "N" {
			allowedMoves[pieceCode] = GetAllowedMovesKnight(square, pieceColor, board)
		}
		if pieceType == "P" {
			allowedMoves[pieceCode] = GetAllowedMovesPawn(square, pieceColor, board)
		}

	}

	return allowedMoves
}
func GetAllowedMovesPawn(square string, pieceColor string, board map[string]string) []string {
	var allowed []string
	var direction int
	var startRank int

	// Define pawn direction based on color
	// White pawns move upwards (rank 2 to 8), Black pawns move downwards (rank 7 to 1)
	if pieceColor == "w" {
		direction = 1
		startRank = 2
	} else {
		direction = -1
		startRank = 7
	}

	squareFile := square[0]            // e.g., 'a'
	squareRank := int(square[1] - '0') // e.g., 1 for 'a1'

	// Check the square directly in front of the pawn
	frontSquare := fmt.Sprintf("%c%d", squareFile, squareRank+direction)
	frontSquareData, exists := board[frontSquare]
	if !exists || frontSquareData == "---" {
		// It's an empty square, add it as a valid move
		allowed = append(allowed, frontSquare)
	}

	// If it's the pawn's starting position, check the two-square move
	if squareRank == startRank {
		frontTwoSquares := fmt.Sprintf("%c%d", squareFile, squareRank+2*direction)
		frontTwoSquaresData, exists := board[frontTwoSquares]
		if !exists || frontTwoSquaresData == "---" {
			// It's an empty square, add it as a valid move
			allowed = append(allowed, frontTwoSquares)
		}
	}

	// Check for diagonal captures
	for _, offset := range []int{-1, 1} {
		// Convert squareFile to an integer (ASCII value) before adding the offset
		newFile := byte(int(squareFile) + offset)
		diagonalSquare := fmt.Sprintf("%c%d", newFile, squareRank+direction)
		diagonalSquareData, exists := board[diagonalSquare]
		if exists && diagonalSquareData != "---" {
			// If it's an opponent's piece, add it as a valid capture move
			diagonalPieceColor := diagonalSquareData[0:1]
			if diagonalPieceColor != pieceColor {
				allowed = append(allowed, diagonalSquare)
			}
		}
	}

	return allowed
}
func GetAllowedMovesKnight(square string, pieceColor string, board map[string]string) []string {
	var allowed []string

	// Directions the knight can move: L-shaped moves
	directions := [][2]int{
		{2, 1},   // Two up, one right
		{2, -1},  // Two up, one left
		{-2, 1},  // Two down, one right
		{-2, -1}, // Two down, one left
		{1, 2},   // One up, two right
		{1, -2},  // One up, two left
		{-1, 2},  // One down, two right
		{-1, -2}, // One down, two left
	}

	squareFile := square[0]            // e.g., 'a'
	squareRank := int(square[1] - '0') // e.g., 1 for 'a1'

	// Iterate over each direction and check possible moves
	for _, dir := range directions {
		curFileIdx := int(squareFile - 'a') // Convert file ('a' -> 0, 'b' -> 1, etc.)
		curRank := squareRank

		// Move in the current direction
		curFileIdx += dir[0]
		curRank += dir[1]

		// Check if within bounds (file 'a' to 'h', rank 1 to 8)
		if curFileIdx < 0 || curFileIdx >= 8 || curRank < 1 || curRank > 8 {
			continue
		}

		// Create new square string (e.g., "a1", "b3", etc.)
		newSquare := fmt.Sprintf("%c%d", 'a'+curFileIdx, curRank)

		// Get the piece at the new square
		newSquareData, exists := board[newSquare]

		// If there's no piece, it's a valid move
		if !exists || newSquareData == "---" {
			allowed = append(allowed, newSquare)
		} else {
			// If the square is occupied, check if it's the same color or an opponent's piece
			newPieceColor := newSquareData[0:1]

			// If it's an opponent's piece (different color), it's a valid capture move
			if newPieceColor != pieceColor {
				allowed = append(allowed, newSquare)
			}
		}
	}

	return allowed
}

func GetAllowedMovesBishop(square string, pieceColor string, board map[string]string) []string {
	var allowed []string

	// Directions the bishop can move: top-right, top-left, bottom-right, bottom-left
	directions := [][2]int{
		{1, 1},   // Top-right
		{-1, 1},  // Top-left
		{1, -1},  // Bottom-right
		{-1, -1}, // Bottom-left
	}

	squareFile := square[0]            // e.g., 'a'
	squareRank := int(square[1] - '0') // e.g., 1 for 'a1'

	// Iterate over each direction and check possible moves
	for _, dir := range directions {
		curFileIdx := int(squareFile - 'a') // Convert file ('a' -> 0, 'b' -> 1, etc.)
		curRank := squareRank

		for {
			// Move in the current direction
			curFileIdx += dir[0]
			curRank += dir[1]

			// Check if within bounds (file 'a' to 'h', rank 1 to 8)
			if curFileIdx < 0 || curFileIdx >= 8 || curRank < 1 || curRank > 8 {
				break
			}

			// Create new square string (e.g., "a1", "b3", etc.)
			newSquare := fmt.Sprintf("%c%d", 'a'+curFileIdx, curRank)

			// Get the piece at the new square
			newSquareData, exists := board[newSquare]

			// If there's no piece, it's a valid move
			if !exists || newSquareData == "---" {
				allowed = append(allowed, newSquare)
			} else {
				// If the square is occupied, check if it's the same color or an opponent's piece
				newPieceColor := newSquareData[0:1]

				// If it's an opponent's piece (different color), it's a valid capture move
				if newPieceColor != pieceColor {
					allowed = append(allowed, newSquare)
				}
				// Stop if the square is occupied (either same color or different color)
				break
			}
		}
	}

	return allowed
}

func GetAllowedMovesRook(square string, pieceColor string, board map[string]string) []string {
	var allowed []string

	// Directions the rook can move: up, down, left, right
	directions := [][2]int{
		{0, 1},  // Right
		{0, -1}, // Left
		{1, 0},  // Up
		{-1, 0}, // Down
	}

	squareFile := square[0]            // e.g., 'a'
	squareRank := int(square[1] - '0') // e.g., 1 for 'a1'

	// Iterate over each direction and check possible moves
	for _, dir := range directions {
		curFileIdx := int(squareFile - 'a') // Convert file ('a' -> 0, 'b' -> 1, etc.)
		curRank := squareRank

		for {
			// Move in the current direction
			curFileIdx += dir[0]
			curRank += dir[1]

			// Check if within bounds (file 'a' to 'h', rank 1 to 8)
			if curFileIdx < 0 || curFileIdx >= 8 || curRank < 1 || curRank > 8 {
				break
			}

			// Create new square string (e.g., "a1", "b3", etc.)
			newSquare := fmt.Sprintf("%c%d", 'a'+curFileIdx, curRank)

			// Get the piece at the new square
			newSquareData, exists := board[newSquare]

			// If there's no piece, it's a valid move
			if !exists || newSquareData == "---" {
				allowed = append(allowed, newSquare)
			} else {
				// If the square is occupied, check if it's the same color or an opponent's piece
				newPieceColor := newSquareData[0:1] // Corrected: piece color is at [1][0]

				// If it's an opponent's piece (different color), it's a valid capture move
				if newPieceColor != pieceColor {
					allowed = append(allowed, newSquare)
				}
				// Stop if the square is occupied (either same color or different color)
				break
			}
		}
	}

	return allowed
}
func GetAllowedMovesQueen(square string, pieceColor string, board map[string]string) []string {
	var allowed []string

	// Directions the queen can move: up, down, left, right, top-right, top-left, bottom-right, bottom-left
	directions := [][2]int{
		{0, 1},   // Right (rook move)
		{0, -1},  // Left (rook move)
		{1, 0},   // Up (rook move)
		{-1, 0},  // Down (rook move)
		{1, 1},   // Top-right (bishop move)
		{-1, 1},  // Top-left (bishop move)
		{1, -1},  // Bottom-right (bishop move)
		{-1, -1}, // Bottom-left (bishop move)
	}

	squareFile := square[0]            // e.g., 'a'
	squareRank := int(square[1] - '0') // e.g., 1 for 'a1'

	// Iterate over each direction and check possible moves
	for _, dir := range directions {
		curFileIdx := int(squareFile - 'a') // Convert file ('a' -> 0, 'b' -> 1, etc.)
		curRank := squareRank

		for {
			// Move in the current direction
			curFileIdx += dir[0]
			curRank += dir[1]

			// Check if within bounds (file 'a' to 'h', rank 1 to 8)
			if curFileIdx < 0 || curFileIdx >= 8 || curRank < 1 || curRank > 8 {
				break
			}

			// Create new square string (e.g., "a1", "b3", etc.)
			newSquare := fmt.Sprintf("%c%d", 'a'+curFileIdx, curRank)

			// Get the piece at the new square
			newSquareData, exists := board[newSquare]

			// If there's no piece, it's a valid move
			if !exists || newSquareData == "---" {
				allowed = append(allowed, newSquare)
			} else {
				// If the square is occupied, check if it's the same color or an opponent's piece
				newPieceColor := newSquareData[0:1]

				// If it's an opponent's piece (different color), it's a valid capture move
				if newPieceColor != pieceColor {
					allowed = append(allowed, newSquare)
				}
				// Stop if the square is occupied (either same color or different color)
				break
			}
		}
	}

	return allowed
}

func GetAllowedMovesKing(square string, pieceColor string, board map[string]string) []string {
	var allowed []string

	// Directions the king can move: up, down, left, right, top-right, top-left, bottom-right, bottom-left
	directions := [][2]int{
		{0, 1},   // Right
		{0, -1},  // Left
		{1, 0},   // Up
		{-1, 0},  // Down
		{1, 1},   // Top-right
		{-1, 1},  // Top-left
		{1, -1},  // Bottom-right
		{-1, -1}, // Bottom-left
	}

	squareFile := square[0]            // e.g., 'a'
	squareRank := int(square[1] - '0') // e.g., 1 for 'a1'

	// Iterate over each direction and check possible moves
	for _, dir := range directions {
		curFileIdx := int(squareFile - 'a') // Convert file ('a' -> 0, 'b' -> 1, etc.)
		curRank := squareRank

		// Move one step in the current direction
		curFileIdx += dir[0]
		curRank += dir[1]

		// Check if within bounds (file 'a' to 'h', rank 1 to 8)
		if curFileIdx < 0 || curFileIdx >= 8 || curRank < 1 || curRank > 8 {
			continue
		}

		// Create new square string (e.g., "a1", "b3", etc.)
		newSquare := fmt.Sprintf("%c%d", 'a'+curFileIdx, curRank)

		// Get the piece at the new square
		newSquareData, exists := board[newSquare]

		// If there's no piece, it's a valid move
		if !exists || newSquareData == "---" {
			allowed = append(allowed, newSquare)
		} else {
			// If the square is occupied, check if it's the same color or an opponent's piece
			newPieceColor := newSquareData[0:1]

			// If it's an opponent's piece (different color), it's a valid capture move
			if newPieceColor != pieceColor {
				allowed = append(allowed, newSquare)
			}
		}
	}

	return allowed
}
