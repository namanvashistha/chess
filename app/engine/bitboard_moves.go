package engine

// import (
// 	"chess-engine/app/domain/dao"
// 	"math/bits"
// )

// // Constants for board size
// const (
// 	AFile uint64 = 0x0101010101010101 // Mask for the A file
// 	BFile uint64 = 0x0202020202020202 // Mask for the B file
// 	CFile uint64 = 0x0404040404040404 // Mask for the C file
// 	DFile uint64 = 0x0808080808080808 // Mask for the D file
// 	EFile uint64 = 0x1010101010101010 // Mask for the E file
// 	FFile uint64 = 0x2020202020202020 // Mask for the F file
// 	GFile uint64 = 0x4040404040404040 // Mask for the G file
// 	HFile uint64 = 0x8080808080808080 // Mask for the H file

// 	AllSquares uint64 = 0xFFFFFFFFFFFFFFFF // All squares on the board
// )

// // Precomputed tables (simplified for brevity, extend these as needed)
// var knightMoves = [64]uint64{}
// var kingMoves = [64]uint64{}

// func init() {
// 	for square := 0; square < 64; square++ {
// 		knightMoves[square] = calculateKnightMoves(square)
// 		kingMoves[square] = calculateKingMoves(square)
// 	}
// }

// // Generate allowed moves for all pieces
// func GenerateAllowedMoves(gs dao.GameState) map[string]uint64 {
// 	moves := make(map[string]uint64)

// 	// Pawns
// 	moves["pawns"] = generatePawnMoves(gs.PawnBitboard, gs.WhiteBitboard|gs.BlackBitboard, gs.EnPassant)

// 	// Knights
// 	moves["knights"] = generatePieceMoves(gs.KnightBitboard, knightMoves, gs.WhiteBitboard)

// 	// Bishops
// 	moves["bishops"] = generateSlidingMoves(gs.BishopBitboard, gs.WhiteBitboard|gs.BlackBitboard, []int{-9, -7, 7, 9})

// 	// Rooks
// 	moves["rooks"] = generateSlidingMoves(gs.RookBitboard, gs.WhiteBitboard|gs.BlackBitboard, []int{-8, -1, 1, 8})

// 	// Queens
// 	moves["queens"] = generateSlidingMoves(gs.QueenBitboard, gs.WhiteBitboard|gs.BlackBitboard, []int{-9, -7, -8, -1, 1, 7, 8, 9})

// 	// Kings
// 	moves["king"] = generatePieceMoves(gs.KingBitboard, kingMoves, gs.WhiteBitboard)

// 	return moves
// }

// func calculateKnightMoves(square int) uint64 {
// 	pos := uint64(1) << square
// 	moves := (pos<<17) & ^AFile |
// 		(pos<<15) & ^HFile |
// 		(pos<<10) & ^(AFile|BFile) |
// 		(pos<<6) & ^(GFile|HFile) |
// 		(pos>>17) & ^HFile |
// 		(pos>>15) & ^AFile |
// 		(pos>>10) & ^(GFile|HFile) |
// 		(pos>>6) & ^(AFile|BFile)
// 	return moves
// }

// func calculateKingMoves(square int) uint64 {
// 	pos := uint64(1) << square
// 	moves := (pos << 8) | (pos >> 8) | (pos<<1) & ^AFile | (pos>>1) & ^HFile |
// 		(pos<<9) & ^AFile | (pos>>9) & ^HFile |
// 		(pos<<7) & ^HFile | (pos>>7) & ^AFile
// 	return moves
// }

// // Generate moves for sliding pieces (rook, bishop, queen)
// func slidingMoves(square int, occupancy uint64, directions []int) uint64 {
// 	var moves uint64
// 	for _, direction := range directions {
// 		sq := square
// 		for {
// 			sq += direction
// 			if sq < 0 || sq >= 64 || (sq%8 == 0 && direction == -1) || (sq%8 == 7 && direction == 1) {
// 				break
// 			}
// 			pos := uint64(1) << sq
// 			moves |= pos
// 			if pos&occupancy != 0 { // Blocked by a piece
// 				break
// 			}
// 		}
// 	}
// 	return moves
// }

// func generatePieceMoves(bitboard uint64, moveTable [64]uint64, friendly uint64) uint64 {
// 	var moves uint64
// 	for bitboard != 0 {
// 		square := bits.TrailingZeros64(bitboard)
// 		moves |= moveTable[square] & ^friendly
// 		bitboard &= bitboard - 1
// 	}
// 	return moves
// }

// func generateSlidingMoves(bitboard, occupancy uint64, directions []int) uint64 {
// 	var moves uint64
// 	for bitboard != 0 {
// 		square := bits.TrailingZeros64(bitboard)
// 		moves |= slidingMoves(square, occupancy, directions)
// 		bitboard &= bitboard - 1
// 	}
// 	return moves
// }

// func generatePawnMoves(pawns, occupancy, enPassant uint64) uint64 {
// 	// Simplified for white pawns, add black and promotions as needed
// 	push := (pawns << 8) & ^occupancy
// 	captures := ((pawns<<7) & ^HFile | (pawns<<9) & ^AFile) & occupancy
// 	return push | captures | enPassant
// }
