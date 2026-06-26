// Command uci exposes the chess engine over the Universal Chess Interface (UCI)
// protocol on stdin/stdout, so it can be driven by standard tooling such as
// Cute Chess, Arena, and Banksia GUI, and tested via gauntlets/EPD suites.
//
// Build:  go build -o bin/uci ./cmd/uci
// Run:    ./bin/uci   (then type UCI commands, or point a GUI at the binary)
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/engine"
)

const (
	engineName   = "GoChess"
	engineAuthor = "namanvashistha"

	defaultMoveOverhead = 30 * time.Millisecond
	defaultMovesToGo    = 30
	minThinkTime        = 10 * time.Millisecond
)

type uciEngine struct {
	out *bufio.Writer
	mu  sync.Mutex // serializes writes to stdout

	state        dao.GameState
	moveOverhead time.Duration

	searchMu sync.Mutex
	stopCh   chan struct{} // non-nil while a search is running
	wg       sync.WaitGroup
}

func main() {
	eng := &uciEngine{
		out:          bufio.NewWriter(os.Stdout),
		state:        engine.StartState(),
		moveOverhead: defaultMoveOverhead,
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024) // long "position ... moves ..." lines
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		switch fields[0] {
		case "uci":
			eng.handleUCI()
		case "isready":
			eng.println("readyok")
		case "ucinewgame":
			eng.stopSearch()
			eng.state = engine.StartState()
		case "setoption":
			eng.handleSetOption(fields)
		case "position":
			eng.handlePosition(fields)
		case "go":
			eng.handleGo(fields)
		case "stop":
			eng.stopSearch()
		case "ponderhit":
			// Not implemented: we never start a ponder search, so ignore.
		case "quit":
			eng.stopSearch()
			eng.out.Flush()
			return
		default:
			// Unknown commands are ignored per the UCI spec.
		}
	}
}

func (e *uciEngine) handleUCI() {
	e.println("id name " + engineName)
	e.println("id author " + engineAuthor)
	// Advertised for GUI compatibility; accepted but currently inert (no TT yet).
	e.println("option name Hash type spin default 16 min 1 max 1024")
	e.println("option name Move Overhead type spin default 30 min 0 max 5000")
	e.println("uciok")
}

func (e *uciEngine) handleSetOption(fields []string) {
	// setoption name <Name...> value <Value...>
	name, value := "", ""
	i := 1
	for i < len(fields) {
		switch fields[i] {
		case "name":
			i++
			var parts []string
			for i < len(fields) && fields[i] != "value" {
				parts = append(parts, fields[i])
				i++
			}
			name = strings.Join(parts, " ")
		case "value":
			i++
			value = strings.Join(fields[i:], " ")
			i = len(fields)
		default:
			i++
		}
	}
	if strings.EqualFold(name, "Move Overhead") {
		if ms, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && ms >= 0 {
			e.moveOverhead = time.Duration(ms) * time.Millisecond
		}
	}
	// Other options (e.g. Hash) are accepted but ignored.
}

// handlePosition parses: position (startpos | fen <6 fields>) [moves m1 m2 ...]
func (e *uciEngine) handlePosition(fields []string) {
	e.stopSearch()

	var gs dao.GameState
	idx := 1
	if idx >= len(fields) {
		return
	}
	switch fields[idx] {
	case "startpos":
		gs = engine.StartState()
		idx++
	case "fen":
		idx++
		if idx+6 > len(fields) {
			e.infoString("invalid position: fen needs 6 fields")
			return
		}
		fen := strings.Join(fields[idx:idx+6], " ")
		parsed, err := engine.ParseFEN(fen)
		if err != nil {
			e.infoString("invalid fen: " + err.Error())
			return
		}
		gs = parsed
		idx += 6
	default:
		return
	}

	if idx < len(fields) && fields[idx] == "moves" {
		idx++
		for ; idx < len(fields); idx++ {
			move, err := engine.ParseUCIMove(gs, fields[idx])
			if err != nil {
				e.infoString("illegal move " + fields[idx] + ": " + err.Error())
				return
			}
			gs = engine.ApplyMove(gs, move)
		}
	}

	e.state = gs
}

func (e *uciEngine) handleGo(fields []string) {
	e.stopSearch()

	opts := engine.SearchOptions{}
	var wtime, btime, winc, binc, movetime int
	movestogo := 0
	hasTime := false

	for i := 1; i < len(fields); i++ {
		readInt := func() int {
			if i+1 < len(fields) {
				i++
				v, _ := strconv.Atoi(fields[i])
				return v
			}
			return 0
		}
		switch fields[i] {
		case "depth":
			opts.MaxDepth = readInt()
		case "movetime":
			movetime = readInt()
			hasTime = true
		case "wtime":
			wtime = readInt()
			hasTime = true
		case "btime":
			btime = readInt()
			hasTime = true
		case "winc":
			winc = readInt()
		case "binc":
			binc = readInt()
		case "movestogo":
			movestogo = readInt()
		case "infinite":
			opts.Infinite = true
		case "nodes", "mate", "searchmoves", "ponder":
			// Not supported; consume any trailing value defensively.
		}
	}

	opts.MoveTime = e.budget(movetime, wtime, btime, winc, binc, movestogo, hasTime)
	if !opts.Infinite && opts.MaxDepth <= 0 && opts.MoveTime == 0 {
		opts.MaxDepth = 6 // sensible default for "go" with no limits
	}

	stop := make(chan struct{})
	opts.Stop = stop

	e.searchMu.Lock()
	e.stopCh = stop
	e.searchMu.Unlock()

	gs := e.state
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		result := engine.Search(gs, opts, e.printInfo)
		e.printBestMove(result)

		e.searchMu.Lock()
		if e.stopCh == stop {
			e.stopCh = nil
		}
		e.searchMu.Unlock()
	}()
}

// budget converts UCI time controls into a per-move time budget. movetime wins
// outright; otherwise we allot a fraction of the side-to-move's remaining clock
// plus its increment, minus the move overhead.
func (e *uciEngine) budget(movetime, wtime, btime, winc, binc, movestogo int, hasTime bool) time.Duration {
	if !hasTime {
		return 0
	}
	if movetime > 0 {
		return clampThink(time.Duration(movetime)*time.Millisecond - e.moveOverhead)
	}

	remaining, inc := wtime, winc
	if e.state.Turn == "b" {
		remaining, inc = btime, binc
	}
	if remaining <= 0 {
		return 0
	}
	mtg := movestogo
	if mtg <= 0 {
		mtg = defaultMovesToGo
	}
	alloc := time.Duration(remaining/mtg+inc)*time.Millisecond - e.moveOverhead
	return clampThink(alloc)
}

func clampThink(d time.Duration) time.Duration {
	if d < minThinkTime {
		return minThinkTime
	}
	return d
}

func (e *uciEngine) printInfo(r engine.SearchResult) {
	ms := r.Elapsed.Milliseconds()
	nps := int64(0)
	if ms > 0 {
		nps = int64(r.Nodes) * 1000 / ms
	}

	var score string
	if r.Mate != 0 {
		score = "mate " + strconv.Itoa(r.Mate)
	} else {
		score = "cp " + strconv.Itoa(r.Score)
	}

	line := fmt.Sprintf("info depth %d score %s nodes %d nps %d time %d",
		r.Depth, score, r.Nodes, nps, ms)
	if len(r.PV) > 0 {
		line += " pv " + pvString(r.PV)
	}
	e.println(line)
}

func (e *uciEngine) printBestMove(r engine.SearchResult) {
	if !r.HasBest {
		e.println("bestmove 0000")
		return
	}
	e.println("bestmove " + engine.MoveToUCI(r.Best))
}

func pvString(pv []dto.Move) string {
	parts := make([]string, len(pv))
	for i, m := range pv {
		parts[i] = engine.MoveToUCI(m)
	}
	return strings.Join(parts, " ")
}

// stopSearch signals any running search to stop and waits for it to finish (so
// its bestmove is emitted before the next command's output).
func (e *uciEngine) stopSearch() {
	e.searchMu.Lock()
	if e.stopCh != nil {
		close(e.stopCh)
		e.stopCh = nil
	}
	e.searchMu.Unlock()
	e.wg.Wait()
}

func (e *uciEngine) infoString(s string) {
	e.println("info string " + s)
}

func (e *uciEngine) println(s string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fmt.Fprintln(e.out, s)
	e.out.Flush()
}
