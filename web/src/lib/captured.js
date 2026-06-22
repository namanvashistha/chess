// Derive captured pieces and material advantage from the live board map
// (square -> FEN piece char). Ported from the original board.js.
const VALUES = { p: 1, n: 3, b: 3, r: 5, q: 9, k: 0 };
const START = { p: 8, n: 2, b: 2, r: 2, q: 1, k: 1 };

export function computeCaptured(state) {
	const onBoard = {
		w: { p: 0, n: 0, b: 0, r: 0, q: 0, k: 0 },
		b: { p: 0, n: 0, b: 0, r: 0, q: 0, k: 0 }
	};
	Object.values(state || {}).forEach((code) => {
		if (!code) return;
		const side = code === code.toUpperCase() ? 'w' : 'b';
		onBoard[side][code.toLowerCase()]++;
	});

	const order = ['q', 'r', 'b', 'n', 'p'];
	const whiteCaptured = []; // black pieces white took (lowercase codes)
	const blackCaptured = []; // white pieces black took (uppercase codes)
	let whiteMat = 0;
	let blackMat = 0;
	order.forEach((t) => {
		const wLost = Math.max(0, START[t] - onBoard.w[t]);
		const bLost = Math.max(0, START[t] - onBoard.b[t]);
		for (let i = 0; i < bLost; i++) whiteCaptured.push(t);
		for (let i = 0; i < wLost; i++) blackCaptured.push(t.toUpperCase());
		whiteMat += onBoard.w[t] * VALUES[t];
		blackMat += onBoard.b[t] * VALUES[t];
	});

	return { whiteCaptured, blackCaptured, adv: whiteMat - blackMat };
}
