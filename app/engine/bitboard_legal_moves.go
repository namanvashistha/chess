package engine

import (
	"chess-engine/app/domain/dao"

	log "github.com/sirupsen/logrus"
)

// GenerateMovesForAllPositions generates all possible moves for all pieces on the board
func GenerateLegalMovesForAllPositions(gs dao.GameState) map[uint64]uint64 {
	pseudo_legal_moves := make(map[uint64]uint64)
	legal_moves := make(map[uint64]uint64)

	// Generate moves for each type of piece

	generatePawnMoves(gs, pseudo_legal_moves, legal_moves)
	generateKnightMoves(gs, pseudo_legal_moves, legal_moves)
	generateBishopMoves(gs, pseudo_legal_moves, legal_moves)
	generateRookMoves(gs, pseudo_legal_moves, legal_moves)
	generateQueenMoves(gs, pseudo_legal_moves, legal_moves)
	generateKingMoves(gs, pseudo_legal_moves, legal_moves)

	isWhiteKingInCheck := CheckIfKingIsInCheck(gs, pseudo_legal_moves, true)
	if isWhiteKingInCheck {
		log.Println("WHITE KING IS IN CHECK")

	}
	isBlackKingInCheck := CheckIfKingIsInCheck(gs, pseudo_legal_moves, false)
	if isBlackKingInCheck {
		log.Println("BLACK KING IS IN CHECK")
	}
	return legal_moves
}

// generatePawnMoves generates moves for pawns, considering forward moves, captures, and en passant
func generatePawnMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {

	for whitePawnBitboard := (gs.PawnBitboard & gs.WhiteBitboard); whitePawnBitboard != 0; whitePawnBitboard &= whitePawnBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the king's position
		piece := whitePawnBitboard & -whitePawnBitboard
		pawnMoves := WhitePawnAttackBitboard[piece]
		pawnMoves &= ^gs.WhiteBitboard

		legal_moves[piece] = pawnMoves
		pseudo_legal_moves[piece] = pawnMoves
	}

	for blackPawnBitboard := (gs.PawnBitboard & gs.BlackBitboard); blackPawnBitboard != 0; blackPawnBitboard &= blackPawnBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the king's position
		piece := blackPawnBitboard & -blackPawnBitboard
		pawnMoves := BlackPawnAttackBitboard[piece]
		pawnMoves &= ^gs.BlackBitboard

		legal_moves[piece] = pawnMoves
		pseudo_legal_moves[piece] = pawnMoves
	}

}

func generateKnightMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {
	// Iterate over all knights on the board
	for knightBitboard := gs.KnightBitboard; knightBitboard != 0; knightBitboard &= knightBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the knight's position
		piece := knightBitboard & -knightBitboard

		// Retrieve precomputed knight moves from the map
		knightMoves := KnightAttackBitboard[piece]

		// Determine the color of the knight and mask out same-color occupied squares
		if gs.WhiteBitboard&piece != 0 {
			// White knight, exclude moves to squares occupied by white pieces
			knightMoves &= ^gs.WhiteBitboard
		} else if gs.BlackBitboard&piece != 0 {
			// Black knight, exclude moves to squares occupied by black pieces
			knightMoves &= ^gs.BlackBitboard
		}

		// Store the generated moves in the map
		legal_moves[piece] = knightMoves
		pseudo_legal_moves[piece] = knightMoves
	}
}

func generateBishopMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {
	// Combine all occupied squares (both white and black pieces)
	allOccupied := gs.WhiteBitboard | gs.BlackBitboard

	// Iterate over all bishops on the board
	for bishopBitboard := gs.BishopBitboard; bishopBitboard != 0; bishopBitboard &= bishopBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the bishop's position
		piece := bishopBitboard & -bishopBitboard

		// Get all potential moves for the bishop
		bishopMoves := BishopAttackBitboard[piece]
		pseudo_legal_moves[piece] = bishopMoves

		rayDirections := []int{-9, -7, 7, 9}

		bishopMoves = removeBlockedMoves(piece, bishopMoves, allOccupied, rayDirections)

		// Mask out same-color occupied squares
		if gs.WhiteBitboard&piece != 0 {
			// White bishop, exclude moves to squares occupied by white pieces
			blackKingBitboard := gs.BlackBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.WhiteBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^blackKingBitboard, rayDirections)
			bishopMoves &= ^gs.WhiteBitboard

		} else if gs.BlackBitboard&piece != 0 {
			// Black bishop, exclude moves to squares occupied by black pieces
			whiteKingBitboard := gs.WhiteBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.BlackBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^whiteKingBitboard, rayDirections)
			bishopMoves &= ^gs.BlackBitboard

		}

		// Store the generated moves in the map
		legal_moves[piece] = bishopMoves
	}
}

// generateRookMoves generates moves for rook pieces (horizontal and vertical sliding)
func generateRookMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {
	// Combine all occupied squares (both white and black pieces)
	allOccupied := gs.WhiteBitboard | gs.BlackBitboard

	// Iterate over all bishops on the board
	for rookBitboard := gs.RookBitboard; rookBitboard != 0; rookBitboard &= rookBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the bishop's position
		piece := rookBitboard & -rookBitboard

		// Get all potential moves for the bishop
		rookMoves := RookAttackBitboard[piece]
		pseudo_legal_moves[piece] = rookMoves

		rayDirections := []int{-8, 8, -1, 1}
		rookMoves = removeBlockedMoves(piece, rookMoves, allOccupied, rayDirections)

		// Mask out same-color occupied squares
		if gs.WhiteBitboard&piece != 0 {
			blackKingBitboard := gs.BlackBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.WhiteBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^blackKingBitboard, rayDirections)
			rookMoves &= ^gs.WhiteBitboard
		} else if gs.BlackBitboard&piece != 0 {
			// Black bishop, exclude moves to squares occupied by black pieces
			whiteKingBitboard := gs.WhiteBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.BlackBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^whiteKingBitboard, rayDirections)
			rookMoves &= ^gs.BlackBitboard
		}

		// Store the generated moves in the map
		legal_moves[piece] = rookMoves
	}
}

// generateQueenMoves generates moves for queen pieces (diagonal, horizontal, and vertical sliding)
func generateQueenMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {
	// Combine all occupied squares (both white and black pieces)
	allOccupied := gs.WhiteBitboard | gs.BlackBitboard

	// Iterate over all bishops on the board
	for queenBitboard := gs.QueenBitboard; queenBitboard != 0; queenBitboard &= queenBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the bishop's position
		piece := queenBitboard & -queenBitboard

		// Get all potential moves for the bishop
		queenMoves := BishopAttackBitboard[piece] | RookAttackBitboard[piece]
		pseudo_legal_moves[piece] = queenMoves

		rayDirections := []int{-8, 8, -1, 1, -9, -7, 7, 9}
		queenMoves = removeBlockedMoves(piece, queenMoves, allOccupied, rayDirections)

		// Mask out same-color occupied squares
		if gs.WhiteBitboard&piece != 0 {
			// White bishop, exclude moves to squares occupied by white pieces
			blackKingBitboard := gs.BlackBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.WhiteBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^blackKingBitboard, rayDirections)
			queenMoves &= ^gs.WhiteBitboard
		} else if gs.BlackBitboard&piece != 0 {
			// Black bishop, exclude moves to squares occupied by black pieces
			whiteKingBitboard := gs.WhiteBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.BlackBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^whiteKingBitboard, rayDirections)
			queenMoves &= ^gs.BlackBitboard
		}

		// Store the generated moves in the map
		legal_moves[piece] = queenMoves
	}
}

func generateKingMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {
	// Iterate over all kings on the board
	for kingBitboard := gs.KingBitboard; kingBitboard != 0; kingBitboard &= kingBitboard - 1 {
		// Isolate the least significant bit (LSB) representing the king's position
		piece := kingBitboard & -kingBitboard

		// Retrieve precomputed king moves from the magic bitboard map
		kingMoves := KingAttackBitboard[piece]

		// Determine the color of the king and mask out same-color occupied squares
		if gs.WhiteBitboard&piece != 0 {
			// White king, exclude moves to squares occupied by white pieces
			kingMoves &= ^gs.WhiteBitboard
			attackedSquares := getAttackedSquares(gs.BlackBitboard, pseudo_legal_moves)
			kingMoves &= ^attackedSquares
		} else if gs.BlackBitboard&piece != 0 {
			// Black king, exclude moves to squares occupied by black pieces
			kingMoves &= ^gs.BlackBitboard
			attackedSquares := getAttackedSquares(gs.WhiteBitboard, pseudo_legal_moves)
			kingMoves &= ^attackedSquares
		}

		// Store the generated moves in the map
		legal_moves[piece] = kingMoves
		pseudo_legal_moves[piece] = kingMoves
	}
}

func getAttackedSquares(enemyBitboard uint64, moves map[uint64]uint64) uint64 {
	attackedSquares := uint64(0)

	for enemyBitboard != 0 {
		piece := enemyBitboard & -enemyBitboard
		if (moves[piece]) != 0 {
			attackedSquares |= moves[piece]
		}
		enemyBitboard &= enemyBitboard - 1
	}

	return attackedSquares
}

func removeBlockedMoves(piece uint64, moves uint64, allOccupied uint64, rayDirections []int) uint64 {
	blockedMoves := uint64(0)

	// Define ray directions: NW, NE, SW, SE

	// Iterate over each direction
	for _, direction := range rayDirections {
		blockedMoves |= traceRay(piece, direction, allOccupied)
	}

	// Remove blocked moves from the potential moves
	return moves & blockedMoves
}

// Trace a single ray in a given direction and find blocked squares
func traceRay(start uint64, direction int, allOccupied uint64) uint64 {
	blocked := uint64(0)
	ray := start

	for {
		if direction > 0 {
			ray <<= direction
		} else {
			ray >>= -direction
		}

		// Stop if the ray moves out of bounds
		if ray == 0 || crossesBoundary(ray, direction) {
			break
		}

		// If there's a blocker, stop further moves along this ray
		blocked |= ray
		if ray&allOccupied != 0 {
			break
		}
	}

	return blocked
}

// Helper function to detect if a move crosses board boundaries
func crossesBoundary(bitboard uint64, direction int) bool {
	if direction == -9 || direction == 7 { // Moving NW or SE
		return (bitboard & 0x8080808080808080) != 0 // Crossing left edge
	} else if direction == -7 || direction == 9 { // Moving NE or SW
		return (bitboard & 0x0101010101010101) != 0 // Crossing right edge
	}
	return false
}

func CheckIfKingIsInCheck(gs dao.GameState, moves map[uint64]uint64, isWhiteKing bool) bool {
	// Determine the king's position
	var kingBitboard uint64
	if isWhiteKing {
		kingBitboard = gs.KingBitboard & gs.WhiteBitboard
	} else {
		kingBitboard = gs.KingBitboard & gs.BlackBitboard
	}

	// Isolate the king's position
	kingPosition := kingBitboard & -kingBitboard

	// Iterate over all opponent pieces in the `moves` map
	for piece, attackSet := range moves {
		// Skip pieces of the same color as the king
		if (gs.WhiteBitboard&piece != 0 && isWhiteKing) || (gs.BlackBitboard&piece != 0 && !isWhiteKing) {
			continue
		}

		// Check if the king's position is attacked
		if attackSet&kingPosition != 0 {
			return true
		}
	}

	// No opponent piece attacks the king's position
	return false
}

func isPieceAttackingKing(gs dao.GameState, kingPosition uint64, opponentBitboard uint64, moves map[uint64]uint64) bool {
	for opponentBitboard != 0 {
		piece := opponentBitboard & -opponentBitboard
		if (moves[piece] & kingPosition) != 0 {
			return true
		}
		opponentBitboard &= opponentBitboard - 1
	}
	return false
}
