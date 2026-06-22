// Live game socket. Binds the server's game_update broadcasts to the
// currentGame store and sends moves. One connection at a time; reconnects on
// unexpected drops; rebinds when the game id changes.
import { currentGame } from './stores.js';
import { token } from './api.js';

let socket = null;
let currentId = null;
let reconnectTimer = null;

const moveSound =
	typeof Audio !== 'undefined'
		? new Audio('https://images.chesscomfiles.com/chess-themes/sounds/_MP3_/default/move-opponent.mp3')
		: null;

function open() {
	const proto = location.protocol === 'https:' ? 'wss' : 'ws';
	socket = new WebSocket(`${proto}://${location.host}/ws/${currentId}`);
	socket.onmessage = (e) => {
		let msg;
		try {
			msg = JSON.parse(e.data);
		} catch {
			return;
		}
		if (msg.payload) currentGame.set(msg.payload);
		if (msg.status === 'success') moveSound?.play().catch(() => {});
	};
	socket.onclose = (e) => {
		if (currentId && !e.wasClean) reconnectTimer = setTimeout(open, 3000);
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
	if (s) {
		s.onclose = null;
		s.close();
	}
}

export function sendMove(piece, source, destination) {
	if (!socket || socket.readyState !== WebSocket.OPEN) return;
	socket.send(
		JSON.stringify({
			type: 'game_update',
			payload: { piece, source, destination, game_id: currentId, token: token() }
		})
	);
}
