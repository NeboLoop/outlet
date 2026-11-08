import { describe, it, expect, vi, beforeEach } from 'vitest';
import { flushSync, mount, tick, unmount } from 'svelte';
import { readable } from 'svelte/store';

const resetPassword = vi.fn();
vi.mock('$lib/api', () => ({ resetPassword: (...args: unknown[]) => resetPassword(...args) }));
vi.mock('$app/stores', () => ({
	page: readable({ url: new URL('https://outlet.test/auth/reset-password?token=tok123') })
}));

import Page from './+page.svelte';

function fill(target: HTMLElement, id: string, value: string) {
	const input = target.querySelector<HTMLInputElement>(`#${id}`)!;
	input.value = value;
	input.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('reset password page', () => {
	beforeEach(() => {
		resetPassword.mockReset();
	});

	it('sends the token and new password to the API and shows success', async () => {
		resetPassword.mockResolvedValue({ message: 'ok' });
		const target = document.createElement('div');
		const app = mount(Page, { target });
		fill(target, 'password', 'new-password');
		fill(target, 'confirm-password', 'new-password');
		flushSync();
		target.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true }));
		await vi.waitFor(() => expect(target.textContent).toContain('Password reset successful'));
		expect(resetPassword).toHaveBeenCalledWith({ token: 'tok123', new_password: 'new-password', confirm_password: 'new-password' });
		unmount(app);
	});

	it('shows the API error and stays on the form', async () => {
		resetPassword.mockRejectedValue(new Error('invalid or expired reset token'));
		const target = document.createElement('div');
		const app = mount(Page, { target });
		fill(target, 'password', 'new-password');
		fill(target, 'confirm-password', 'new-password');
		flushSync();
		target.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true }));
		await vi.waitFor(() => expect(target.textContent).toContain('invalid or expired reset token'));
		await tick();
		expect(target.textContent).not.toContain('Password reset successful');
		expect(target.textContent).toContain('Request a new link');
		unmount(app);
	});
});
