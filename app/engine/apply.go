package engine

import (
	"chess-engine/app/domain/dao"
	"strings"
)

// This file holds the single source of truth for how a move changes a position.
//
// There used to be two appliers: ApplyMove (dto-based, full rules) for the
// server path, and simulateMove (bitboard-based, from/to bits only) used by both
// the legality filter and the search. simulateMove silently could not perform
// castling, en passant, or promotion, which meant the legality filter accepted
// illegal en-passant captures and the search played a different game of chess
// than the server did. Both now go through applyBitboardMove.

type pieceKind uint8

const (
	kindNone pieceKind = iota
	kindPawn
	kindKnight
	kindBishop
	kindRook
	kindQueen
	kindKing
)

const (
	rank1Mask = uint64(0x00000000000000FF)
	rank8Mask = uint64(0xFF00000000000000)
)

// Castling squares, as single-bit boards.
var (
	sqA1 = squareBit("a1")
	sqC1 = squareBit("c1")
	sqD1 = squareBit("d1")
	sqE1 = squareBit("e1")
	sqF1 = squareBit("f1")
	sqG1 = squareBit("g1")
	sqH1 = squareBit("h1")
	sqA8 = squareBit("a8")
	sqC8 = squareBit("c8")
	sqD8 = squareBit("d8")
	sqE8 = squareBit("e8")
	sqF8 = squareBit("f8")
	sqG8 = squareBit("g8")
	sqH8 = squareBit("h8")
)

func squareBit(square string) uint64 {
	return uint64(1) << uint(PositionToIndex(square))
}

func kindAt(gs dao.GameState, sq uint64) pieceKind {
	switch {
	case gs.PawnBitboard&sq != 0:
		return kindPawn
	case gs.KnightBitboard&sq != 0:
		return kindKnight
	case gs.BishopBitboard&sq != 0:
		return kindBishop
	case gs.RookBitboard&sq != 0:
		return kindRook
	case gs.QueenBitboard&sq != 0:
		return kindQueen
	case gs.KingBitboard&sq != 0:
		return kindKing
	}
	return kindNone
}

// clearSquare removes whatever occupies sq from every bitboard.
func clearSquare(gs *dao.GameState, sq uint64) {
	mask := ^sq
	gs.WhiteBitboard &= mask
	gs.BlackBitboard &= mask
	gs.PawnBitboard &= mask
	gs.KnightBitboard &= mask
	gs.BishopBitboard &= mask
	gs.RookBitboard &= mask
	gs.QueenBitboard &= mask
	gs.KingBitboard &= mask
}

func placePiece(gs *dao.GameState, sq uint64, kind pieceKind, isWhite bool) {
	if isWhite {
		gs.WhiteBitboard |= sq
	} else {
		gs.BlackBitboard |= sq
	}
	switch kind {
	case kindPawn:
		gs.PawnBitboard |= sq
	case kindKnight:
		gs.KnightBitboard |= sq
	case kindBishop:
		gs.BishopBitboard |= sq
	case kindRook:
		gs.RookBitboard |= sq
	case kindQueen:
		gs.QueenBitboard |= sq
	case kindKing:
		gs.KingBitboard |= sq
	}
}

// promotionKind maps a promotion letter ("q"|"r"|"b"|"n", case-insensitive;
// anything else, including empty, means queen) to its piece kind.
func promotionKind(promotion string) pieceKind {
	switch strings.ToLower(promotion) {
	case "r":
		return kindRook
	case "b":
		return kindBishop
	case "n":
		return kindKnight
	}
	return kindQueen
}

// isPromotionSquare reports whether a pawn of the given colour landing on dst
// promotes.
func isPromotionSquare(dst uint64, isWhite bool) bool {
	if isWhite {
		return dst&rank8Mask != 0
	}
	return dst&rank1Mask != 0
}

// applyBitboardMove plays src->dst and returns the resulting position. It does
// not validate legality, flip the turn, or set LastMove; those belong to the
// callers (ApplyMove for the server path, the search for its own tree).
//
// It handles captures, en-passant captures, castling rook relocation, promotion
// (honouring the requested piece), castling-rights updates including a rook
// captured on its home square, and setting/clearing the en-passant square.
func applyBitboardMove(gs dao.GameState, src, dst uint64, promotion string) dao.GameState {
	kind := kindAt(gs, src)
	if kind == kindNone {
		return gs // nothing on the source square: not a move
	}

	ns := gs
	isWhite := gs.WhiteBitboard&src != 0
	occupied := gs.WhiteBitboard | gs.BlackBitboard

	// En passant. GameState.EnPassant carries two bits -- the target square and
	// the square of the pawn that made the double push -- so when a pawn lands on
	// the (empty) target, the captured pawn is simply the other bit.
	//
	// simulateMove used to skip this entirely. The captured pawn therefore stayed
	// on the board during legality checking and kept blocking its rank, which hid
	// discovered checks and made illegal en-passant captures look legal.
	isEnPassantCapture := kind == kindPawn && dst&gs.EnPassant != 0 && dst&occupied == 0
	if isEnPassantCapture {
		clearSquare(&ns, gs.EnPassant&^dst)
	}

	// Castling: a king moving two files drags its rook across. Reaching g1/c1/g8
	// /c8 from e1/e8 is only ever generated for castling, since a king otherwise
	// moves one square.
	if kind == kindKing {
		switch {
		case src == sqE1 && dst == sqG1:
			relocateRook(&ns, sqH1, sqF1, true)
		case src == sqE1 && dst == sqC1:
			relocateRook(&ns, sqA1, sqD1, true)
		case src == sqE8 && dst == sqG8:
			relocateRook(&ns, sqH8, sqF8, false)
		case src == sqE8 && dst == sqC8:
			relocateRook(&ns, sqA8, sqD8, false)
		}
	}

	clearSquare(&ns, src)
	clearSquare(&ns, dst)

	newKind := kind
	if kind == kindPawn && isPromotionSquare(dst, isWhite) {
		newKind = promotionKind(promotion)
	}
	placePiece(&ns, dst, newKind, isWhite)

	ns.CastlingRights = updatedCastlingRights(gs.CastlingRights, src, dst, kind, isWhite)
	ns.EnPassant = newEnPassantSquare(kind, isWhite, src, dst)

	// Fifty-move rule: the clock counts plies since the last capture or pawn
	// move, and any pawn move or capture (including en passant) resets it.
	if kind == kindPawn || dst&occupied != 0 || isEnPassantCapture {
		ns.HalfmoveClock = 0
	} else {
		ns.HalfmoveClock = gs.HalfmoveClock + 1
	}
	return ns
}

func relocateRook(gs *dao.GameState, from, to uint64, isWhite bool) {
	clearSquare(gs, from)
	placePiece(gs, to, kindRook, isWhite)
}

// updatedCastlingRights removes rights invalidated by this move: the king
// moving, a rook leaving its home square, or anything capturing a rook on its
// home square (the last case was previously not handled at all).
func updatedCastlingRights(rights string, src, dst uint64, kind pieceKind, isWhite bool) string {
	if kind == kindKing {
		if isWhite {
			rights = strings.NewReplacer("K", "", "Q", "").Replace(rights)
		} else {
			rights = strings.NewReplacer("k", "", "q", "").Replace(rights)
		}
	}

	for _, corner := range []struct {
		sq    uint64
		right string
	}{
		{sqA1, "Q"}, {sqH1, "K"}, {sqA8, "q"}, {sqH8, "k"},
	} {
		if src == corner.sq || dst == corner.sq {
			rights = strings.ReplaceAll(rights, corner.right, "")
		}
	}
	return rights
}

// newEnPassantSquare returns the two-bit en-passant marker after this move: the
// square the capturing pawn would move to, plus the pushed pawn's own square.
// Any move that is not a pawn double push clears it.
func newEnPassantSquare(kind pieceKind, isWhite bool, src, dst uint64) uint64 {
	if kind != kindPawn {
		return 0
	}
	if isWhite && dst == src<<16 {
		return (src << 8) | dst
	}
	if !isWhite && dst == src>>16 {
		return (src >> 8) | dst
	}
	return 0
}
