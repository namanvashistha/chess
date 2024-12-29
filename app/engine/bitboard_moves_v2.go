package engine

import (
	"chess-engine/app/domain/dao"
	"log"
)

// GenerateMovesForAllPositions generates all possible moves for all pieces on the board
func GenerateLegalMovesForAllPositions(gs dao.GameState) map[uint64]uint64 {
	moves := make(map[uint64]uint64)

	// Combined occupancy bitboards
	allOccupancy := gs.WhiteBitboard | gs.BlackBitboard
	friendlyOccupancy := gs.WhiteBitboard
	if gs.Turn == "b" {
		friendlyOccupancy = gs.BlackBitboard
	}

	// Generate moves for each type of piece
	generatePawnMoves(gs.PawnBitboard, allOccupancy, friendlyOccupancy, gs.EnPassant, moves)
	generateKnightMoves(gs.KnightBitboard, allOccupancy, friendlyOccupancy, moves)
	generateBishopMoves(gs.BishopBitboard, allOccupancy, moves)
	generateRookMoves(gs.RookBitboard, allOccupancy, moves)
	generateQueenMoves(gs.QueenBitboard, allOccupancy, moves)
	generateKingMoves(gs.KingBitboard, allOccupancy, friendlyOccupancy, moves)

	log.Println(moves)
	return moves
}

// generatePawnMoves generates moves for pawns, considering forward moves, captures, and en passant
func generatePawnMoves(pawnBitboard uint64, allOccupancy uint64, friendlyOccupancy uint64, enPassant uint64, moves map[uint64]uint64) {
	var pawnMoves uint64
	for pawnBitboard != 0 {
		piece := pawnBitboard & -pawnBitboard // Isolate the least significant bit (LSB)
		pawnBitboard &= pawnBitboard - 1      // Remove the LSB

		// Move one square forward (check for empty square)
		if piece<<8&allOccupancy == 0 { // Square in front is empty
			// Ensure the pawn isn't on the 8th rank (promoted pawns shouldn't move forward)
			if piece<<8&0xFF000000000000 == 0 {
				pawnMoves |= piece << 8
			}
		}

		// Move two squares forward from the starting position (only for pawns on the second rank)
		if piece&0xFF000000000000 == 0 && piece<<16&allOccupancy == 0 { // On second rank and both squares empty
			// Make sure the piece is on the second rank and not promoting
			if piece<<16&0xFF000000000000 == 0 {
				pawnMoves |= piece << 16
			}
		}

		// Pawn captures diagonally to the left (only if the target is occupied by an opponent)
		if piece<<7&allOccupancy != 0 && piece<<7&friendlyOccupancy == 0 {
			// Make sure it's within bounds (not off the board)
			if piece<<7&0xFEFEFEFEFEFEFEFE != 0 {
				pawnMoves |= piece << 7
			}
		}

		// Pawn captures diagonally to the right (only if the target is occupied by an opponent)
		if piece<<9&allOccupancy != 0 && piece<<9&friendlyOccupancy == 0 {
			// Make sure it's within bounds (not off the board)
			if piece<<9&0xFEFEFEFEFEFEFEFE != 0 {
				pawnMoves |= piece << 9
			}
		}

		// En Passant captures to the left (check if the enPassant square is to the left)
		if enPassant != 0 && piece<<7&enPassant != 0 {
			// Make sure it's within bounds (not off the board)
			if piece<<7&0xFEFEFEFEFEFEFEFE != 0 {
				pawnMoves |= piece << 7
			}
		}

		// En Passant captures to the right (check if the enPassant square is to the right)
		if enPassant != 0 && piece<<9&enPassant != 0 {
			// Make sure it's within bounds (not off the board)
			if piece<<9&0xFEFEFEFEFEFEFEFE != 0 {
				pawnMoves |= piece << 9
			}
		}

		// Check for pawn promotion
		if piece&0xFF000000000000 == 0 {
			pawnMoves |= piece << 8 // Move one square forward for promotion
		}

		moves[piece] = pawnMoves
		pawnMoves = 0 // Reset the pawn moves for the next piece
	}
}

// generateKnightMoves generates legal moves for knight pieces
func generateKnightMoves(knightBitboard, allOccupancy, friendlyOccupancy uint64, moves map[uint64]uint64) {
	offsets := []int{-17, -15, -10, -6, 6, 10, 15, 17}
	for knightBitboard != 0 {
		piece := knightBitboard & -knightBitboard
		knightBitboard &= knightBitboard - 1

		var knightMoves uint64
		for _, offset := range offsets {
			move := moveSquare(piece, offset)
			if isValidSquare(move) && (move&friendlyOccupancy == 0) {
				knightMoves |= move
			}
		}

		moves[piece] = knightMoves
		knightMoves = 0 // Reset the knight moves for the next piece
	}
}

// generateBishopMoves generates moves for bishop pieces (diagonal sliding)
func generateBishopMoves(bishopBitboard, allOccupancy uint64, moves map[uint64]uint64) {
	directions := []int{-9, -7, 7, 9} // Diagonal directions
	for bishopBitboard != 0 {
		piece := bishopBitboard & -bishopBitboard
		bishopBitboard &= bishopBitboard - 1

		var bishopMoves uint64
		for _, direction := range directions {
			for currentSquare := piece; isValidSquare(currentSquare); currentSquare = moveSquare(currentSquare, direction) {
				if allOccupancy&currentSquare != 0 { // Stop if occupied
					break
				}
				bishopMoves |= currentSquare
			}
		}

		moves[piece] = bishopMoves
		bishopMoves = 0 // Reset the bishop moves for the next piece
	}
}

// generateRookMoves generates moves for rook pieces (horizontal and vertical sliding)
func generateRookMoves(rookBitboard, allOccupancy uint64, moves map[uint64]uint64) {
	directions := []int{-8, -1, 1, 8} // Horizontal and vertical directions
	for rookBitboard != 0 {
		piece := rookBitboard & -rookBitboard
		rookBitboard &= rookBitboard - 1

		var rookMoves uint64
		for _, direction := range directions {
			for currentSquare := piece; isValidSquare(currentSquare); currentSquare = moveSquare(currentSquare, direction) {
				if allOccupancy&currentSquare != 0 { // Stop if occupied
					break
				}
				rookMoves |= currentSquare
			}
		}

		moves[piece] = rookMoves
		rookMoves = 0 // Reset the rook moves for the next piece
	}
}

// generateQueenMoves generates moves for queen pieces (combination of rook and bishop)
func generateQueenMoves(queenBitboard, allOccupancy uint64, moves map[uint64]uint64) {
	directions := []int{-9, -7, -8, -1, 1, 7, 8, 9} // Combination of diagonal, horizontal, and vertical
	for queenBitboard != 0 {
		piece := queenBitboard & -queenBitboard
		queenBitboard &= queenBitboard - 1

		var queenMoves uint64
		for _, direction := range directions {
			for currentSquare := piece; isValidSquare(currentSquare); currentSquare = moveSquare(currentSquare, direction) {
				if allOccupancy&currentSquare != 0 { // Stop if occupied
					break
				}
				queenMoves |= currentSquare
			}
		}

		moves[piece] = queenMoves
	}
}

// generateKingMoves generates moves for king pieces (one square in any direction)
func generateKingMoves(kingBitboard, allOccupancy, friendlyOccupancy uint64, moves map[uint64]uint64) {
	offsets := []int{-1, 1, -8, 8, -9, 9, -7, 7}
	for kingBitboard != 0 {
		piece := kingBitboard & -kingBitboard
		kingBitboard &= kingBitboard - 1

		var kingMoves uint64
		for _, offset := range offsets {
			move := moveSquare(piece, offset)
			if isValidSquare(move) && (move&friendlyOccupancy == 0) {
				kingMoves |= move
			}
		}

		moves[piece] = kingMoves
	}
}

// isValidSquare checks if a square is within the bounds of the board (0-63)
func isValidSquare(square uint64) bool {
	return square >= 1 && square <= 0x8000000000000000
}

// moveSquare returns the square after applying a given offset (used for knight, king, etc.)
func moveSquare(piece uint64, offset int) uint64 {
	return piece + uint64(offset)
}
