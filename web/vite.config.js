import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		// In dev, proxy API, WebSocket, and legacy static assets to the Go backend.
		proxy: {
			'/api': 'http://localhost:9000',
			'/static': 'http://localhost:9000',
			'/ws': { target: 'ws://localhost:9000', ws: true }
		}
	}
});
