// Fails the build when a Help page didn't prerender: the Help index and one
// page per guide in src/lib/help/guides must be in build/.
import { existsSync, readdirSync } from 'node:fs';
import { join } from 'node:path';

const app = join(import.meta.dirname, '..');
const slugs = readdirSync(join(app, 'src/lib/help/guides'))
	.filter((name) => name.endsWith('.md'))
	.map((name) => name.slice(0, -3));

const pages = ['help.html', ...slugs.map((slug) => `help/${slug}.html`)];
const missing = pages.filter((page) => !existsSync(join(app, 'build', page)));

if (missing.length > 0) {
	console.error(`Help pages did not prerender: ${missing.join(', ')}`);
	process.exit(1);
}
console.log(`Help pages prerendered: ${pages.length}`);
