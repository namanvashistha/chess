package engine

import (
	"chess-engine/app/domain/dao"

	log "github.com/sirupsen/logrus"
)

func GenerateLegalMovesForAllPositions(gs dao.GameState) map[uint64]uint64 {
	pseudo_legal_moves := GeneratePseudoLegalMoves(gs)
	legal_moves := filterLegalMoves(gs, pseudo_legal_moves)

	isWhiteKingInCheck, _ := CheckIfKingIsInCheck(gs, pseudo_legal_moves, true)
	if isWhiteKingInCheck {
		log.Println("WHITE KING IS IN CHECK")
	}

	isBlackKingInCheck, _ := CheckIfKingIsInCheck(gs, pseudo_legal_moves, false)
	if isBlackKingInCheck {
		log.Println("BLACK KING IS IN CHECK")
	}

	return legal_moves
}

func filterLegalMoves(gs dao.GameState, pseudoLegalMoves map[uint64]uint64) map[uint64]uint64 {
	legalMoves := make(map[uint64]uint64)

	for piece, moves := range pseudoLegalMoves {
		for move := moves; move != 0; move &= move - 1 {
			movePosition := move & -move

			simulatedGameState := simulateMove(gs, piece, movePosition)
			isWhite := gs.WhiteBitboard&piece != 0
			if !isKingInCheck(simulatedGameState, isWhite) {
				legalMoves[piece] |= movePosition
			}
		}
	}
	return legalMoves
}

func simulateMove(gs dao.GameState, piece uint64, move uint64) dao.GameState {
	newGameState := gs

	if gs.WhiteBitboard&piece != 0 {
		newGameState.WhiteBitboard &= ^piece
	} else {
		newGameState.BlackBitboard &= ^piece
	}
	newGameState.PawnBitboard &= ^piece
	newGameState.KnightBitboard &= ^piece
	newGameState.BishopBitboard &= ^piece
	newGameState.RookBitboard &= ^piece
	newGameState.QueenBitboard &= ^piece
	newGameState.KingBitboard &= ^piece

	newGameState.WhiteBitboard &= ^move
	newGameState.BlackBitboard &= ^move
	newGameState.PawnBitboard &= ^move
	newGameState.KnightBitboard &= ^move
	newGameState.BishopBitboard &= ^move
	newGameState.RookBitboard &= ^move
	newGameState.QueenBitboard &= ^move
	newGameState.KingBitboard &= ^move

	if gs.WhiteBitboard&piece != 0 {
		newGameState.WhiteBitboard |= move
	} else {
		newGameState.BlackBitboard |= move
	}
	switch {
	case gs.PawnBitboard&piece != 0:
		newGameState.PawnBitboard |= move
	case gs.KnightBitboard&piece != 0:
		newGameState.KnightBitboard |= move
	case gs.BishopBitboard&piece != 0:
		newGameState.BishopBitboard |= move
	case gs.RookBitboard&piece != 0:
		newGameState.RookBitboard |= move
	case gs.QueenBitboard&piece != 0:
		newGameState.QueenBitboard |= move
	case gs.KingBitboard&piece != 0:
		newGameState.KingBitboard |= move
	}
	return newGameState
}

func isKingInCheck(gs dao.GameState, isWhiteKing bool) bool {
	pseudo_legal_moves := GeneratePseudoLegalMoves(gs)

	isInCheck, _ := CheckIfKingIsInCheck(gs, pseudo_legal_moves, isWhiteKing)
	return isInCheck
}

func GeneratePseudoLegalMoves(gs dao.GameState) map[uint64]uint64 {
	pseudo_legal_moves := make(map[uint64]uint64)

	generatePawnMoves(gs, pseudo_legal_moves, make(map[uint64]uint64))
	generateKnightMoves(gs, pseudo_legal_moves, make(map[uint64]uint64))
	generateBishopMoves(gs, pseudo_legal_moves, make(map[uint64]uint64))
	generateRookMoves(gs, pseudo_legal_moves, make(map[uint64]uint64))
	generateQueenMoves(gs, pseudo_legal_moves, make(map[uint64]uint64))
	generateKingMoves(gs, pseudo_legal_moves, make(map[uint64]uint64))

	return pseudo_legal_moves
}

func generatePawnMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {

	for whitePawnBitboard := (gs.PawnBitboard & gs.WhiteBitboard); whitePawnBitboard != 0; whitePawnBitboard &= whitePawnBitboard - 1 {

		piece := whitePawnBitboard & -whitePawnBitboard
		pawnMoves := WhitePawnAttackBitboard[piece]
		pawnMoves &= ^gs.WhiteBitboard

		legal_moves[piece] = pawnMoves
		pseudo_legal_moves[piece] = pawnMoves
	}

	for blackPawnBitboard := (gs.PawnBitboard & gs.BlackBitboard); blackPawnBitboard != 0; blackPawnBitboard &= blackPawnBitboard - 1 {

		piece := blackPawnBitboard & -blackPawnBitboard
		pawnMoves := BlackPawnAttackBitboard[piece]
		pawnMoves &= ^gs.BlackBitboard

		legal_moves[piece] = pawnMoves
		pseudo_legal_moves[piece] = pawnMoves
	}

}

func generateKnightMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {

	for knightBitboard := gs.KnightBitboard; knightBitboard != 0; knightBitboard &= knightBitboard - 1 {

		piece := knightBitboard & -knightBitboard

		knightMoves := KnightAttackBitboard[piece]

		if gs.WhiteBitboard&piece != 0 {

			knightMoves &= ^gs.WhiteBitboard
		} else if gs.BlackBitboard&piece != 0 {

			knightMoves &= ^gs.BlackBitboard
		}

		legal_moves[piece] = knightMoves
		pseudo_legal_moves[piece] = knightMoves
	}
}

func generateBishopMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {

	allOccupied := gs.WhiteBitboard | gs.BlackBitboard

	for bishopBitboard := gs.BishopBitboard; bishopBitboard != 0; bishopBitboard &= bishopBitboard - 1 {

		piece := bishopBitboard & -bishopBitboard

		bishopMoves := BishopAttackBitboard[piece]
		pseudo_legal_moves[piece] = bishopMoves

		rayDirections := []int{-9, -7, 7, 9}

		bishopMoves = removeBlockedMoves(piece, bishopMoves, allOccupied, rayDirections)

		if gs.WhiteBitboard&piece != 0 {

			blackKingBitboard := gs.BlackBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.WhiteBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^blackKingBitboard, rayDirections)
			bishopMoves &= ^gs.WhiteBitboard

		} else if gs.BlackBitboard&piece != 0 {

			whiteKingBitboard := gs.WhiteBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.BlackBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^whiteKingBitboard, rayDirections)
			bishopMoves &= ^gs.BlackBitboard

		}

		legal_moves[piece] = bishopMoves
	}
}

func generateRookMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {

	allOccupied := gs.WhiteBitboard | gs.BlackBitboard

	for rookBitboard := gs.RookBitboard; rookBitboard != 0; rookBitboard &= rookBitboard - 1 {

		piece := rookBitboard & -rookBitboard

		rookMoves := RookAttackBitboard[piece]
		pseudo_legal_moves[piece] = rookMoves

		rayDirections := []int{-8, 8, -1, 1}
		rookMoves = removeBlockedMoves(piece, rookMoves, allOccupied, rayDirections)

		if gs.WhiteBitboard&piece != 0 {
			blackKingBitboard := gs.BlackBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.WhiteBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^blackKingBitboard, rayDirections)
			rookMoves &= ^gs.WhiteBitboard
		} else if gs.BlackBitboard&piece != 0 {

			whiteKingBitboard := gs.WhiteBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.BlackBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^whiteKingBitboard, rayDirections)
			rookMoves &= ^gs.BlackBitboard
		}

		legal_moves[piece] = rookMoves
	}
}

func generateQueenMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {

	allOccupied := gs.WhiteBitboard | gs.BlackBitboard

	for queenBitboard := gs.QueenBitboard; queenBitboard != 0; queenBitboard &= queenBitboard - 1 {

		piece := queenBitboard & -queenBitboard

		queenMoves := BishopAttackBitboard[piece] | RookAttackBitboard[piece]
		pseudo_legal_moves[piece] = queenMoves

		rayDirections := []int{-8, 8, -1, 1, -9, -7, 7, 9}
		queenMoves = removeBlockedMoves(piece, queenMoves, allOccupied, rayDirections)

		if gs.WhiteBitboard&piece != 0 {

			blackKingBitboard := gs.BlackBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.WhiteBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^blackKingBitboard, rayDirections)
			queenMoves &= ^gs.WhiteBitboard
		} else if gs.BlackBitboard&piece != 0 {

			whiteKingBitboard := gs.WhiteBitboard & gs.KingBitboard
			pseudo_legal_moves[piece] = ^gs.BlackBitboard & removeBlockedMoves(piece, pseudo_legal_moves[piece], allOccupied&^whiteKingBitboard, rayDirections)
			queenMoves &= ^gs.BlackBitboard
		}

		legal_moves[piece] = queenMoves
	}
}

func generateKingMoves(gs dao.GameState, pseudo_legal_moves map[uint64]uint64, legal_moves map[uint64]uint64) {
	for kingBitboard := gs.KingBitboard; kingBitboard != 0; kingBitboard &= kingBitboard - 1 {
		piece := kingBitboard & -kingBitboard

		kingMoves := KingAttackBitboard[piece]

		if gs.WhiteBitboard&piece != 0 {
			kingMoves &= ^gs.WhiteBitboard
			attackedSquares := getAttackedSquares(gs.BlackBitboard, pseudo_legal_moves)
			kingMoves &= ^attackedSquares
		} else if gs.BlackBitboard&piece != 0 {
			kingMoves &= ^gs.BlackBitboard
			attackedSquares := getAttackedSquares(gs.WhiteBitboard, pseudo_legal_moves)
			kingMoves &= ^attackedSquares
		}

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

	for _, direction := range rayDirections {
		blockedMoves |= traceRay(piece, direction, allOccupied)
	}

	return moves & blockedMoves
}

func traceRay(start uint64, direction int, allOccupied uint64) uint64 {
	blocked := uint64(0)
	ray := start

	for {
		if direction > 0 {
			ray <<= direction
		} else {
			ray >>= -direction
		}

		if ray == 0 || crossesBoundary(ray, direction) {
			break
		}

		blocked |= ray
		if ray&allOccupied != 0 {
			break
		}
	}

	return blocked
}

func crossesBoundary(bitboard uint64, direction int) bool {
	if direction == -9 || direction == 7 {
		return (bitboard & 0x8080808080808080) != 0
	} else if direction == -7 || direction == 9 {
		return (bitboard & 0x0101010101010101) != 0
	}
	return false
}

func CheckIfKingIsInCheck(gs dao.GameState, moves map[uint64]uint64, isWhiteKing bool) (bool, uint64) {
	var kingBitboard uint64
	if isWhiteKing {
		kingBitboard = gs.KingBitboard & gs.WhiteBitboard
	} else {
		kingBitboard = gs.KingBitboard & gs.BlackBitboard
	}

	kingPosition := kingBitboard & -kingBitboard

	for piece, attackSet := range moves {
		if (gs.WhiteBitboard&piece != 0 && isWhiteKing) || (gs.BlackBitboard&piece != 0 && !isWhiteKing) {
			continue
		}

		if attackSet&kingPosition != 0 {
			return true, piece
		}
	}

	return false, 0
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
