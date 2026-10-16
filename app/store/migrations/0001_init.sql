-- Games are stored as a list of moves; every position is derived by replaying
-- them. This replaces the GORM schema, which stored bitboards next to the
-- moves and could disagree with them.
DROP TABLE IF EXISTS game_moves, game_states, chess_games, roles, users CASCADE;

CREATE TABLE users (
    id         bigserial PRIMARY KEY,
    name       text        NOT NULL,
    -- SHA-256 of the session token. NULL for the bot and for deleted accounts,
    -- which therefore cannot sign in.
    token_hash bytea UNIQUE,
    bot        boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- There is one bot user.
CREATE UNIQUE INDEX users_single_bot ON users (bot) WHERE bot;

CREATE TABLE games (
    -- Random and unguessable: it is the share link for a private game.
    id         text PRIMARY KEY,
    mode       text        NOT NULL CHECK (mode IN ('online', 'bot', 'local')),
    bot_level  text        NOT NULL DEFAULT '',
    white_id   bigint REFERENCES users (id),
    black_id   bigint REFERENCES users (id),
    -- Listed games appear in the lobby; unlisted ones are reachable by link only.
    listed     boolean     NOT NULL DEFAULT true,
    result     text        NOT NULL DEFAULT '' CHECK (result IN ('', '1-0', '0-1', '1/2-1/2')),
    reason     text        NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    ended_at   timestamptz
);

CREATE INDEX games_by_white ON games (white_id, created_at DESC);
CREATE INDEX games_by_black ON games (black_id, created_at DESC);
CREATE INDEX games_in_progress ON games (created_at DESC) WHERE result = '' AND listed;

CREATE TABLE moves (
    game_id   text        NOT NULL REFERENCES games (id) ON DELETE CASCADE,
    ply       int         NOT NULL CHECK (ply >= 0),
    uci       text        NOT NULL,
    san       text        NOT NULL,
    played_at timestamptz NOT NULL DEFAULT now(),
    -- One move per ply: a duplicate or out-of-order write fails here.
    PRIMARY KEY (game_id, ply)
);
