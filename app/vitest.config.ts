import { defineConfig } from 'vitest/config';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	plugins: [sveltekit()],
	// Svelte's browser build, so component tests can mount components in jsdom
	resolve: { conditions: ['browser'] },
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom',
		globals: true,
		setupFiles: ['./src/test-setup.ts'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html'],
			include: ['src/lib/**/*.{js,ts,svelte}', 'src/routes/**/*.{js,ts,svelte}'],
			exclude: [
				'**/*.test.{js,ts}',
				'**/*.spec.{js,ts}',
				'**/node_modules/**',
				'**/.svelte-kit/**',
				'**/build/**',
				'**/*.d.ts'
			],
			thresholds: {
				lines: 70,
				functions: 70,
				branches: 70,
				statements: 70
			}
		}
	}
});
