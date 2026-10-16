// Client for the JSON API. The session is an HttpOnly cookie the server sets,
// so requests just need to be same-origin. Errors come back as
// { error: { code, message } } with a non-2xx status.
import { user } from './stores.js';

export class ApiError extends Error {
	constructor(status, code, message) {
		super(message);
		this.status = status;
		this.code = code;
	}
}

async function request(method, url, body) {
	const res = await fetch(url, {
		method,
		headers: body ? { 'Content-Type': 'application/json' } : {},
		body: body ? JSON.stringify(body) : undefined
	});
	if (res.status === 204) return null;
	const data = await res.json().catch(() => null);
	if (!res.ok) {
		const err = data?.error ?? {};
		throw new ApiError(res.status, err.code ?? 'unknown', err.message ?? 'Something went wrong.');
	}
	return data;
}

// Ensure there is a session and hydrate the `user` store: reuse the cookie's
// session if it is still valid, otherwise start a new anonymous one.
export async function ensureUser() {
	try {
		const me = await request('GET', '/api/me');
		user.set(me);
		return me;
	} catch (e) {
		if (!(e instanceof ApiError) || e.status !== 401) throw e;
	}
	const s = await request('POST', '/api/session');
	user.set(s.user);
	return s.user;
}

export const lobby = () => request('GET', '/api/lobby');
export const getGame = (id) => request('GET', `/api/games/${id}`);
export const joinGame = (id) => request('POST', `/api/games/${id}/join`);

// mode: "online" | "bot" | "local"; color: "w" | "b" | "random"; level for bots.
export async function createGame({ mode, color = 'random', level, listed }) {
	const { id } = await request('POST', '/api/games', { mode, color, level, listed });
	return id;
}
