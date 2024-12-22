package dto

type Move struct {
	// piece: "wR1",
	// source_square: "a1",
	// destination_square: "a4",

	Piece       string `json:"piece"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	GameId      string `json:"game_id"`
	Token       string `json:"token"`
}

type ChessRequest struct {
	Moves []Move `json:"moves"`
}

type ChessResponse struct {
	Board string `json:"board"`
}

type TokenGetRequest struct {
	Token string `json:"token"`
}

type JoinChessGameRequest struct {
	Token      string `json:"token"`
	InviteCode string `json:"invite_code"`
}
