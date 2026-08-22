package engine

import "math/bits"

// Transposition table.
//
// Chess positions repeat constantly within a search -- 1.e4 e5 2.Nf3 and
// 1.Nf3 e5 2.e4 reach the same place -- and iterative deepening re-searches the
// whole tree at every depth. Without a table all of that is recomputed.
//
// The table is owned by the caller and passed in through SearchOptions rather
// than living in a package variable, because the web server runs searches for
// several games concurrently and a shared table would be a data race.

type ttFlag uint8

const (
	ttExact ttFlag = iota // score is the true value
	ttLower               // score is a lower bound (fail-high, beta cutoff)
	ttUpper               // score is an upper bound (fail-low)
)

type ttEntry struct {
	key   uint64
	src   uint64
	dst   uint64
	score int32
	depth int8
	flag  ttFlag
	promo byte
}

// TranspositionTable is a fixed-size, always-replace hash table.
type TranspositionTable struct {
	entries []ttEntry
	mask    uint64
}

// NewTranspositionTable allocates a table of about sizeMB megabytes, rounded
// down to a power of two entries so indexing is a mask rather than a modulo.
func NewTranspositionTable(sizeMB int) *TranspositionTable {
	if sizeMB < 1 {
		sizeMB = 1
	}
	const entrySize = 40 // approximate; only used to pick a count
	count := uint64(sizeMB) * 1024 * 1024 / entrySize
	if count < 1024 {
		count = 1024
	}
	// Largest power of two not exceeding count.
	count = uint64(1) << (bits.Len64(count) - 1)

	return &TranspositionTable{
		entries: make([]ttEntry, count),
		mask:    count - 1,
	}
}

// Clear empties the table. Called for "ucinewgame" so a new game does not
// inherit the previous one's entries.
func (t *TranspositionTable) Clear() {
	if t == nil {
		return
	}
	for i := range t.entries {
		t.entries[i] = ttEntry{}
	}
}

// probe looks up a position. It returns the stored best move (for ordering)
// whenever the key matches, and a usable score only when the entry was searched
// at least as deep as this node needs and its bound permits a cutoff.
func (t *TranspositionTable) probe(key uint64, depth, alpha, beta int) (move botMove, hasMove bool, score int, usable bool) {
	if t == nil {
		return botMove{}, false, 0, false
	}
	e := &t.entries[key&t.mask]
	if e.key != key {
		return botMove{}, false, 0, false
	}

	if e.src != 0 {
		promo := ""
		if e.promo != 0 {
			promo = string([]byte{e.promo})
		}
		move, hasMove = botMove{src: e.src, dst: e.dst, promo: promo}, true
	}

	// Mate scores encode a distance, which is relative to where they were found;
	// reusing one at a different point in the tree reports the wrong mate
	// distance. Bounds from mate scores are skipped rather than corrected.
	if int(e.depth) < depth || isMateScore(int(e.score)) {
		return move, hasMove, 0, false
	}

	s := int(e.score)
	switch e.flag {
	case ttExact:
		return move, hasMove, s, true
	case ttLower:
		if s >= beta {
			return move, hasMove, s, true
		}
	case ttUpper:
		if s <= alpha {
			return move, hasMove, s, true
		}
	}
	return move, hasMove, 0, false
}

// store writes an entry, always replacing whatever was there. Depth-preferred
// replacement is a refinement worth making once there is a benchmark to show it
// helps; always-replace is the honest starting point.
func (t *TranspositionTable) store(key uint64, depth int, score int, flag ttFlag, best botMove) {
	if t == nil {
		return
	}
	if depth > 127 {
		depth = 127
	}
	var promo byte
	if best.promo != "" {
		promo = best.promo[0]
	}
	t.entries[key&t.mask] = ttEntry{
		key:   key,
		src:   best.src,
		dst:   best.dst,
		score: int32(score),
		depth: int8(depth),
		flag:  flag,
		promo: promo,
	}
}
