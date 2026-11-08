import { describe, it, expect, vi } from 'vitest';
import { flushSync, mount, unmount } from 'svelte';

const forgotPassword = vi.fn();
vi.mock('$lib/api', () => ({ forgotPassword: (...args: unknown[]) => forgotPassword(...args) }));

import Page from './+page.svelte';

describe('forgot password page', () => {
	it('asks the API to send the reset link before saying so', async () => {
		forgotPassword.mockResolvedValue({ message: 'sent' });
		const target = document.createElement('div');
		const app = mount(Page, { target });
		const input = target.querySelector<HTMLInputElement>('#email')!;
		input.value = ' ana@example.com ';
		input.dispatchEvent(new Event('input', { bubbles: true }));
		flushSync();
		expect(target.textContent).not.toContain('Check your email');
		target.querySelector('form')!.dispatchEvent(new Event('submit', { cancelable: true }));
		await vi.waitFor(() => expect(target.textContent).toContain('Check your email'));
		expect(forgotPassword).toHaveBeenCalledWith({ email: 'ana@example.com' });
		unmount(app);
	});
});
