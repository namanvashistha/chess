package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"fmt"

	log "github.com/sirupsen/logrus"
)

func ProcessMove(game *dao.ChessGame, move dto.Move, user dao.User) error {

	if user.ID != game.WhiteUser.ID && user.ID != game.BlackUser.ID {
		log.Errorf("Invalid move: user %d is not in the game", user.ID)
		return fmt.Errorf("invalid move: user %d is not in the game", user.ID)
	}

	if game.State.Turn == "w" && user.ID != game.WhiteUser.ID {
		log.Errorf("Invalid move: user %d is not white", user.ID)
		return fmt.Errorf("invalid move: user %d is not white", user.ID)
	} else if game.State.Turn == "b" && user.ID != game.BlackUser.ID {
		log.Errorf("Invalid move: user %d is not black", user.ID)
		return fmt.Errorf("invalid move: user %d is not black", user.ID)
	}

	if game.State.Turn == "w" && !(int(move.Piece[0]) >= 65 && int(move.Piece[0]) <= 90) {
		log.Errorf("Invalid move: user %d is not white", user.ID)
		return fmt.Errorf("invalid move: user %d is not white", user.ID)
	}
	if game.State.Turn == "b" && !(int(move.Piece[0]) >= 97 && int(move.Piece[0]) <= 122) {
		log.Errorf("Invalid move: user %d is not black", user.ID)
		return fmt.Errorf("invalid move: user %d is not black", user.ID)
	}

	sourceIdx := PositionToIndex(move.Source)
	destinationIdx := PositionToIndex(move.Destination)

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

	game.State.LastMove = move.Piece + move.Source + move.Destination

	game.State.Turn = ToggleTurn(game.State.Turn)

	return nil
}

func UpdateBitboard(bitboard uint64, sourceIdx, destIdx int) uint64 {

	bitboard &= ^(1 << sourceIdx)

	bitboard |= (1 << destIdx)
	return bitboard
}

func ToggleTurn(currentTurn string) string {
	if currentTurn == "w" {
		return "b"
	}
	return "w"
}
