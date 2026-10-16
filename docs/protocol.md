# API and WebSocket protocol

All bodies are JSON. Errors are `{"error": {"code": "...", "message": "..."}}`
with a 4xx or 5xx status: `code` is for programs, `message` is for people.

## HTTP

| Method and path | Auth | Does |
|---|---|---|
| `POST /api/session` | none | Creates an anonymous user. Sets the session cookie and returns `{user, token}`. |
| `GET /api/me` | session | The current user. 401 when signed out. |
| `PATCH /api/me` | session | `{"name": "..."}`, 1 to 24 characters. |
| `DELETE /api/me` | session | Removes the identity; the user's games stay, as "Deleted player". |
| `GET /api/lobby` | optional | `{open, live, mine}`: listed games with an open seat, listed games in progress, and the viewer's recent games. |
| `POST /api/games` | session | `{"mode": "online" \| "bot" \| "local", "color": "w" \| "b" \| "random", "level": "easy" \| "medium" \| "hard", "listed": bool}`. Returns `{id}`. |
| `GET /api/games/{id}` | optional | The game as the viewer sees it (the `game` object below). |
| `POST /api/games/{id}/join` | session | Takes the open seat of an online game. Returns `{color}`. 409 when full. |
| `GET /api/games/{id}/pgn` | none | The game as a PGN file. |
| `GET /health` | none | 200 with `{status, rooms}`, or 503 when the database is unreachable. |

A session is the `session` cookie or `Authorization: Bearer <token>`.

## WebSocket: `GET /ws/games/{id}`

Signing in is optional; without a session the connection watches.

### From the client

```json
{"t": "move", "ply": 4, "uci": "g1f3"}
```

`ply` is the number of moves already played. A move for any other ply is
rejected as `stale`, so a resent message can never be applied twice.
Promotions carry the piece: `e7e8q`.

### From the server

**snapshot**: the whole game. Sent on connect, to everyone when a seat is
taken, and to a client after a `stale` or `save_failed` error.

```json
{"t": "snapshot", "game": {
  "id": "Hx7kP2mQ9a", "mode": "online", "bot_level": "",
  "white": {"id": 1, "name": "brave-otter-0412", "bot": false},
  "black": null,
  "status": "waiting", "result": "", "reason": "",
  "moves": [{"uci": "e2e4", "san": "e4"}],
  "fen": "rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1",
  "turn": "b", "check": false,
  "legal": ["a7a6", "..."],
  "you": "w"
}}
```

`you` is the viewer's colour: `w`, `b`, `both` in a pass-and-play game, or `""`
for a spectator. `legal` is empty unless the game is `active`.

**move**: one move, sent to everyone.

```json
{"t": "move", "ply": 1, "uci": "e7e5", "san": "e5",
 "fen": "...", "turn": "w", "check": false, "legal": ["..."],
 "status": "active", "result": "", "reason": ""}
```

`ply` is this move's index, so after it the game has `ply + 1` moves. A
client whose move count differs should reconnect for a snapshot.

When a move ends the game, `result` is `1-0`, `0-1` or `1/2-1/2` and `reason`
is `checkmate`, `stalemate`, `repetition`, `fifty_move` or
`insufficient_material`.

**error**: sent only to the client whose message caused it.

```json
{"t": "error", "code": "not_your_turn", "message": "it is not your turn"}
```

Codes: `bad_message`, `not_your_turn`, `stale`, `illegal`, `game_over`,
`save_failed`.
