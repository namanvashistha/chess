package engine

import (
	"slices"
	"testing"
)

func TestWatchers(t *testing.T) {
	g := NewGame()
	for _, m := range []string{"e2e4", "d7d5"} {
		if _, err := g.Play(m); err != nil {
			t.Fatal(err)
		}
	}
	guards, attackers := g.Watchers("e4")
	if len(guards) != 0 || !slices.Equal(attackers, []string{"d5"}) {
		t.Fatalf("e4: guards %v attackers %v", guards, attackers)
	}
	guards, attackers = g.Watchers("d5")
	if !slices.Equal(guards, []string{"d8"}) || !slices.Equal(attackers, []string{"e4"}) {
		t.Fatalf("d5: guards %v attackers %v", guards, attackers)
	}
	if gu, at := g.Watchers("e5"); gu != nil || at != nil {
		t.Fatalf("empty square: %v %v", gu, at)
	}
}
