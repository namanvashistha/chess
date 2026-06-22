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

// Create (or reuse) an anonymous user; hydrates the `user` store.
export async function ensureUser() {
	let t = localStorage.getItem(KEY_TOKEN);
	const data = localStorage.getItem(KEY_DATA);
	if (t && data) {
		const parsed = JSON.parse(data);
		user.set(parsed);
		return parsed;
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

export async function joinGame(inviteCode) {
	return post('/api/chess/game/join', { invite_code: inviteCode, token: token() });
}
