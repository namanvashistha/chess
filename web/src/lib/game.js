// The client's model of a game: the server's view (see app/live/protocol.go)
// plus what the board needs, derived once per update.
//
//   board     square -> FEN piece letter, e.g. { e1: 'K', e8: 'k', ... }
//   targets   from-square -> [to-squares] the viewer may play now
//   lastMove  the last move's UCI, or null
//   canMove   whether the viewer has the move
const FILES = 'abcdefgh';

export function boardFromFEN(fen) {
	const board = {};
	const rows = fen.split(' ')[0].split('/');
	rows.forEach((row, r) => {
		let f = 0;
		for (const ch of row) {
			if (ch >= '1' && ch <= '8') f += Number(ch);
			else board[FILES[f++] + (8 - r)] = ch;
		}
	});
	return board;
}

function derive(view) {
	const canMove =
		view.status === 'active' &&
		!view.result &&
		(view.you === 'both' || view.you === view.turn);
	const targets = {};
	if (canMove) {
		for (const uci of view.legal) {
			const from = uci.slice(0, 2);
			const to = uci.slice(2, 4);
			const list = (targets[from] ??= []);
			if (!list.includes(to)) list.push(to);
		}
	}
	const last = view.moves[view.moves.length - 1];
	return { ...view, board: boardFromFEN(view.fen), targets, canMove, lastMove: last?.uci ?? null };
}

// fromSnapshot builds the model from a full view.
export const fromSnapshot = (view) => derive(view);

// withMove applies a "move" message to the model.
export function withMove(game, m) {
	if (m.ply !== game.moves.length) return null; // out of step: wait for a snapshot
	return derive({
		...game,
		moves: [...game.moves, { uci: m.uci, san: m.san }],
		fen: m.fen,
		turn: m.turn,
		check: m.check,
		legal: m.legal,
		status: m.status,
		result: m.result,
		reason: m.reason
	});
}

// uciFor turns a board gesture into the move to send. Promotions become a
// queen for now.
export function uciFor(game, from, to) {
	const base = from + to;
	return game.legal.includes(base) ? base : game.legal.includes(base + 'q') ? base + 'q' : null;
}

const REASONS = {
	checkmate: 'by checkmate',
	stalemate: 'by stalemate',
	repetition: 'by threefold repetition',
	fifty_move: 'by the fifty-move rule',
	insufficient_material: 'by insufficient material'
};

export function resultText(game) {
	if (!game?.result) return '';
	const who = game.result === '1-0' ? 'White wins' : game.result === '0-1' ? 'Black wins' : 'Draw';
	return REASONS[game.reason] ? `${who} ${REASONS[game.reason]}` : who;
}
