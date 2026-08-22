package engine

import (
	"chess-engine/app/domain/dao"
	"math/bits"
)

// Position evaluation.
//
// Material alone cannot distinguish a good position from a bad one, so every
// quiet move looked identical to the search: it shuffled pieces, and extra
// depth bought nothing. Adding a transposition table cut node counts by 64% and
// gained zero Elo for exactly that reason.
//
// Piece-square tables are the classic fix and by far the best strength per line
// of code: they encode "knights belong in the centre, rooks on the seventh, the
// king behind pawns in the middlegame and active in the endgame". Values are the
// widely published Simplified Evaluation Function tables.

// Tables are written rank 8 first so they read like a board from White's side.
// initPST flips them into this engine's indexing, where square 0 is a1.
var (
	pawnPST = [64]int{
		0, 0, 0, 0, 0, 0, 0, 0,
		50, 50, 50, 50, 50, 50, 50, 50,
		10, 10, 20, 30, 30, 20, 10, 10,
		5, 5, 10, 25, 25, 10, 5, 5,
		0, 0, 0, 20, 20, 0, 0, 0,
		5, -5, -10, 0, 0, -10, -5, 5,
		5, 10, 10, -20, -20, 10, 10, 5,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	knightPST = [64]int{
		-50, -40, -30, -30, -30, -30, -40, -50,
		-40, -20, 0, 0, 0, 0, -20, -40,
		-30, 0, 10, 15, 15, 10, 0, -30,
		-30, 5, 15, 20, 20, 15, 5, -30,
		-30, 0, 15, 20, 20, 15, 0, -30,
		-30, 5, 10, 15, 15, 10, 5, -30,
		-40, -20, 0, 5, 5, 0, -20, -40,
		-50, -40, -30, -30, -30, -30, -40, -50,
	}
	bishopPST = [64]int{
		-20, -10, -10, -10, -10, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 10, 10, 5, 0, -10,
		-10, 5, 5, 10, 10, 5, 5, -10,
		-10, 0, 10, 10, 10, 10, 0, -10,
		-10, 10, 10, 10, 10, 10, 10, -10,
		-10, 5, 0, 0, 0, 0, 5, -10,
		-20, -10, -10, -10, -10, -10, -10, -20,
	}
	rookPST = [64]int{
		0, 0, 0, 0, 0, 0, 0, 0,
		5, 10, 10, 10, 10, 10, 10, 5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		-5, 0, 0, 0, 0, 0, 0, -5,
		0, 0, 0, 5, 5, 0, 0, 0,
	}
	queenPST = [64]int{
		-20, -10, -10, -5, -5, -10, -10, -20,
		-10, 0, 0, 0, 0, 0, 0, -10,
		-10, 0, 5, 5, 5, 5, 0, -10,
		-5, 0, 5, 5, 5, 5, 0, -5,
		0, 0, 5, 5, 5, 5, 0, -5,
		-10, 5, 5, 5, 5, 5, 0, -10,
		-10, 0, 5, 0, 0, 0, 0, -10,
		-20, -10, -10, -5, -5, -10, -10, -20,
	}
	kingMiddlegamePST = [64]int{
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-30, -40, -40, -50, -50, -40, -40, -30,
		-20, -30, -30, -40, -40, -30, -30, -20,
		-10, -20, -20, -20, -20, -20, -20, -10,
		20, 20, 0, 0, 0, 0, 20, 20,
		20, 30, 10, 0, 0, 10, 30, 20,
	}
	kingEndgamePST = [64]int{
		-50, -40, -30, -20, -20, -30, -40, -50,
		-30, -20, -10, 0, 0, -10, -20, -30,
		-30, -10, 20, 30, 30, 20, -10, -30,
		-30, -10, 30, 40, 40, 30, -10, -30,
		-30, -10, 30, 40, 40, 30, -10, -30,
		-30, -10, 20, 30, 30, 20, -10, -30,
		-30, -30, 0, 0, 0, 0, -30, -30,
		-50, -30, -30, -30, -30, -30, -30, -50,
	}
)

// White-oriented tables in engine indexing (square 0 = a1), plus their vertical
// mirrors for Black.
var (
	pstWhite [kindKing + 1][64]int
	pstBlack [kindKing + 1][64]int
	kingMGW  [64]int
	kingMGB  [64]int
	kingEGW  [64]int
	kingEGB  [64]int
)

func init() {
	load := func(src [64]int, white *[64]int, black *[64]int) {
		for sq := 0; sq < 64; sq++ {
			// Source row 0 is rank 8; engine square 0 is a1.
			white[sq] = src[(7-sq/8)*8+sq%8]
		}
		for sq := 0; sq < 64; sq++ {
			black[sq] = white[sq^56] // mirror the rank
		}
	}
	load(pawnPST, &pstWhite[kindPawn], &pstBlack[kindPawn])
	load(knightPST, &pstWhite[kindKnight], &pstBlack[kindKnight])
	load(bishopPST, &pstWhite[kindBishop], &pstBlack[kindBishop])
	load(rookPST, &pstWhite[kindRook], &pstBlack[kindRook])
	load(queenPST, &pstWhite[kindQueen], &pstBlack[kindQueen])
	load(kingMiddlegamePST, &kingMGW, &kingMGB)
	load(kingEndgamePST, &kingEGW, &kingEGB)
}

const (
	bishopPairBonus     = 30
	doubledPawnPenalty  = 15
	isolatedPawnPenalty = 15
	rookOpenFileBonus   = 20
	tempoBonus          = 10
	// maxPhase is the phase weight of a full set of pieces; it drives the
	// middlegame/endgame blend of the king tables.
	maxPhase = 24
)

var fileMasks = func() [8]uint64 {
	var f [8]uint64
	for i := 0; i < 8; i++ {
		f[i] = fileAMask << uint(i)
	}
	return f
}()

// gamePhase is 24 with all pieces on the board and 0 in a bare-king endgame.
func gamePhase(gs dao.GameState) int {
	phase := bits.OnesCount64(gs.KnightBitboard) +
		bits.OnesCount64(gs.BishopBitboard) +
		2*bits.OnesCount64(gs.RookBitboard) +
		4*bits.OnesCount64(gs.QueenBitboard)
	if phase > maxPhase {
		phase = maxPhase
	}
	return phase
}

// evaluate scores a position in centipawns from White's perspective.
func evaluate(gs dao.GameState) int {
	score := evalMaterial(gs)
	score += pieceSquareScore(gs)
	score += pawnStructureScore(gs)
	score += bishopPairScore(gs)
	score += rookFileScore(gs)

	// A small bonus for having the move; without it the engine sees perfectly
	// symmetrical positions as dead equal and is indifferent to losing a tempo.
	if gs.Turn == "b" {
		score -= tempoBonus
	} else {
		score += tempoBonus
	}
	return score
}

func pieceSquareScore(gs dao.GameState) int {
	score := 0
	for _, b := range [...]struct {
		bb   uint64
		kind pieceKind
	}{
		{gs.PawnBitboard, kindPawn},
		{gs.KnightBitboard, kindKnight},
		{gs.BishopBitboard, kindBishop},
		{gs.RookBitboard, kindRook},
		{gs.QueenBitboard, kindQueen},
	} {
		for w := b.bb & gs.WhiteBitboard; w != 0; w &= w - 1 {
			score += pstWhite[b.kind][bits.TrailingZeros64(w&-w)]
		}
		for bl := b.bb & gs.BlackBitboard; bl != 0; bl &= bl - 1 {
			score -= pstBlack[b.kind][bits.TrailingZeros64(bl&-bl)]
		}
	}

	// The king's ideal square inverts between the opening and the endgame, so
	// its table is blended by game phase rather than picked outright.
	phase := gamePhase(gs)
	blend := func(mg, eg int) int {
		return (mg*phase + eg*(maxPhase-phase)) / maxPhase
	}
	if wk := gs.KingBitboard & gs.WhiteBitboard; wk != 0 {
		sq := bits.TrailingZeros64(wk)
		score += blend(kingMGW[sq], kingEGW[sq])
	}
	if bk := gs.KingBitboard & gs.BlackBitboard; bk != 0 {
		sq := bits.TrailingZeros64(bk)
		score -= blend(kingMGB[sq], kingEGB[sq])
	}
	return score
}

// pawnStructureScore penalises doubled and isolated pawns.
func pawnStructureScore(gs dao.GameState) int {
	white := gs.PawnBitboard & gs.WhiteBitboard
	black := gs.PawnBitboard & gs.BlackBitboard

	side := func(pawns uint64) int {
		penalty := 0
		for f := 0; f < 8; f++ {
			onFile := bits.OnesCount64(pawns & fileMasks[f])
			if onFile == 0 {
				continue
			}
			if onFile > 1 {
				penalty += doubledPawnPenalty * (onFile - 1)
			}
			var neighbours uint64
			if f > 0 {
				neighbours |= fileMasks[f-1]
			}
			if f < 7 {
				neighbours |= fileMasks[f+1]
			}
			if pawns&neighbours == 0 {
				penalty += isolatedPawnPenalty * onFile
			}
		}
		return penalty
	}
	return side(black) - side(white)
}

func bishopPairScore(gs dao.GameState) int {
	score := 0
	if bits.OnesCount64(gs.BishopBitboard&gs.WhiteBitboard) >= 2 {
		score += bishopPairBonus
	}
	if bits.OnesCount64(gs.BishopBitboard&gs.BlackBitboard) >= 2 {
		score -= bishopPairBonus
	}
	return score
}

// rookFileScore rewards rooks on files with no pawns at all.
func rookFileScore(gs dao.GameState) int {
	score := 0
	for f := 0; f < 8; f++ {
		if gs.PawnBitboard&fileMasks[f] != 0 {
			continue
		}
		score += rookOpenFileBonus * bits.OnesCount64(gs.RookBitboard&gs.WhiteBitboard&fileMasks[f])
		score -= rookOpenFileBonus * bits.OnesCount64(gs.RookBitboard&gs.BlackBitboard&fileMasks[f])
	}
	return score
}
