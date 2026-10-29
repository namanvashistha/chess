# Architecture

## Packages

| Package | Job |
|---|---|
| `app/engine` | Chess rules, move generation, search, bot levels. `engine.Game` is the one place a move is checked and applied. |
| `app/store` | Users, games and moves. `Postgres` in production, `Memory` for tests; both pass the same test suite. |
| `app/live` | Games in progress. One room (goroutine) per loaded game, one writer goroutine per WebSocket connection. |
| `app/server` | HTTP: the JSON API, sessions, the WebSocket endpoint, and the built web app. |
| `cmd/server` | Wires the above together from environment variables. |
| `cmd/uci` | The engine over UCI, for chess GUIs, engine matches and the lichess bot. |

## Data

```text
users  (id, name, token_hash, bot, created_at)
games  (id, mode, bot_level, white_id, black_id, listed, result, reason, created_at, ended_at)
moves  (game_id, ply, uci, san, played_at)   primary key (game_id, ply)
```

A game's position is never stored. Loading a game replays its moves through
`engine.Game`, which also rebuilds the repetition history. A game's status is
derived: `finished` if it has a result, `waiting` if a seat is open, otherwise
`active`.

`AppendMove` inserts only when `ply` equals the current number of moves and the
game has no result, so the database itself refuses a duplicate, skipped or
late move. `TakeSeat` only fills an empty seat. Both are safe without any
application-level locking.

Game ids are 10 random characters. They are the share link, so a private
(unlisted) game is reachable only by someone who has the link.

Migrations are SQL files in `app/store/migrations`, embedded in the binary and
applied at start under an advisory lock.

## Live games

`live.Manager` loads a room the first time a game is asked for (a snapshot,
a join, or a socket) and unloads it after two minutes with nobody connected.

A room is a goroutine reading closures from its inbox. Everything that touches
the game runs there, in order: moves from either player, joins from the HTTP
API, the bot's replies. A move is checked (seat, turn, ply, legality), written
to the store, and only then sent to every connected client. If the write fails
the move is undone and the sender gets an error and a fresh snapshot.

Bot searches run off the room goroutine on a fixed pool of workers, each with
its own transposition table, so a busy server cannot be made to run unbounded
searches. A finished search is posted back to the room with the ply it was for
and is dropped if the game has moved on.

Each WebSocket connection has a reader and a writer goroutine. The writer owns
all writes, so a slow client only delays itself; if its 64-message queue fills,
it is disconnected. Pings every 25 seconds and a 60-second read deadline clear
out dead connections, and messages are capped at 1 KB.

## Sessions

`POST /api/session` creates an anonymous user and a 32-byte random token. The
browser gets it as an `HttpOnly`, `SameSite=Lax` cookie; native apps send it as
`Authorization: Bearer`. Only the token's SHA-256 is stored. State-changing
requests from another origin are refused, and session and game creation are
rate-limited per client.
