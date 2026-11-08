import { describe, it, expect, vi, afterEach } from 'vitest';
import { webapi } from './gocliRequest';

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
