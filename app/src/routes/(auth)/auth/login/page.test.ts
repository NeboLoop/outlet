import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flushSync, mount, unmount } from 'svelte';
import { readable } from 'svelte/store';

const login = vi.fn();
const sendSignInEmail = vi.fn();
const signInWithEmail = vi.fn();
const goto = vi.fn();
const setSession = vi.fn();
vi.mock('$lib/api', () => ({
	login: (...args: unknown[]) => login(...args),
	sendSignInEmail: (...args: unknown[]) => sendSignInEmail(...args),
	signInWithEmail: (...args: unknown[]) => signInWithEmail(...args)
}));
vi.mock('$app/navigation', () => ({ goto: (...args: unknown[]) => goto(...args) }));
vi.mock('$app/stores', () => ({ page: readable({ url: new URL('https://outlet.test/auth/login') }) }));
vi.mock('$lib/stores/auth.svelte', () => ({ authStore: { setSession: (...args: unknown[]) => setSession(...args) } }));

import Page from './+page.svelte';

function fill(target: HTMLElement, id: string, value: string) {
	const input = target.querySelector<HTMLInputElement>(`#${id}`)!;
	input.value = value;
	input.dispatchEvent(new Event('input', { bubbles: true }));
	flushSync();
}

function submit(target: HTMLElement) {
	target.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true }));
}

function click(target: HTMLElement, text: string) {
	const button = [...target.querySelectorAll('button')].find((b) => b.textContent?.trim() === text)!;
	button.click();
	flushSync();
}

describe('login page', () => {
	beforeEach(() => {
		login.mockReset();
		sendSignInEmail.mockReset();
		signInWithEmail.mockReset();
		goto.mockReset();
		setSession.mockReset();
	});

	it('asks for the email first, sends the sign-in email, then signs in with the code', async () => {
		sendSignInEmail.mockResolvedValue({ message: 'sent' });
		signInWithEmail.mockResolvedValue({ token: 'jwt', user: { id: 'u1' } });
		const target = document.createElement('div');
		const app = mount(Page, { target });
		expect(target.querySelector('#password')).toBeNull();

		fill(target, 'email', ' ana@example.com ');
		submit(target);
		await vi.waitFor(() => expect(target.textContent).toContain('Check your email'));
		expect(sendSignInEmail).toHaveBeenCalledWith({ email: 'ana@example.com' });

		fill(target, 'code', ' 042917 ');
		submit(target);
		await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/'));
		expect(signInWithEmail).toHaveBeenCalledWith({ email: 'ana@example.com', code: '042917' });
		expect(setSession).toHaveBeenCalledWith('jwt', { id: 'u1' });
		unmount(app);
	});

	it('shows a wrong code and stays on the code form', async () => {
		sendSignInEmail.mockResolvedValue({ message: 'sent' });
		signInWithEmail.mockRejectedValue(new Error('That sign-in link or code is wrong or has expired.'));
		const target = document.createElement('div');
		const app = mount(Page, { target });
		fill(target, 'email', 'ana@example.com');
		submit(target);
		await vi.waitFor(() => expect(target.querySelector('#code')).not.toBeNull());
		fill(target, 'code', '000000');
		submit(target);
		await vi.waitFor(() => expect(target.textContent).toContain('wrong or has expired'));
		expect(target.querySelector('#code')).not.toBeNull();
		expect(goto).not.toHaveBeenCalled();
		unmount(app);
	});

	it('keeps password sign-in and forgot password as the other way in', async () => {
		login.mockResolvedValue({ token: 'jwt', user: { id: 'u1' } });
		const target = document.createElement('div');
		const app = mount(Page, { target });
		click(target, 'Use a password instead');
		expect(target.querySelector('a[href="/auth/forgot-password"]')).not.toBeNull();
		fill(target, 'email', 'ana@example.com');
		fill(target, 'password', 'secret-password');
		submit(target);
		await vi.waitFor(() => expect(goto).toHaveBeenCalledWith('/'));
		expect(login).toHaveBeenCalledWith({ email: 'ana@example.com', password: 'secret-password' });
		expect(sendSignInEmail).not.toHaveBeenCalled();
		unmount(app);
	});
});
