package pkg

import "testing"

// Regression: GenerateRandomString used math/rand reseeded from
// time.Now().UnixNano() on every call, so two tokens minted inside the same
// clock tick were identical -- which also collides the DB's unique index on
// users.token.
func TestGenerateRandomStringIsDistinct(t *testing.T) {
	const n = 2000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		s := GenerateRandomString(80)
		if len(s) != 80 {
			t.Fatalf("length = %d, want 80", len(s))
		}
		if _, dup := seen[s]; dup {
			t.Fatalf("duplicate token after %d draws: %q", i, s)
		}
		seen[s] = struct{}{}
	}
}

func TestGenerateRandomBoolCoversBothValues(t *testing.T) {
	var sawTrue, sawFalse bool
	for i := 0; i < 200 && !(sawTrue && sawFalse); i++ {
		if GenerateRandomBool() {
			sawTrue = true
		} else {
			sawFalse = true
		}
	}
	if !sawTrue || !sawFalse {
		t.Errorf("GenerateRandomBool never produced both values (true=%v false=%v)", sawTrue, sawFalse)
	}
}
