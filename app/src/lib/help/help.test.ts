import { describe, it, expect, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import { readable } from 'svelte/store';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join, relative } from 'node:path';

vi.mock('$app/stores', () => ({ page: readable({ route: { id: '/[brandSlug]/lists' } }) }));

import GuidePage from '../../routes/(brand-select)/help/[slug]/+page.svelte';
import HelpLink from '$lib/components/ui/HelpLink.svelte';
import { guides, guideFiles, findGuide, searchGuides } from './guides';
import { screenGuides, guideForScreen } from './screens';

const routesDir = join(__dirname, '../../routes');
const srcDir = join(__dirname, '../..');

function walk(dir: string): string[] {
	return readdirSync(dir).flatMap((name) => {
		const path = join(dir, name);
		return statSync(path).isDirectory() ? walk(path) : [path];
	});
}

describe('help guides', () => {
	it('lists every guide file, and has a file for every listed guide', () => {
		expect(guideFiles.sort()).toEqual(guides.map((guide) => guide.slug).sort());
	});

	it.each(guides.map((guide) => [guide.slug, guide] as const))('%s has a title and a summary', (_, guide) => {
		expect(guide.title).not.toBe('');
		expect(guide.title).not.toContain(':');
		expect(guide.summary).not.toBe('');
		expect(guide.markdown).not.toMatch(/\{\{label\./);
	});

	it.each(guides.map((guide) => [guide.slug, guide] as const))('%s renders', (_, guide) => {
		const target = document.createElement('div');
		const app = mount(GuidePage, { target, props: { data: { guide } } as never });
		expect(target.querySelector('h1')?.textContent).toBe(guide.title);
		expect(target.querySelectorAll('p').length).toBeGreaterThan(1);
		unmount(app);
	});

	it('links only to guides that exist', () => {
		for (const guide of guides) {
			for (const [, slug] of guide.markdown.matchAll(/\]\(\/help\/([a-z0-9-]+)/g)) {
				expect(findGuide(slug), `${guide.slug} links to ${slug}`).toBeDefined();
			}
		}
	});

	it('finds a guide by the words in it', () => {
		expect(searchGuides('').length).toBe(guides.length);
		expect(searchGuides('DKIM').map((guide) => guide.slug)).toContain('sending-domain-and-dns');
		expect(searchGuides('nothing-says-this-zzz')).toEqual([]);
	});
});

describe('help links', () => {
	it('send every screen to a guide that exists', () => {
		for (const [screen, slug] of Object.entries(screenGuides)) {
			expect(findGuide(slug), `${screen} points to ${slug}`).toBeDefined();
		}
	});

	it('cover every screen of the app', () => {
		const screens = walk(routesDir)
			.filter((path) => path.endsWith('+page.svelte'))
			.map((path) =>
				('/' + relative(routesDir, path).replace(/\/?\+page\.svelte$/, '')).replace(/\/$/, '')
			)
			// Signed out screens have no Help: Help needs a sign-in
			.filter((id) => !id.startsWith('/(auth)') && !id.startsWith('/(onboarding)/setup'))
			.filter((id) => !id.startsWith('/(brand-select)/help'));
		expect(screens.length).toBeGreaterThan(30);
		for (const id of screens) {
			expect(guideForScreen(id === '' ? '/' : id), `${id} has no guide`).toBeDefined();
		}
	});

	it('name only guides that exist', () => {
		for (const path of walk(srcDir).filter((path) => path.endsWith('.svelte'))) {
			const source = readFileSync(path, 'utf8');
			for (const [, slug] of source.matchAll(/<HelpLink[^>]*\bguide="([^"]+)"/g)) {
				expect(findGuide(slug), `${path} names ${slug}`).toBeDefined();
			}
		}
	});

	it('go to the screen guide, and to the index from Help itself', () => {
		const target = document.createElement('div');
		const app = mount(HelpLink, { target });
		expect(target.querySelector('a')?.getAttribute('href')).toBe('/help/lists-and-subscribers');
		unmount(app);
		expect(guideForScreen('/[brandSlug]/lists/[id]/subscribers')).toBe('lists-and-subscribers');
		expect(guideForScreen('/(brand-select)/help/[slug]')).toBeUndefined();
	});
});
