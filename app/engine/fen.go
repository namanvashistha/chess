package engine

import (
	"chess-engine/app/domain/dao"
	"fmt"
	"strconv"
	"strings"
)

// StartFEN is the standard chess starting position in Forsyth–Edwards Notation.
const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// fenPieceBit returns a pointer to the (color, piece) bitboards a FEN piece
// letter belongs to, or false if the letter is not a valid piece.
func fenPieceBit(c byte, gs *dao.GameState) (color, piece *uint64, ok bool) {
	white := c >= 'A' && c <= 'Z'
	colorBB := &gs.BlackBitboard
	if white {
		colorBB = &gs.WhiteBitboard
	}
	switch c {
	case 'P', 'p':
		return colorBB, &gs.PawnBitboard, true
	case 'R', 'r':
		return colorBB, &gs.RookBitboard, true
	case 'N', 'n':
		return colorBB, &gs.KnightBitboard, true
	case 'B', 'b':
		return colorBB, &gs.BishopBitboard, true
	case 'Q', 'q':
		return colorBB, &gs.QueenBitboard, true
	case 'K', 'k':
		return colorBB, &gs.KingBitboard, true
	}
	return nil, nil, false
}

// ParseFEN parses a FEN string into a GameState. Halfmove/fullmove counters are
// accepted but discarded (GameState tracks no draw counters). The en-passant
// target is expanded into the engine's two-bit convention (see ToFEN/ProcessMove):
// the target square plus the just-pushed pawn's square, so the move generator's
// `EnPassant & opponentBitboard` guard fires.
func ParseFEN(fen string) (dao.GameState, error) {
	var gs dao.GameState
	fields := strings.Fields(fen)
	if len(fields) < 4 {
		return gs, fmt.Errorf("invalid FEN: expected at least 4 fields, got %d", len(fields))
	}

	// Field 1: piece placement, rank 8 (top) first, file a (left) first.
	ranks := strings.Split(fields[0], "/")
	if len(ranks) != 8 {
		return gs, fmt.Errorf("invalid FEN: expected 8 ranks, got %d", len(ranks))
	}
	for r, rankStr := range ranks {
		file := 0
		for i := 0; i < len(rankStr); i++ {
			c := rankStr[i]
			if c >= '1' && c <= '8' {
				file += int(c - '0')
				continue
			}
			color, piece, ok := fenPieceBit(c, &gs)
			if !ok {
				return gs, fmt.Errorf("invalid FEN: bad piece %q in rank %d", string(c), 8-r)
			}
			if file > 7 {
				return gs, fmt.Errorf("invalid FEN: rank %d overflows", 8-r)
			}
			// FEN rank 0 is board rank 8; board index 0 is a1.
			idx := (7-r)*8 + file
			bit := uint64(1) << uint(idx)
			*color |= bit
			*piece |= bit
			file++
		}
		if file != 8 {
			return gs, fmt.Errorf("invalid FEN: rank %d has %d files", 8-r, file)
		}
	}

	// Field 2: side to move.
	switch fields[1] {
	case "w", "b":
		gs.Turn = fields[1]
	default:
		return gs, fmt.Errorf("invalid FEN: side to move %q", fields[1])
	}

	// Field 3: castling rights ("-" means none). Stored verbatim in KQkq form.
	if fields[2] == "-" {
		gs.CastlingRights = ""
	} else {
		gs.CastlingRights = fields[2]
	}

	// Field 4: en-passant target square.
	if fields[3] != "-" {
		idx := PositionToIndex(fields[3])
		if idx < 0 || idx > 63 {
			return gs, fmt.Errorf("invalid FEN: en passant square %q", fields[3])
		}
		target := uint64(1) << uint(idx)
		rank := idx / 8 // 0-based: rank 3 == 2, rank 6 == 5
		switch rank {
		case 2: // white just pushed; pushed pawn sits one rank above the target
			gs.EnPassant = target | (target << 8)
		case 5: // black just pushed; pushed pawn sits one rank below the target
			gs.EnPassant = target | (target >> 8)
		default:
			return gs, fmt.Errorf("invalid FEN: en passant square %q not on rank 3 or 6", fields[3])
		}
	}

	return gs, nil
}

// ToFEN renders a GameState as a FEN string. It emits "0 1" for the halfmove and
// fullmove counters, which the engine does not track. The en-passant target is
// the single rank-3/rank-6 square (the lower-or-higher of the two stored bits).
func ToFEN(gs dao.GameState) string {
	board := ConvertGameStateToMap(gs)

	var sb strings.Builder
	for r := 7; r >= 0; r-- { // board rank 8 down to rank 1
		empty := 0
		for f := 0; f < 8; f++ {
			square := string(rune('a'+f)) + strconv.Itoa(r+1)
			if p, ok := board[square]; ok && p != "" {
				if empty > 0 {
					sb.WriteString(strconv.Itoa(empty))
					empty = 0
				}
				sb.WriteString(p)
			} else {
				empty++
			}
		}
		if empty > 0 {
			sb.WriteString(strconv.Itoa(empty))
		}
		if r > 0 {
			sb.WriteByte('/')
		}
	}

	turn := gs.Turn
	if turn == "" {
		turn = "w"
	}

	castling := gs.CastlingRights
	if castling == "" {
		castling = "-"
	}

	ep := "-"
	if gs.EnPassant != 0 {
		// The target square is the one on rank 3 (white push) or rank 6 (black push).
		for b := gs.EnPassant; b != 0; b &= b - 1 {
			bit := b & -b
			idx := bitIndex(bit)
			if rank := idx / 8; rank == 2 || rank == 5 {
				ep = bitToSquare(bit, defFiles, defRanks)
				break
			}
		}
	}

	return fmt.Sprintf("%s %s %s %s 0 1", sb.String(), turn, castling, ep)
}

// StartState returns the GameState for the standard starting position.
func StartState() dao.GameState {
	gs, _ := ParseFEN(StartFEN)
	return gs
}

// bitIndex returns the 0-based index of a single set bit.
func bitIndex(bit uint64) int {
	idx := 0
	for bit != 1 {
		bit >>= 1
		idx++
	}
	return idx
}
