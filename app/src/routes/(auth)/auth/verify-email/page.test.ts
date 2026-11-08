import { describe, it, expect, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import { readable } from 'svelte/store';

const verifyEmail = vi.fn();
vi.mock('$lib/api', () => ({ verifyEmail: (...args: unknown[]) => verifyEmail(...args) }));
vi.mock('$app/stores', () => ({
	page: readable({ url: new URL('https://outlet.test/auth/verify-email?token=tok123') })
}));

import Page from './+page.svelte';

describe('verify email page', () => {
	it('verifies the token with the API', async () => {
		verifyEmail.mockResolvedValue({ message: 'ok' });
		const target = document.createElement('div');
		const app = mount(Page, { target });
		await vi.waitFor(() => expect(target.textContent).toContain('Email verified successfully'));
		expect(verifyEmail).toHaveBeenCalledWith({ token: 'tok123' });
		unmount(app);
	});

	it('shows why verification failed', async () => {
		verifyEmail.mockRejectedValue(new Error('invalid or expired verification token'));
		const target = document.createElement('div');
		const app = mount(Page, { target });
		await vi.waitFor(() => expect(target.textContent).toContain('invalid or expired verification token'));
		expect(target.textContent).not.toContain('Email verified successfully');
		unmount(app);
	});
});
