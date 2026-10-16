package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Postgres is the production Store.
type Postgres struct {
	pool *pgxpool.Pool
}

// OpenPostgres connects to url and applies any pending migrations.
func OpenPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect: %w", err)
	}
	p := &Postgres{pool: pool}
	if err := p.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return p, nil
}

// migrate applies each embedded migration not yet recorded, in file-name
// order, each in its own transaction. A lock keeps two starting servers from
// racing.
func (p *Postgres) migrate(ctx context.Context) error {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	conn, err := p.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	const lockKey = 7_461_626 // arbitrary, fixed for this app
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey) //nolint:errcheck

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version int PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	for _, f := range files {
		name := strings.TrimPrefix(f, "migrations/")
		version, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return fmt.Errorf("migration %s: name must start with a number", name)
		}
		var done bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		sql, err := migrations.ReadFile(f)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sql)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version)
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

func (p *Postgres) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }
func (p *Postgres) Close()                         { p.pool.Close() }

func (p *Postgres) CreateUser(ctx context.Context, name string, tokenHash []byte) (User, error) {
	u := User{Name: name}
	err := p.pool.QueryRow(ctx,
		"INSERT INTO users (name, token_hash) VALUES ($1, $2) RETURNING id, created_at",
		name, tokenHash).Scan(&u.ID, &u.CreatedAt)
	return u, err
}

func (p *Postgres) UserByTokenHash(ctx context.Context, tokenHash []byte) (User, error) {
	var u User
	err := p.pool.QueryRow(ctx,
		"SELECT id, name, bot, created_at FROM users WHERE token_hash = $1", tokenHash).
		Scan(&u.ID, &u.Name, &u.Bot, &u.CreatedAt)
	return u, notFound(err)
}

func (p *Postgres) RenameUser(ctx context.Context, id int64, name string) (User, error) {
	u := User{ID: id}
	err := p.pool.QueryRow(ctx,
		"UPDATE users SET name = $2 WHERE id = $1 AND NOT bot RETURNING name, bot, created_at",
		id, name).Scan(&u.Name, &u.Bot, &u.CreatedAt)
	return u, notFound(err)
}

func (p *Postgres) DeleteUser(ctx context.Context, id int64) error {
	tag, err := p.pool.Exec(ctx,
		"UPDATE users SET name = $2, token_hash = NULL WHERE id = $1 AND NOT bot", id, DeletedName)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (p *Postgres) BotUser(ctx context.Context) (User, error) {
	u := User{Bot: true}
	// The insert is a no-op once the bot exists (the partial unique index), so
	// this is safe to race.
	_, err := p.pool.Exec(ctx,
		"INSERT INTO users (name, bot) VALUES ($1, true) ON CONFLICT (bot) WHERE bot DO NOTHING", BotName)
	if err != nil {
		return User{}, err
	}
	err = p.pool.QueryRow(ctx, "SELECT id, name, created_at FROM users WHERE bot").
		Scan(&u.ID, &u.Name, &u.CreatedAt)
	return u, err
}

func (p *Postgres) CreateGame(ctx context.Context, g NewGame) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO games (id, mode, bot_level, white_id, black_id, listed)
		 VALUES ($1, $2, $3, NULLIF($4, 0), NULLIF($5, 0), $6)`,
		g.ID, g.Mode, g.BotLevel, g.WhiteID, g.BlackID, g.Listed)
	return err
}

const gameColumns = `g.id, g.mode, g.bot_level, g.listed, g.result, g.reason, g.created_at, g.ended_at,
	w.id, w.name, w.bot, b.id, b.name, b.bot,
	(SELECT count(*) FROM moves m WHERE m.game_id = g.id)`

const gameJoins = `FROM games g
	LEFT JOIN users w ON w.id = g.white_id
	LEFT JOIN users b ON b.id = g.black_id`

type rowScanner interface{ Scan(dest ...any) error }

func scanGame(row rowScanner) (Game, int, error) {
	var (
		g            Game
		wID, bID     *int64
		wName, bName *string
		wBot, bBot   *bool
		plies        int
	)
	err := row.Scan(&g.ID, &g.Mode, &g.BotLevel, &g.Listed, &g.Result, &g.Reason, &g.CreatedAt, &g.EndedAt,
		&wID, &wName, &wBot, &bID, &bName, &bBot, &plies)
	if err != nil {
		return Game{}, 0, err
	}
	if wID != nil {
		g.White = &User{ID: *wID, Name: *wName, Bot: *wBot}
	}
	if bID != nil {
		g.Black = &User{ID: *bID, Name: *bName, Bot: *bBot}
	}
	return g, plies, nil
}

func (p *Postgres) Game(ctx context.Context, id string) (Game, error) {
	g, _, err := scanGame(p.pool.QueryRow(ctx, "SELECT "+gameColumns+" "+gameJoins+" WHERE g.id = $1", id))
	if err != nil {
		return Game{}, notFound(err)
	}
	rows, err := p.pool.Query(ctx, "SELECT uci, san FROM moves WHERE game_id = $1 ORDER BY ply", id)
	if err != nil {
		return Game{}, err
	}
	g.Moves, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Move, error) {
		var m Move
		return m, r.Scan(&m.UCI, &m.SAN)
	})
	return g, err
}

func (p *Postgres) TakeSeat(ctx context.Context, gameID, color string, userID int64) error {
	column := map[string]string{"w": "white_id", "b": "black_id"}[color]
	if column == "" {
		return fmt.Errorf("bad color %q", color)
	}
	tag, err := p.pool.Exec(ctx,
		"UPDATE games SET "+column+" = $2 WHERE id = $1 AND "+column+" IS NULL AND result = ''",
		gameID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		if _, err := p.Game(ctx, gameID); err != nil {
			return err
		}
		return ErrSeatTaken
	}
	return nil
}

func (p *Postgres) AppendMove(ctx context.Context, gameID string, ply int, m Move) error {
	// Inserting only when ply equals the current move count, and the game has
	// no result, makes the write itself the concurrency check.
	tag, err := p.pool.Exec(ctx,
		`INSERT INTO moves (game_id, ply, uci, san)
		 SELECT $1, $2::int, $3, $4
		 WHERE EXISTS (SELECT 1 FROM games WHERE id = $1 AND result = '')
		   AND (SELECT count(*) FROM moves WHERE game_id = $1) = $2::int`,
		gameID, ply, m.UCI, m.SAN)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrConflict
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (p *Postgres) Finish(ctx context.Context, gameID, result, reason string) error {
	tag, err := p.pool.Exec(ctx,
		"UPDATE games SET result = $2, reason = $3, ended_at = now() WHERE id = $1 AND result = ''",
		gameID, result, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	return nil
}

func (p *Postgres) Lobby(ctx context.Context, userID int64, limit int) (Lobby, error) {
	var l Lobby
	var err error
	l.Open, err = p.summaries(ctx, `WHERE g.listed AND g.mode = 'online' AND g.result = ''
		AND (g.white_id IS NULL OR g.black_id IS NULL) AND g.created_at > $1
		AND coalesce(g.white_id, g.black_id) <> $2
		ORDER BY g.created_at DESC LIMIT $3`, time.Now().Add(-WaitingTTL), userID, limit)
	if err != nil {
		return Lobby{}, err
	}
	l.Live, err = p.summaries(ctx, `WHERE g.listed AND g.result = ''
		AND g.white_id IS NOT NULL AND g.black_id IS NOT NULL
		AND g.white_id <> $1 AND g.black_id <> $1
		ORDER BY g.created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return Lobby{}, err
	}
	if userID != 0 {
		l.Mine, err = p.summaries(ctx, `WHERE g.white_id = $1 OR g.black_id = $1
			ORDER BY g.created_at DESC LIMIT $2`, userID, limit)
	}
	return l, err
}

func (p *Postgres) summaries(ctx context.Context, where string, args ...any) ([]Summary, error) {
	rows, err := p.pool.Query(ctx, "SELECT "+gameColumns+" "+gameJoins+" "+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		g, plies, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, summarize(g, plies))
	}
	return out, rows.Err()
}

func summarize(g Game, plies int) Summary {
	return Summary{ID: g.ID, Mode: g.Mode, White: g.White, Black: g.Black,
		Status: g.Status(), Result: g.Result, Plies: plies, CreatedAt: g.CreatedAt}
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
