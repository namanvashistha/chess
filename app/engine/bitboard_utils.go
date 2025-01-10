package engine

import (
	"chess-engine/app/domain/dao"
	"fmt"
)

func getPieceCode(bit uint64, isWhite bool, gameState dao.GameState) string {
	// Check which bitboard the piece belongs to
	pieceTypes := []string{"pawn", "rook", "knight", "bishop", "queen", "king"}
	pieceCodes := []string{"p", "r", "n", "b", "q", "k"}

	bitboards := map[string]uint64{
		"pawn":   gameState.PawnBitboard,
		"rook":   gameState.RookBitboard,
		"knight": gameState.KnightBitboard,
		"bishop": gameState.BishopBitboard,
		"queen":  gameState.QueenBitboard,
		"king":   gameState.KingBitboard,
	}

	for i, piece := range pieceTypes {
		if bitboards[piece]&bit != 0 {
			if isWhite {
				return string(pieceCodes[i][0] - 32) // Uppercase for white pieces
			}
			return pieceCodes[i]
		}
	}
	return ""
}

func ConvertGameStateToMap(gameState dao.GameState) map[string]string {
	board := make(map[string]string)

	files := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	// files := []string{"h", "g", "f", "e", "d", "c", "b", "a"}
	// ranks := []string{"8", "7", "6", "5", "4", "3", "2", "1"}
	ranks := []string{"1", "2", "3", "4", "5", "6", "7", "8"}

	// Create the board layout
	for i := 0; i < 64; i++ {
		row := i / 8
		col := i % 8
		squareKey := fmt.Sprintf("%s%s", files[col], ranks[row])
		bit := uint64(1) << uint(i)

		// Check for pieces in the bitboards and assign the corresponding piece code
		if gameState.WhiteBitboard&bit != 0 {
			board[squareKey] = getPieceCode(bit, true, gameState)
		} else if gameState.BlackBitboard&bit != 0 {
			board[squareKey] = getPieceCode(bit, false, gameState)
		}
		// log.Info("squareKey: ", squareKey)
		// log.Info("board[squareKey]: ", board[squareKey])

	}

	return board
}

// func ConvertAllowedMovesToMap(allowedMoves map[uint64]uint64) map[string][]string {
// 	log.Info(allowedMoves)
// 	moves := make(map[string][]string)

// 	files := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
// 	ranks := []string{"8", "7", "6", "5", "4", "3", "2", "1"}

// 	// Create the board layout
// 	for i := 0; i < 64; i++ {
// 		row := i / 8
// 		col := i % 8
// 		squareKey := fmt.Sprintf("%s%s", files[col], ranks[row])
// 		bit := uint64(1) << uint(i)

// 		// Check for pieces in the bitboards and assign the corresponding piece code
// 		if _, ok := allowedMoves[bit]; ok {
// 			moves[squareKey] = []string{}
// 			for j := 0; j < 64; j++ {
// 				if allowedMoves[bit]&(1<<uint(j)) != 0 {
// 					row := j / 8
// 					col := j % 8
// 					moves[squareKey] = append(moves[squareKey], fmt.Sprintf("%s%s", files[col], ranks[row]))
// 				}
// 			}
// 		}
// 	}

// 	return moves
// }

func ConvertLegalMovesToMap(allowedMoves map[uint64]uint64) map[string][]string {
	moves := make(map[string][]string)

	files := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	ranks := []string{"1", "2", "3", "4", "5", "6", "7", "8"}
	// ranks := []string{"8", "7", "6", "5", "4", "3", "2", "1"}

	// Iterate over all allowed moves
	for piecePos, moveBitboard := range allowedMoves {
		pieceSquare := bitToSquare(piecePos, files, ranks)
		moves[pieceSquare] = []string{}
		// log.Info("piecePos:pieceSquare:moveBitboard ", piecePos, ":", pieceSquare, ":", moveBitboard)

		// Iterate over all possible moves
		for bitIndex := 0; bitIndex < 64; bitIndex++ {
			if moveBitboard&(uint64(1)<<uint(bitIndex)) != 0 {
				targetSquare := bitToSquare(uint64(1)<<uint(bitIndex), files, ranks)
				moves[pieceSquare] = append(moves[pieceSquare], targetSquare)
				// log.Info("bitIndex:pieceSquare:targetSquare ", bitIndex, ":", pieceSquare, ":", targetSquare)
			}

		}
	}

	return moves
}

func bitToSquare(bitboard uint64, files, ranks []string) string {
	if bitboard == 0 {
		return ""
	}

	// Find the bit index (position of the piece in the bitboard)
	bitIndex := 0
	for bitboard != 1 {
		bitboard >>= 1
		bitIndex++
	}

	file := files[bitIndex%8] // Column (0-7 maps to a-h)
	rank := ranks[bitIndex/8] // Row (0-7 maps to 1-8, bottom to top)
	return file + rank
}

func PositionToIndex(position string) int {
	// Convert the chessboard position (e.g., "c1") to an index (e.g., 0-63)
	// This will depend on your bitboard encoding
	column := position[0] - 'a'
	row := position[1] - '1'
	return int(row*8 + column) // Assuming 8x8 board
}
