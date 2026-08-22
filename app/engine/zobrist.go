package engine

import (
	"chess-engine/app/domain/dao"
	"math/bits"
	"strings"
)

// Zobrist hashing gives every position a 64-bit key, which is what makes
// repetition detection possible: two positions repeat when their keys match.
//
// The keys are generated from a fixed seed, never from time or crypto/rand. A
// key table that changed between runs would make the engine's behaviour
// irreproducible, which is exactly the property the deterministic move ordering
// exists to provide.
const zobristSeed = 0x9E3779B97F4A7C15

var (
	// zPiece is indexed [colour][kind][square]; colour 0 is white.
	zPiece     [2][kindKing + 1][64]uint64
	zSideBlack uint64
	// zCastle is indexed by position in "KQkq".
	zCastle [4]uint64
	// zEnPassant is indexed by the file (0-7) of the en-passant target square.
	zEnPassant [8]uint64
)

func init() {
	state := uint64(zobristSeed)
	next := func() uint64 {
		// splitmix64: small, fast, and deterministic.
		state += 0x9E3779B97F4A7C15
		z := state
		z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
		z = (z ^ (z >> 27)) * 0x94D049BB133111EB
		return z ^ (z >> 31)
	}

	for colour := 0; colour < 2; colour++ {
		for kind := kindPawn; kind <= kindKing; kind++ {
			for sq := 0; sq < 64; sq++ {
				zPiece[colour][kind][sq] = next()
			}
		}
	}
	zSideBlack = next()
	for i := range zCastle {
		zCastle[i] = next()
	}
	for i := range zEnPassant {
		zEnPassant[i] = next()
	}
}

// PositionKey returns the Zobrist key for a position: piece placement, side to
// move, castling rights, and the en-passant file.
//
// The key is recomputed from scratch rather than updated incrementally. That is
// ~32 XORs per call, which is negligible next to this engine's real cost -- it
// regenerates the full legal move list at every node.
func PositionKey(gs dao.GameState) uint64 {
	var key uint64

	boards := [...]struct {
		bb   uint64
		kind pieceKind
	}{
		{gs.PawnBitboard, kindPawn},
		{gs.KnightBitboard, kindKnight},
		{gs.BishopBitboard, kindBishop},
		{gs.RookBitboard, kindRook},
		{gs.QueenBitboard, kindQueen},
		{gs.KingBitboard, kindKing},
	}
	for _, b := range boards {
		for w := b.bb & gs.WhiteBitboard; w != 0; w &= w - 1 {
			key ^= zPiece[0][b.kind][bits.TrailingZeros64(w&-w)]
		}
		for bl := b.bb & gs.BlackBitboard; bl != 0; bl &= bl - 1 {
			key ^= zPiece[1][b.kind][bits.TrailingZeros64(bl&-bl)]
		}
	}

	if gs.Turn == "b" {
		key ^= zSideBlack
	}

	for i, right := range []string{"K", "Q", "k", "q"} {
		if strings.Contains(gs.CastlingRights, right) {
			key ^= zCastle[i]
		}
	}

	// The en-passant file is folded in whenever the marker is set, without first
	// checking that a capture is actually available. That can distinguish two
	// otherwise identical positions, so it may miss a repetition -- it can never
	// invent one, which is the safe direction to err.
	if file, ok := enPassantFile(gs.EnPassant); ok {
		key ^= zEnPassant[file]
	}

	return key
}

// enPassantFile returns the file of the en-passant target square. GameState
// stores two bits (target plus the pushed pawn); the target is the one on rank 3
// or rank 6.
func enPassantFile(enPassant uint64) (int, bool) {
	for b := enPassant; b != 0; b &= b - 1 {
		idx := bits.TrailingZeros64(b & -b)
		if rank := idx / 8; rank == 2 || rank == 5 {
			return idx % 8, true
		}
	}
	return 0, false
}
