// The live game socket. Snapshots replace the game; move messages extend it;
// errors are shown briefly. Reconnects after unexpected drops, and the server
// sends a fresh snapshot on every connect, so nothing is lost.
import { get } from 'svelte/store';
import { currentGame, moveError } from './stores.js';
import { fromSnapshot, withMove } from './game.js';

let socket = null;
let currentId = null;
let reconnectTimer = null;
let retries = 0;

const moveSound =
	typeof Audio !== 'undefined'
		? new Audio('https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/move-opponent.mp3')
		: null;

function open() {
	const proto = location.protocol === 'https:' ? 'wss' : 'ws';
	const s = new WebSocket(`${proto}://${location.host}/ws/games/${currentId}`);
	socket = s;
	s.onopen = () => (retries = 0);
	s.onmessage = (e) => {
		let msg;
		try {
			msg = JSON.parse(e.data);
		} catch {
			return;
		}
		if (msg.t === 'snapshot') {
			currentGame.set(fromSnapshot(msg.game));
		} else if (msg.t === 'move') {
			const next = withMove(get(currentGame), msg);
			if (next) {
				currentGame.set(next);
				moveSound?.play().catch(() => {});
			} else {
				s.close(); // out of step; reconnecting brings a snapshot
			}
		} else if (msg.t === 'error') {
			moveError.set(msg.message);
			setTimeout(() => moveError.set(null), 2500);
		}
	};
	s.onclose = () => {
		if (socket !== s || !currentId) return;
		const delay = Math.min(10000, 500 * 2 ** retries++);
		reconnectTimer = setTimeout(open, delay);
	};
}

export function connectSocket(id) {
	if (currentId === id && socket && socket.readyState <= WebSocket.OPEN) return;
	disconnectSocket();
	currentId = id;
	open();
}

export function disconnectSocket() {
	clearTimeout(reconnectTimer);
	const s = socket;
	currentId = null;
	socket = null;
	s?.close();
}

export function sendMove(ply, uci) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	socket.send(JSON.stringify({ t: 'move', ply, uci }));
}
