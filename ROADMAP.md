# Roadmap

What comes after Cozy Chess 1.1. Written 2026-10-18, while 1.0 is in App Store
review. Like the rest of this repository's planning docs, it records reasoning
rather than commitments.

## Next: puzzle mode

Cozy Chess teaches while you play. Puzzle mode teaches without the game around
it: one position, one idea, the coach naming the pattern instead of only scoring
the move.

It ships **inside Cozy Chess**, not as a second app. One listing accumulates
ratings, there is no second keyword fight, and every piece of the pipeline is
already built.

### Why this and not another app

Demand was checked before committing, because the thin-competition traps are
easy to walk into.

**Google Trends, United States, 12 months to 2026-10-18** (relative, shared
scale within each row's query):

| Keyword | Mean interest |
| --- | --- |
| `chess app` | 51.3 |
| `sudoku app` | 9.5 |
| `minesweeper app` | 1.4 |

**Bare category interest:**

| Keyword | US | India |
| --- | --- | --- |
| chess | 72.8 | **77.0** |
| sudoku | 26.3 | 10.2 |
| minesweeper | 26.8 | 1.1 |
| nonogram | 0.1 | 0.0 |

Chess leads both markets, and India is its strongest — which is where the Hindi
localisation already shipped pays off.

**Apple iTunes Search API, US store, top 50 results:**

| Term | Top app ratings | Median of top 50 |
| --- | --- | --- |
| chess puzzles | 845,655 | 817 |
| sudoku | 1,864,525 | 11,099 |
| minesweeper | 337,932 | 97 |

`chess puzzles` has real demand with a thin middle. `sudoku` has real demand and
a mature field — rank 25 still carries eleven thousand ratings.

Rejected on the way here:

- **Minesweeper.** The thin field looked like an opening, but app-intent search
  is 1.4 and its top related queries are `minesweeper google`, `play
  minesweeper`, `online minesweeper`. Google serves the game as a search widget,
  so demand never reaches a store. A thin field is more often a ceiling than a
  gap.
- **Nonogram and picross.** Essentially no web search footprint at all.
- **Sudoku.** Real app demand, but a mature field and nothing here to reuse.

Method caveats, same as any proxy: Trends is relative and not comparable across
separate queries, rating counts are not searches, and the search API returns one
country's store.

### What it is

- A **daily puzzle**, and themed sets: forks, pins, skewers, discovered attacks,
  back-rank, deflection, simple endgames.
- The coach names the **pattern**, not just the evaluation. `coach.go` already
  answers "how much did that cost you"; puzzle mode needs "what is the idea
  here".
- Difficulty from each puzzle's rating, adapting to the player.
- Calm by default: no timers, no streak-loss pressure. Streaks may encourage,
  never punish.
- Fully offline, like the rest of the app.

### What it builds on

| Already here | Used for |
| --- | --- |
| `app/engine/coach.go` — `Assessment{Kind,Title,Text,Loss,Best}` | verdicts on the player's attempt |
| search, `evaluate.go`, `see.go` | confirming the solution is forced and unique |
| the two-step adaptive hint in the iOS app | the puzzle hint ladder |
| `mobile/chesscore` → `Chesscore.xcframework` | same gomobile bind |
| `Theme`, `Localize`, `Sound`, `ShareCard`, `BoardView`, `GlassPiece` | the whole surface |
| `make ios-engine`, `make ios-upload` | unchanged |

### Where the puzzles come from

Preferred: the **Lichess open puzzle database**, which is CC0 and carries
themes, ratings and solution lines. Verify the licence and attribution terms
before shipping. Bundle a curated subset — a few thousand across themes and
ratings — so the app stays offline and the download stays small.

Alternative: generate positions with the engine by searching for forced
tactical wins. No licence question, more work, and less human-curated quality.

### Engine work

- Verify each puzzle: the solution line is forced, and the first move is unique.
- Classify the tactic so the coach can name it.
- A local-only rating for the player, since there is no server and none is
  wanted.

### Product shape

Free: the daily puzzle and the first set. One-time unlock for the rest, through
StoreKit 2, checked on the device. Still no backend, no accounts, no analytics.

### Open questions

- Bundle size with a few thousand puzzles, and whether sets should be separate
  downloads.
- Hindi and Spanish wording for the tactic names — these are terms of art, and a
  bad translation reads worse than English.
- Does puzzle mode eventually become its own app for a second listing? Start
  inside, decide once there are numbers.
- Whether the daily puzzle needs a notification, and whether that stays calm.

## Later

- More unlock content for the main game: openings, endgames, a kids mode.
- A second listing once Cozy Chess has conversion data worth copying.

## Before any of it

Cozy Chess has shipped nothing and measured nothing yet. The first real numbers
— review outcome, downloads, retention, conversion — should land before this
roadmap is treated as settled. The bottleneck identified so far is distribution,
not product, and no amount of puzzle mode fixes that.
