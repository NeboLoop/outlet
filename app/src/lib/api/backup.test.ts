import { describe, it, expect, vi, afterEach } from 'vitest';
import { downloadBackup } from './index';

afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
	localStorage.clear();
});

describe('downloadBackup', () => {
	it('fetches the streamed file signed in and saves it under the server name', async () => {
		localStorage.setItem('auth_token', 'tok');
		const fetchMock = vi.fn(
			async () =>
				new Response('SQLite format 3', {
					status: 200,
					headers: { 'Content-Disposition': 'attachment; filename="outlet-2026-11-08.db"' }
				})
		);
		vi.stubGlobal('fetch', fetchMock);
		vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() });
		const saved: string[] = [];
		vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
			saved.push(this.download);
		});

		await downloadBackup('b1', 'fallback.db');

		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('/api/admin/backup/b1/download');
		expect(init.headers).toEqual({ Authorization: 'Bearer tok' });
		expect(saved).toEqual(['outlet-2026-11-08.db']);
	});

	it('reports a refusal', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('{"error":"Only a super admin can do this."}', { status: 403 })));
		await expect(downloadBackup('b1', 'x.db')).rejects.toThrow('Only a super admin can do this.');
	});
});
