package dto

type Move struct {
	// piece: "wR1",
	// source_square: "a1",
	// destination_square: "a4",

	Piece       string `json:"piece"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	// Promotion is the piece to promote a pawn to, as a lowercase letter
	// ("q"|"r"|"b"|"n"). Empty means promote to queen (the historical default).
	Promotion string `json:"promotion,omitempty"`
	GameId    string `json:"game_id"`
	Token     string `json:"token"`
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

type CreateBotGameRequest struct {
	Token      string `json:"token"`
	Difficulty string `json:"difficulty"` // "easy" | "medium" | "hard"
}

type JoinChessGameRequest struct {
	Token      string `json:"token"`
	InviteCode string `json:"invite_code"`
}
