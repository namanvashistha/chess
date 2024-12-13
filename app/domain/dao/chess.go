package dao

import "encoding/json"

type Chess struct {
	ID   int    `gorm:"column:id; primary_key; not null" json:"id"`
	Name string `gorm:"column:name" json:"name"`
	BaseModel
}

type ChessGame struct {
	ID       int             `gorm:"primaryKey;autoIncrement;not null" json:"id"`
	Board    json.RawMessage `gorm:"column:board;type:jsonb" json:"board"`    // JSONB column type to store the chessboard state
	Turn     string          `gorm:"type:varchar(10);not null" json:"turn"`   // "white" or "black"
	Status   string          `gorm:"type:varchar(20);not null" json:"status"` // "ongoing", "checkmate", etc.
	LastMove string          `gorm:"type:varchar(10)" json:"lastMove"`        // e.g., "e2e4"
	BaseModel
}
