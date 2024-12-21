package dto

type Move struct {
	// piece: "wR1",
	// source_square: "a1",
	// destination_square: "a4",

	Piece       string `json:"piece"`
	Source      string `json:"source"`
	Destination string `json:"destination`
	GameId      string `json:"game_id"`
}

type ChessRequest struct {
	Moves []Move `json:"moves"`
}

type ChessResponse struct {
	Board string `json:"board"`
}

type CreateChessGameRequest struct {
	Token string `json:"token"`
}
