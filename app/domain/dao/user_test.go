package dao

import (
	"encoding/json"
	"strings"
	"testing"
)

// A ChessGame is broadcast in full to every client watching the game. If
// User.Token is serialisable, that broadcast hands both players' session
// credentials to each other (and to any spectator).
func TestUserTokenIsNeverSerialised(t *testing.T) {
	secret := "SUPER-SECRET-SESSION-TOKEN"

	user := User{ID: 7, Name: "someone", Token: secret, Status: 1}
	b, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("marshal user: %v", err)
	}
	if strings.Contains(string(b), secret) {
		t.Errorf("User JSON leaks the token: %s", b)
	}

	game := ChessGame{ID: 1, WhiteUser: &user, BlackUser: &user}
	b, err = json.Marshal(game)
	if err != nil {
		t.Fatalf("marshal game: %v", err)
	}
	if strings.Contains(string(b), secret) {
		t.Errorf("ChessGame JSON leaks a player's token: %s", b)
	}
}
