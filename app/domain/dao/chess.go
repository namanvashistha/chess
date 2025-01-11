package dao

type ChessGame struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement;not null" json:"id"`
	InviteCode string `gorm:"column:invite_code" json:"invite_code"`
	Winner     string `gorm:"column:winner" json:"winner"`
	// ChessStateId int                 `gorm:"column:chess_state_id;not null" json:"chess_state_id"`
	// ChessState   ChessState          `gorm:"foreignKey:ChessStateId" json:"chess_state"`
	WhiteUserId  *int                `gorm:"column:white_user_id" json:"white_user_id"`
	BlackUserId  *int                `gorm:"column:black_user_id" json:"black_user_id"`
	WhiteUser    *User               `gorm:"foreignKey:WhiteUserId" json:"white_user"`
	BlackUser    *User               `gorm:"foreignKey:BlackUserId" json:"black_user"`
	State        GameState           `gorm:"foreignKey:GameID" json:"state"`
	LegalMoves   map[string][]string `json:"legal_moves" gorm:"-"` // Excluded from GORM
	CurrentState map[string]string   `json:"current_state" gorm:"-"`
	BoardLayout  [8][8][2]string     `json:"board_layout" gorm:"-"` // Excluded from GORM
	BaseModel
}

type GameState struct {
	ID             int    `gorm:"primaryKey;autoIncrement" json:"id"`
	GameID         int    `gorm:"not null;index" json:"game_id"`
	WhiteBitboard  uint64 `gorm:"type:numeric(20,0);not null" json:"white_bitboard"`
	BlackBitboard  uint64 `gorm:"type:numeric(20,0);not null" json:"black_bitboard"`
	PawnBitboard   uint64 `gorm:"type:numeric(20,0);not null" json:"pawn_bitboard"`
	RookBitboard   uint64 `gorm:"type:numeric(20,0);not null" json:"rook_bitboard"`
	KnightBitboard uint64 `gorm:"type:numeric(20,0);not null" json:"knight_bitboard"`
	BishopBitboard uint64 `gorm:"type:numeric(20,0);not null" json:"bishop_bitboard"`
	QueenBitboard  uint64 `gorm:"type:numeric(20,0);not null" json:"queen_bitboard"`
	KingBitboard   uint64 `gorm:"type:numeric(20,0);not null" json:"king_bitboard"`
	EnPassant      uint64 `gorm:"type:numeric(20,0)" json:"en_passant"`
	CastlingRights string `gorm:"type:varchar(4)" json:"castling_rights"`
	LastMove       string `gorm:"type:varchar(10)" json:"last_move"`
	Turn           string `gorm:"type:varchar(1);not null" json:"turn"`
	BaseModel
}
