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

	pieceBitboards := map[string]struct {
		pieceBitboard *uint64
		colorBitboard *uint64
	}{
		"B": {&game.State.BishopBitboard, &game.State.WhiteBitboard},
		"N": {&game.State.KnightBitboard, &game.State.WhiteBitboard},
		"R": {&game.State.RookBitboard, &game.State.WhiteBitboard},
		"Q": {&game.State.QueenBitboard, &game.State.WhiteBitboard},
		"K": {&game.State.KingBitboard, &game.State.WhiteBitboard},
		"P": {&game.State.PawnBitboard, &game.State.WhiteBitboard},
		"b": {&game.State.BishopBitboard, &game.State.BlackBitboard},
		"n": {&game.State.KnightBitboard, &game.State.BlackBitboard},
		"r": {&game.State.RookBitboard, &game.State.BlackBitboard},
		"q": {&game.State.QueenBitboard, &game.State.BlackBitboard},
		"k": {&game.State.KingBitboard, &game.State.BlackBitboard},
		"p": {&game.State.PawnBitboard, &game.State.BlackBitboard},
	}

	if move.Piece == "K" || move.Piece == "k" {

		if move.Source == "e1" && move.Destination == "g1" && strings.Contains(game.State.CastlingRights, "K") {

			*pieceBitboards["R"].colorBitboard &= ^(1 << PositionToIndex("h1"))
			*pieceBitboards["R"].pieceBitboard &= ^(1 << PositionToIndex("h1"))
			*pieceBitboards["R"].colorBitboard |= (1 << PositionToIndex("f1"))
			*pieceBitboards["R"].pieceBitboard |= (1 << PositionToIndex("f1"))
		}

		if move.Source == "e1" && move.Destination == "c1" && strings.Contains(game.State.CastlingRights, "Q") {

			*pieceBitboards["R"].colorBitboard &= ^(1 << PositionToIndex("a1"))
			*pieceBitboards["R"].pieceBitboard &= ^(1 << PositionToIndex("a1"))
			*pieceBitboards["R"].colorBitboard |= (1 << PositionToIndex("d1"))
			*pieceBitboards["R"].pieceBitboard |= (1 << PositionToIndex("d1"))
		}

		if move.Source == "e8" && move.Destination == "g8" && strings.Contains(game.State.CastlingRights, "k") {

			*pieceBitboards["r"].colorBitboard &= ^(1 << PositionToIndex("h8"))
			*pieceBitboards["r"].pieceBitboard &= ^(1 << PositionToIndex("h8"))
			*pieceBitboards["r"].colorBitboard |= (1 << PositionToIndex("f8"))
			*pieceBitboards["r"].pieceBitboard |= (1 << PositionToIndex("f8"))
		}

		if move.Source == "e8" && move.Destination == "c8" && strings.Contains(game.State.CastlingRights, "q") {

			*pieceBitboards["r"].colorBitboard &= ^(1 << PositionToIndex("a8"))
			*pieceBitboards["r"].pieceBitboard &= ^(1 << PositionToIndex("a8"))
			*pieceBitboards["r"].colorBitboard |= (1 << PositionToIndex("d8"))
			*pieceBitboards["r"].pieceBitboard |= (1 << PositionToIndex("d8"))
		}

		if game.State.Turn == "w" {
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "K", "")
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "Q", "")
		} else {
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "k", "")
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "q", "")
		}
	}

	if move.Piece == "R" {
		if move.Source == "a1" {
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "Q", "")
		} else if move.Source == "h1" {
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "K", "")
		}
	} else if move.Piece == "r" {
		if move.Source == "a8" {
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "q", "")
		} else if move.Source == "h8" {
			game.State.CastlingRights = strings.ReplaceAll(game.State.CastlingRights, "k", "")
		}
	}

	if move.Piece == "P" || move.Piece == "p" { // Handle en passant capture
		if (game.State.EnPassant > 0) && (1<<destinationIdx&game.State.EnPassant != 0) {
			captureSquareIdx := PositionToIndex(string(move.Destination[0]) + string(move.Source[1])) // Captured pawn's square

			if move.Piece == "P" { // White pawn captures black pawn
				*pieceBitboards["p"].colorBitboard &= ^(1 << captureSquareIdx)
				*pieceBitboards["p"].pieceBitboard &= ^(1 << captureSquareIdx)
			} else if move.Piece == "p" { // Black pawn captures white pawn
				*pieceBitboards["P"].colorBitboard &= ^(1 << captureSquareIdx)
				*pieceBitboards["P"].pieceBitboard &= ^(1 << captureSquareIdx)
			}
		}

		// Set en passant square for potential future capture
		if move.Source[1] == '2' && move.Destination[1] == '4' { // White two-square move
			game.State.EnPassant = 1 << PositionToIndex(string(move.Destination[0])+"3")
		} else if move.Source[1] == '7' && move.Destination[1] == '5' { // Black two-square move
			game.State.EnPassant = 1 << PositionToIndex(string(move.Destination[0])+"6")
		} else {
			game.State.EnPassant = 0 // Clear en passant
		}
	} else {
		game.State.EnPassant = 0 // Clear en passant for non-pawn moves
	}

	if game.State.WhiteBitboard&(1<<destinationIdx) != 0 {

		if game.State.BishopBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["B"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["B"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.KnightBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["N"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["N"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.RookBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["R"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["R"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.QueenBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["Q"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["Q"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.KingBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["K"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["K"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.PawnBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["P"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["P"].pieceBitboard &= ^(1 << destinationIdx)
		}
	}

	if game.State.BlackBitboard&(1<<destinationIdx) != 0 {

		if game.State.BishopBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["b"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["b"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.KnightBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["n"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["n"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.RookBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["r"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["r"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.QueenBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["q"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["q"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.KingBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["k"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["k"].pieceBitboard &= ^(1 << destinationIdx)
		}
		if game.State.PawnBitboard&(1<<destinationIdx) != 0 {
			*pieceBitboards["p"].colorBitboard &= ^(1 << destinationIdx)
			*pieceBitboards["p"].pieceBitboard &= ^(1 << destinationIdx)
		}
	}

	*pieceBitboards[move.Piece].colorBitboard &= ^(1 << sourceIdx)
	*pieceBitboards[move.Piece].pieceBitboard &= ^(1 << sourceIdx)

	*pieceBitboards[move.Piece].pieceBitboard |= (1 << destinationIdx)
	*pieceBitboards[move.Piece].colorBitboard |= (1 << destinationIdx)

	// Pawn promotion
	if (move.Piece == "P" && move.Destination[1] == '8') || (move.Piece == "p" && move.Destination[1] == '1') {
		// Remove pawn from bitboards
		*pieceBitboards[move.Piece].colorBitboard &= ^(1 << destinationIdx)
		*pieceBitboards[move.Piece].pieceBitboard &= ^(1 << destinationIdx)

		// Add queen to bitboards
		if move.Piece == "P" { // Promote white pawn to queen
			*pieceBitboards["Q"].colorBitboard |= (1 << destinationIdx)
			*pieceBitboards["Q"].pieceBitboard |= (1 << destinationIdx)
		} else if move.Piece == "p" { // Promote black pawn to queen
			*pieceBitboards["q"].colorBitboard |= (1 << destinationIdx)
			*pieceBitboards["q"].pieceBitboard |= (1 << destinationIdx)
		}
	}

	game.State.LastMove = move.Piece + move.Source + move.Destination

	game.State.Turn = ToggleTurn(game.State.Turn)

	return game.State.LastMove, nil
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

func ConvertToStandardNotation(move string, isCapture bool, isCastling bool) string {
	// Special case for castling
	if isCastling {
		if move == "O-O" {
			return "O-O"
		} else if move == "O-O-O" {
			return "O-O-O"
		}
	}

	// Extract components of the move
	piece := move[:1] // First character
	from := move[1:3] // Next two characters
	to := move[3:]    // Last two characters

	// Handle pawn moves (no piece designation for pawns)
	if piece == "" || piece == "P" {
		if isCapture {
			return string(from[0]) + "x" + to // Use the file of the pawn for capture
		}
		return to
	}

	// Handle standard piece moves
	if isCapture {
		return piece + "x" + to
	}

	return piece + to
}
