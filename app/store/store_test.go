package store

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The same behaviour is checked against both implementations, so the memory
// store used by the server tests cannot drift from Postgres.

func TestMemory(t *testing.T) {
	testStore(t, func() Store { return NewMemory() })
}

// TestPostgres runs when TEST_DATABASE_URL names a database whose name contains
// "test". The test wipes that database's public schema.
func TestPostgres(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cfg.Database, "test") {
		t.Fatalf("refusing to wipe database %q: its name must contain \"test\"", cfg.Database)
	}
	testStore(t, func() Store {
		conn, err := pgx.Connect(ctx, url)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
			t.Fatal(err)
		}
		conn.Close(ctx)
		p, err := OpenPostgres(ctx, url)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(p.Close)
		return p
	})
}

func testStore(t *testing.T, open func() Store) {
	ctx := context.Background()

	t.Run("users", func(t *testing.T) {
		s := open()
		u, err := s.CreateUser(ctx, "ada", []byte("hash-a"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.UserByTokenHash(ctx, []byte("hash-a"))
		if err != nil || got.ID != u.ID || got.Name != "ada" {
			t.Fatalf("UserByTokenHash = %+v, %v", got, err)
		}
		if _, err := s.UserByTokenHash(ctx, []byte("nope")); !errors.Is(err, ErrNotFound) {
			t.Errorf("unknown token: err = %v, want ErrNotFound", err)
		}
		if r, err := s.RenameUser(ctx, u.ID, "grace"); err != nil || r.Name != "grace" {
			t.Errorf("RenameUser = %+v, %v", r, err)
		}
		if err := s.DeleteUser(ctx, u.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.UserByTokenHash(ctx, []byte("hash-a")); !errors.Is(err, ErrNotFound) {
			t.Errorf("deleted user still signs in: err = %v", err)
		}
		bot1, err := s.BotUser(ctx)
		if err != nil || !bot1.Bot {
			t.Fatalf("BotUser = %+v, %v", bot1, err)
		}
		if bot2, _ := s.BotUser(ctx); bot2.ID != bot1.ID {
			t.Errorf("BotUser made a second bot: %d then %d", bot1.ID, bot2.ID)
		}
	})

	t.Run("seats and moves", func(t *testing.T) {
		s := open()
		a, _ := s.CreateUser(ctx, "a", []byte("a"))
		b, _ := s.CreateUser(ctx, "b", []byte("b"))
		c, _ := s.CreateUser(ctx, "c", []byte("c"))
		if err := s.CreateGame(ctx, NewGame{ID: "g1", Mode: ModeOnline, WhiteID: a.ID, Listed: true}); err != nil {
			t.Fatal(err)
		}
		g, _ := s.Game(ctx, "g1")
		if g.Status() != StatusWaiting || g.White == nil || g.White.Name != "a" || g.Black != nil {
			t.Fatalf("new game = %+v", g)
		}

		if err := s.TakeSeat(ctx, "g1", "b", b.ID); err != nil {
			t.Fatal(err)
		}
		if err := s.TakeSeat(ctx, "g1", "b", c.ID); !errors.Is(err, ErrSeatTaken) {
			t.Errorf("second TakeSeat: err = %v, want ErrSeatTaken", err)
		}
		if err := s.TakeSeat(ctx, "missing", "b", c.ID); !errors.Is(err, ErrNotFound) {
			t.Errorf("TakeSeat on a missing game: err = %v, want ErrNotFound", err)
		}

		if err := s.AppendMove(ctx, "g1", 0, Move{"e2e4", "e4"}); err != nil {
			t.Fatal(err)
		}
		if err := s.AppendMove(ctx, "g1", 0, Move{"d2d4", "d4"}); !errors.Is(err, ErrConflict) {
			t.Errorf("duplicate ply: err = %v, want ErrConflict", err)
		}
		if err := s.AppendMove(ctx, "g1", 2, Move{"g1f3", "Nf3"}); !errors.Is(err, ErrConflict) {
			t.Errorf("skipped ply: err = %v, want ErrConflict", err)
		}
		if err := s.AppendMove(ctx, "g1", 1, Move{"e7e5", "e5"}); err != nil {
			t.Fatal(err)
		}

		if err := s.Finish(ctx, "g1", "1-0", "resignation"); err != nil {
			t.Fatal(err)
		}
		if err := s.Finish(ctx, "g1", "0-1", "resignation"); !errors.Is(err, ErrConflict) {
			t.Errorf("second Finish: err = %v, want ErrConflict", err)
		}
		if err := s.AppendMove(ctx, "g1", 2, Move{"g1f3", "Nf3"}); !errors.Is(err, ErrConflict) {
			t.Errorf("move after the end: err = %v, want ErrConflict", err)
		}

		g, _ = s.Game(ctx, "g1")
		if g.Status() != StatusFinished || g.Result != "1-0" || g.EndedAt == nil || len(g.Moves) != 2 ||
			g.Moves[1] != (Move{"e7e5", "e5"}) {
			t.Errorf("finished game = %+v", g)
		}
	})

	t.Run("lobby", func(t *testing.T) {
		s := open()
		a, _ := s.CreateUser(ctx, "a", []byte("a"))
		b, _ := s.CreateUser(ctx, "b", []byte("b"))
		c, _ := s.CreateUser(ctx, "c", []byte("c"))
		must := func(err error) {
			t.Helper()
			if err != nil {
				t.Fatal(err)
			}
		}
		must(s.CreateGame(ctx, NewGame{ID: "open", Mode: ModeOnline, WhiteID: a.ID, Listed: true}))
		must(s.CreateGame(ctx, NewGame{ID: "hidden", Mode: ModeOnline, WhiteID: a.ID, Listed: false}))
		must(s.CreateGame(ctx, NewGame{ID: "live", Mode: ModeOnline, WhiteID: a.ID, BlackID: c.ID, Listed: true}))

		l, err := s.Lobby(ctx, b.ID, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(l.Open) != 1 || l.Open[0].ID != "open" || len(l.Live) != 1 || len(l.Mine) != 0 {
			t.Errorf("b's lobby: open %+v, live %+v, mine %+v", l.Open, l.Live, l.Mine)
		}
		l, _ = s.Lobby(ctx, a.ID, 10)
		if len(l.Open) != 0 || len(l.Mine) != 3 {
			t.Errorf("a's lobby: open %d, mine %d; want 0 and 3", len(l.Open), len(l.Mine))
		}
	})
}
