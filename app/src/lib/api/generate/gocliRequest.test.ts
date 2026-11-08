import { describe, it, expect, vi, afterEach } from 'vitest';
import { webapi, download } from './gocliRequest';

function mockFetch() {
	const fetchMock = vi.fn(async () => new Response('{}', { status: 200 }));
	vi.stubGlobal('fetch', fetchMock);
	return fetchMock;
}

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('webapi.delete', () => {
	it('sends the request body when the call has params and a body', async () => {
		const fetchMock = mockFetch();
		await webapi.delete('/api/admin/gdpr/contacts/c1', {}, { confirm: true });

		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('/api/admin/gdpr/contacts/c1');
		expect(init.method).toBe('DELETE');
		expect(JSON.parse(String(init.body))).toEqual({ confirm: true });
	});

	it('puts params in the query and the body when called with params only', async () => {
		const fetchMock = mockFetch();
		await webapi.delete('/api/admin/things/1', { force: true });

		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('/api/admin/things/1?force=true');
		expect(JSON.parse(String(init.body))).toEqual({ force: true });
	});
});

describe('webapi.get', () => {
	it('sends params as the query and no body', async () => {
		const fetchMock = mockFetch();
		await webapi.get('/api/admin/things', { page: 2, empty: '' });

		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('/api/admin/things?page=2');
		expect(init.body).toBeUndefined();
	});
});

describe('download', () => {
	it('signs the request in and saves the file under the name the server gives', async () => {
		localStorage.setItem('auth_token', 'tok');
		localStorage.setItem('currentOrgId', 'org1');
		const fetchMock = vi.fn(
			async () =>
				new Response('email\na@example.com\n', {
					status: 200,
					headers: { 'Content-Disposition': 'attachment; filename="news-subscribers-2026-11-08.csv"' }
				})
		);
		vi.stubGlobal('fetch', fetchMock);
		vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:x'), revokeObjectURL: vi.fn() });
		const saved: string[] = [];
		const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
			saved.push(this.download);
		});

		await download('/api/admin/lists/1/export', 'fallback.csv');

		const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
		expect(url).toBe('/api/admin/lists/1/export');
		expect(init.headers).toEqual({ Authorization: 'Bearer tok', 'X-Org-Id': 'org1' });
		expect(saved).toEqual(['news-subscribers-2026-11-08.csv']);
		click.mockRestore();
		localStorage.clear();
	});

	it('reports the server error', async () => {
		vi.stubGlobal('fetch', vi.fn(async () => new Response('{"error":"You don\'t have access to this brand."}', { status: 403 })));
		await expect(download('/api/admin/lists/2/export', 'x.csv')).rejects.toThrow("You don't have access to this brand.");
	});
});
