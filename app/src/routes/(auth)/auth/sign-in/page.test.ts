import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flushSync, mount, unmount } from 'svelte';
import { readable } from 'svelte/store';

const signInWithEmail = vi.fn();
const goto = vi.fn();
const setSession = vi.fn();
let url = new URL('https://outlet.test/auth/sign-in?token=tok123');
vi.mock('$lib/api', () => ({ signInWithEmail: (...args: unknown[]) => signInWithEmail(...args) }));
vi.mock('$app/navigation', () => ({ goto: (...args: unknown[]) => goto(...args) }));
vi.mock('$app/stores', () => ({ page: { subscribe: (fn: (v: unknown) => void) => readable({ url }).subscribe(fn) } }));
vi.mock('$lib/stores/auth.svelte', () => ({ authStore: { setSession: (...args: unknown[]) => setSession(...args) } }));

import Page from './+page.svelte';

describe('sign-in link page', () => {
	beforeEach(() => {
		signInWithEmail.mockReset();
		goto.mockReset();
		setSession.mockReset();
		url = new URL('https://outlet.test/auth/sign-in?token=tok123');
	});

	it('does not use the link on load, only when the person continues', async () => {
		signInWithEmail.mockResolvedValue({ token: 'jwt', user: { id: 'u1' } });
		const target = document.createElement('div');
		const app = mount(Page, { target });
		flushSync();
		expect(signInWithEmail).not.toHaveBeenCalled();
		target.querySelector('button')!.click();
		await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/'));
		expect(signInWithEmail).toHaveBeenCalledWith({ token: 'tok123' });
		expect(setSession).toHaveBeenCalledWith('jwt', { id: 'u1' });
		unmount(app);
	});

	it('shows a used or expired link and offers a new one', async () => {
		signInWithEmail.mockRejectedValue(new Error('That sign-in link or code is wrong or has expired.'));
		const target = document.createElement('div');
		const app = mount(Page, { target });
		target.querySelector('button')!.click();
		await vi.waitFor(() => expect(target.textContent).toContain('wrong or has expired'));
		expect(target.querySelector('a[href="/auth/login"]')).not.toBeNull();
		expect(goto).not.toHaveBeenCalled();
		unmount(app);
	});

	it('says so when the link has no token', () => {
		url = new URL('https://outlet.test/auth/sign-in');
		const target = document.createElement('div');
		const app = mount(Page, { target });
		expect(target.textContent).toContain('missing its token');
		expect(target.querySelector('button')).toBeNull();
		unmount(app);
	});
});
