package dao

import "encoding/json"

type ChessGame struct {
	ID           int        `gorm:"column:id;primaryKey;autoIncrement;not null" json:"id"`
	InviteCode   string     `gorm:"column:invite_code" json:"invite_code"`
	WhitePlayer  string     `gorm:"column:white_player" json:"white_player"`
	BlackPlayer  string     `gorm:"column:black_player" json:"black_player"`
	WhiteScore   int        `gorm:"column:white_score" json:"white_score"`
	BlackScore   int        `gorm:"column:black_score" json:"black_score"`
	Winner       string     `gorm:"column:winner" json:"winner"`
	ChessStateId int        `gorm:"column:chess_state_id;not null" json:"chess_state_id"`
	ChessState   ChessState `gorm:"foreignKey:ChessStateId" json:"chess_state"`
	BaseModel
}

type ChessState struct {
	ID           int                 `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	Board        json.RawMessage     `gorm:"column:board;type:jsonb" json:"board"`
	Turn         string              `gorm:"type:varchar(10);not null" json:"turn"`
	Status       string              `gorm:"type:varchar(20);not null" json:"status"`
	LastMove     string              `gorm:"type:varchar(10)" json:"last_move"`
	AllowedMoves map[string][]string `json:"allowed_moves" gorm:"-"` // Excluded from GORM
	BoardLayout  [8][8]string        `json:"board_layout" gorm:"-"`  // Excluded from GORM
	BaseModel
}
