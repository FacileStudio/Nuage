import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

const api = 'http://localhost:4000';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	// muse ships uncompiled .svelte.ts; without this vite dev refuses to start
	// while build and check stay green, so nothing else in the gate catches it.
	optimizeDeps: { exclude: ['@facile/muse'] },
	// In production the Go binary serves this build and answers /api and /webdav
	// itself. Only the dev server needs to forward them to a local API.
	server: {
		proxy: {
			'/api': { target: api, changeOrigin: true },
			'/webdav': { target: api, changeOrigin: true }
		}
	}
});
