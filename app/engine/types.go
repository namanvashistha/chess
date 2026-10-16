package engine

// GameState is one chess position as bitboards. Bit 0 is a1, bit 7 is h1 and
// bit 63 is h8. A piece's colour comes from WhiteBitboard/BlackBitboard and its
// kind from the per-kind boards.
type GameState struct {
	WhiteBitboard  uint64
	BlackBitboard  uint64
	PawnBitboard   uint64
	RookBitboard   uint64
	KnightBitboard uint64
	BishopBitboard uint64
	QueenBitboard  uint64
	KingBitboard   uint64
	// EnPassant holds the square a pawn skipped over on the last move, if any.
	EnPassant uint64
	// CastlingRights is a subset of "KQkq", or "" when neither side can castle.
	CastlingRights string
	// HalfmoveClock counts plies since the last capture or pawn move, for the
	// fifty-move rule.
	HalfmoveClock int
	Turn          string // "w" or "b"
}

// Move is a move between two algebraic squares. Piece is the FEN letter of the
// moving piece; Promotion is "", or one of "q" "r" "b" "n".
type Move struct {
	Piece       string
	Source      string
	Destination string
	Promotion   string
}
