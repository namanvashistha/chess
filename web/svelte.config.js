import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		// SPA: emit a single index.html fallback; the Go backend serves it for
		// all app routes (/ and /game/:id) and the client router takes over.
		adapter: adapter({
			fallback: 'index.html',
			pages: 'build',
			assets: 'build'
		})
	}
};

export default config;
