import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';

const backend = process.env.OUTLET_BACKEND ?? 'http://localhost:20202';

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],

	resolve: {
		alias: {
			$src: 'src'
		}
	},

	server: {
		host: '0.0.0.0',
		// Proxy backend routes during development. OUTLET_BACKEND overrides the
		// Docker port, e.g. OUTLET_BACKEND=http://localhost:8888 pnpm dev
		proxy: {
			...Object.fromEntries(
				['/api', '/sdk', '/health', '/s/', '/confirm/', '/u/', '/w/', '/mcp', '/webhooks'].map((path) => [
					path,
					{ target: backend, changeOrigin: true }
				])
			),
			'/ws': { target: backend.replace(/^http/, 'ws'), ws: true, changeOrigin: true }
		}
	}
});
