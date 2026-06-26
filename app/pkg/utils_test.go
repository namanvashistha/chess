package pkg

import (
	"chess-engine/app/domain/dto"
	"testing"
)

// Regression: adding the optional "promotion" field to dto.Move must not break
// binding of payloads that omit it. Previously the strict binder errored on the
// missing key and left Token unset, which surfaced as "user 0 is not in the game".
func TestBindPayloadOmitsOptionalField(t *testing.T) {
	payload := map[string]interface{}{
		"piece":       "P",
		"source":      "e2",
		"destination": "e4",
		"game_id":     "58",
		"token":       "tok123",
		// no "promotion"
	}
	var move dto.Move
	if err := BindPayloadToStruct(payload, &move); err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if move.Token != "tok123" {
		t.Errorf("Token = %q, want %q (later fields must still bind)", move.Token, "tok123")
	}
	if move.Source != "e2" || move.Destination != "e4" || move.Piece != "P" {
		t.Errorf("unexpected move: %+v", move)
	}
	if move.Promotion != "" {
		t.Errorf("Promotion = %q, want empty", move.Promotion)
	}
}

func TestBindPayloadSetsOptionalWhenPresent(t *testing.T) {
	payload := map[string]interface{}{
		"piece": "P", "source": "e7", "destination": "e8",
		"promotion": "n", "game_id": "1", "token": "t",
	}
	var move dto.Move
	if err := BindPayloadToStruct(payload, &move); err != nil {
		t.Fatalf("bind failed: %v", err)
	}
	if move.Promotion != "n" {
		t.Errorf("Promotion = %q, want %q", move.Promotion, "n")
	}
}

func TestBindPayloadRejectsNonString(t *testing.T) {
	payload := map[string]interface{}{"piece": 42}
	var move dto.Move
	if err := BindPayloadToStruct(payload, &move); err == nil {
		t.Error("expected error for non-string value, got nil")
	}
}
