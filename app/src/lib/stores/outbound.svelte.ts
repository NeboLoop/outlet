// The brand's outbound overview, for the Outbound badge in the sidebar and
// the outbound row on the brand dashboard. The brand layout loads it with
// the brand and again on each navigation; nothing polls it.

import { getOutboundOverview, type OutboundOverview } from '$lib/api';

let current = $state<OutboundOverview | null>(null);
let currentOrg = '';
// Only the latest load lands: an older answer (another brand's, after a
// switch) never overwrites a newer one.
let latest = 0;

export const outbound = {
	get overview(): OutboundOverview | null {
		return current;
	}
};

// orgId is the brand the request goes out for (the X-Org-Id the API client
// sends): on a switch, the last brand's numbers clear at once.
export async function refreshOutbound(orgId: string): Promise<void> {
	const n = ++latest;
	if (orgId !== currentOrg) {
		currentOrg = orgId;
		current = null;
	}
	try {
		const o = await getOutboundOverview();
		if (n === latest) current = o;
	} catch {
		if (n === latest) current = null;
	}
}

// "3 replies waiting", for the Outbound badge's screen reader text.
export function repliesWaitingLabel(n: number): string {
	return `${n} ${n === 1 ? 'reply' : 'replies'} waiting`;
}
