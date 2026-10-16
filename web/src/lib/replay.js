// Positions for move review, rebuilt from the UCI move list. Display only: the
// moves are already known to be legal, so this just moves pieces, including
// the rook in castling, the pawn taken en passant, and the promoted piece.
import { boardFromFEN } from './game.js';

const START = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR';

function apply(board, uci) {
	const from = uci.slice(0, 2);
	const to = uci.slice(2, 4);
	const piece = board[from];
	const pawn = piece === 'P' || piece === 'p';
	if (pawn && from[0] !== to[0] && !board[to]) delete board[to[0] + from[1]];
	if ((piece === 'K' || piece === 'k') && Math.abs(from.charCodeAt(0) - to.charCodeAt(0)) === 2) {
		const rank = from[1];
		const [rf, rt] = to[0] === 'g' ? ['h', 'f'] : ['a', 'd'];
		board[rt + rank] = board[rf + rank];
		delete board[rf + rank];
	}
	delete board[from];
	const promo = uci[4];
	board[to] = promo ? (piece === 'P' ? promo.toUpperCase() : promo) : piece;
}

// positionsFrom returns the board after each ply: positions[0] is the start.
export function positionsFrom(moves) {
	const positions = [boardFromFEN(START)];
	for (const m of moves ?? []) {
		const next = { ...positions[positions.length - 1] };
		apply(next, m.uci);
		positions.push(next);
	}
	return positions;
}
