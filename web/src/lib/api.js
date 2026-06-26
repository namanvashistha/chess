// Thin client over the Go JSON API. All responses are { response_key, data }.
import { user } from './stores.js';

const KEY_TOKEN = 'userToken';
const KEY_DATA = 'userData';

export function token() {
	return localStorage.getItem(KEY_TOKEN);
}

export function userId() {
	try {
		return JSON.parse(localStorage.getItem(KEY_DATA))?.id ?? null;
	} catch {
		return null;
	}
}

async function post(url, body) {
	const res = await fetch(url, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body ?? {})
	});
	return res.json();
}

// Ensure a valid anonymous user and hydrate the `user` store. A stored token is
// VERIFIED against the server (it can go stale if the DB is reset/migrated); if
// it no longer maps to a user, a fresh identity is created. This prevents moves
// from being rejected as "user 0 is not in the game".
export async function ensureUser() {
	const stored = localStorage.getItem(KEY_TOKEN);
	if (stored) {
		try {
			const me = await post('/api/user/me', { token: stored });
			if (me.response_key === 'SUCCESS' && me.data && me.data.id) {
				localStorage.setItem(KEY_DATA, JSON.stringify(me.data));
				user.set(me.data);
				return me.data;
			}
		} catch {
			// network error: fall through and try to (re)create
		}
	}
	const j = await post('/api/user');
	if (j.response_key === 'SUCCESS') {
		localStorage.setItem(KEY_TOKEN, j.data.token);
		localStorage.setItem(KEY_DATA, JSON.stringify(j.data));
		user.set(j.data);
		return j.data;
	}
	return null;
}

export async function listGames() {
	const res = await fetch('/api/chess/game');
	const j = await res.json();
	return j.response_key === 'SUCCESS' && Array.isArray(j.data) ? j.data : [];
}

export async function getGame(id) {
	const res = await fetch(`/api/chess/game/${id}`);
	const j = await res.json();
	return j.response_key === 'SUCCESS' ? j.data : null;
}

export async function createGame() {
	return post('/api/chess/game', { token: token() });
}

export async function createBotGame(difficulty) {
	return post('/api/chess/game/bot', { token: token(), difficulty });
}

export async function createLocalGame() {
	return post('/api/chess/game/local', { token: token() });
}

export async function joinGame(inviteCode) {
	return post('/api/chess/game/join', { invite_code: inviteCode, token: token() });
}
