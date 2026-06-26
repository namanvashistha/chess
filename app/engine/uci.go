package engine

import (
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"fmt"
	"strings"
)

// ParseUCIMove converts a UCI move string (e.g. "e2e4", "e7e8q", "e1g1") into a
// dto.Move for the given position. The moving piece is derived from the board so
// the caller need not supply it. The optional 5th character is the promotion
// piece. Returns an error for malformed input or an empty source square.
func ParseUCIMove(gs dao.GameState, uci string) (dto.Move, error) {
	uci = strings.TrimSpace(uci)
	if len(uci) < 4 || len(uci) > 5 {
		return dto.Move{}, fmt.Errorf("invalid UCI move %q", uci)
	}

	source := uci[0:2]
	destination := uci[2:4]
	if !isSquare(source) || !isSquare(destination) {
		return dto.Move{}, fmt.Errorf("invalid UCI move %q", uci)
	}

	piece := ConvertGameStateToMap(gs)[source]
	if piece == "" {
		return dto.Move{}, fmt.Errorf("no piece on source square %q for move %q", source, uci)
	}

	move := dto.Move{Piece: piece, Source: source, Destination: destination}
	if len(uci) == 5 {
		promo := strings.ToLower(uci[4:5])
		switch promo {
		case "q", "r", "b", "n":
			move.Promotion = promo
		default:
			return dto.Move{}, fmt.Errorf("invalid promotion %q in move %q", promo, uci)
		}
	}
	return move, nil
}

// MoveToUCI renders a dto.Move as a UCI move string. A non-empty promotion is
// appended as a lowercase letter.
func MoveToUCI(m dto.Move) string {
	uci := m.Source + m.Destination
	if m.Promotion != "" {
		uci += strings.ToLower(m.Promotion)
	}
	return uci
}

func isSquare(s string) bool {
	return len(s) == 2 && s[0] >= 'a' && s[0] <= 'h' && s[1] >= '1' && s[1] <= '8'
}
