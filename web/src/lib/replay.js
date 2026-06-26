// Reconstruct board positions from the stored move list. Each move is the
// engine's "piece+from+to" string, e.g. "Pe2e4", "Ng1f3". This is display-only
// reconstruction (no legality), matching the engine's behaviour for castling,
// en passant, and auto-queen promotion.
const FILES = 'abcdefgh';

function initialMap() {
	const m = {};
	const back = ['R', 'N', 'B', 'Q', 'K', 'B', 'N', 'R'];
	for (let f = 0; f < 8; f++) {
		m[FILES[f] + '8'] = back[f].toLowerCase();
		m[FILES[f] + '7'] = 'p';
		m[FILES[f] + '2'] = 'P';
		m[FILES[f] + '1'] = back[f];
	}
	return m;
}

function applyMove(map, s) {
	const piece = s[0];
	const from = s.slice(1, 3);
	const to = s.slice(3, 5);
	const moving = map[from] || piece;
	const isPawn = piece === 'P' || piece === 'p';

	// En passant: pawn moves diagonally onto an empty square.
	if (isPawn && from[0] !== to[0] && !map[to]) {
		delete map[to[0] + from[1]];
	}
	// Castling: move the rook too.
	if (piece === 'K' || piece === 'k') {
		if (from === 'e1' && to === 'g1') { map['f1'] = map['h1']; delete map['h1']; }
		else if (from === 'e1' && to === 'c1') { map['d1'] = map['a1']; delete map['a1']; }
		else if (from === 'e8' && to === 'g8') { map['f8'] = map['h8']; delete map['h8']; }
		else if (from === 'e8' && to === 'c8') { map['d8'] = map['a8']; delete map['a8']; }
	}
	let placed = moving;
	if (piece === 'P' && to[1] === '8') placed = 'Q'; // auto-queen
	if (piece === 'p' && to[1] === '1') placed = 'q';

	delete map[from];
	map[to] = placed;
}

// positions[k] = the board map after k plies (positions[0] = start).
export function positionsFrom(moves) {
	const positions = [initialMap()];
	let cur = { ...positions[0] };
	for (const mv of moves || []) {
		cur = { ...cur };
		applyMove(cur, mv.move);
		positions.push(cur);
	}
	return positions;
}

// Readable-ish SAN for a move, given the position before it.
export function sanLabel(prevMap, s) {
	const piece = s[0];
	const from = s.slice(1, 3);
	const to = s.slice(3, 5);
	if (piece === 'K' || piece === 'k') {
		if (to === 'g1' || to === 'g8') return 'O-O';
		if (to === 'c1' || to === 'c8') return 'O-O-O';
	}
	const isPawn = piece === 'P' || piece === 'p';
	const capture = !!prevMap[to] || (isPawn && from[0] !== to[0]);
	if (isPawn) {
		const promo = to[1] === '8' || to[1] === '1' ? '=Q' : '';
		return (capture ? from[0] + 'x' : '') + to + promo;
	}
	return piece.toUpperCase() + (capture ? 'x' : '') + to;
}
