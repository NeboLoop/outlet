// Helpers for the Sending settings screen.

import type { SendingPathway } from '$lib/api';

// The pathways a domain can move to: an outreach domain only to another
// mailbox pool, a sending domain only to another platform, ses or smtp pathway.
export function moveTargets(pathways: SendingPathway[], fromId: string, purpose: string): SendingPathway[] {
	const pool = purpose === 'outreach';
	return pathways.filter((p) => p.id !== fromId && (p.provider === 'mailboxes') === pool);
}
