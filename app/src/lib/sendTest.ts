import { getOrganizationBySlug, type SendEmailRequest } from '$lib/api';

/**
 * Sends one email through the brand's sending API (/sdk/v1/emails) with the
 * brand's own API key: the same path, sender and suppression rules as the
 * brand's apps, so a test proves the real setup works. The admin session can
 * read the key; the SDK endpoint does not accept admin logins.
 */
export async function sendWithBrandKey(brandSlug: string, req: SendEmailRequest): Promise<string> {
	const org = await getOrganizationBySlug({}, brandSlug);
	const res = await fetch('/sdk/v1/emails', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json', 'X-API-Key': org.api_key },
		body: JSON.stringify(req)
	});
	const result = await res.json().catch(() => null);
	if (!res.ok || !result?.success) {
		throw new Error(result?.message || result?.error || `Send failed with status ${res.status}`);
	}
	return result.message_id;
}
