// Helpers shared by the Outbound screens.

import type { OutboundCampaignResults, OutboundMailboxHealth } from '$lib/api';

export function errorMessage(err: unknown, fallback: string): string {
	return err instanceof Error && err.message ? err.message : fallback;
}

// The API sends rates as percents (0 to 100).
export function percent(rate: number): string {
	return `${(rate || 0).toFixed(1)}%`;
}

// Which rows of a list a page shows: "1 to 50 of 120".
export function pageRange(offset: number, shown: number, total: number): string {
	if (shown === 0) return `0 of ${total}`;
	return `${offset + 1} to ${offset + shown} of ${total}`;
}

// A mailbox's daily limit in words: its own or its pool's, and the warm-up
// that holds it below that limit.
export function mailboxLimit(m: OutboundMailboxHealth): string {
	const limit = m.limit_in_use ? `${m.limit_in_use} a day` : 'no daily limit';
	const whose = m.daily_limit ? 'its own' : "the pool's";
	if (m.warming_up) return `Warming up: ${m.allowed_today} today, climbing to ${limit} (${whose})`;
	return m.limit_in_use ? `${limit} (${whose})` : 'No daily limit';
}

// Whether a campaign bounces above the rate at which its outreach pathway
// pauses a mailbox (a limit of 0 never pauses).
export function overBounceLimit(c: OutboundCampaignResults): boolean {
	return c.bounce_limit > 0 && c.bounce_rate > c.bounce_limit;
}

export const CLASSIFICATIONS: { value: string; label: string }[] = [
	{ value: 'interested', label: 'Interested' },
	{ value: 'not_interested', label: 'Not interested' },
	{ value: 'question', label: 'Question' },
	{ value: 'out_of_office', label: 'Out of office' },
	{ value: 'wrong_person', label: 'Wrong person' },
	{ value: 'unsubscribe', label: 'Unsubscribe' },
	{ value: 'complaint', label: 'Complaint' }
];

// A reply the rules couldn't sort waits as needs_review: it reads as not sorted.
export function classificationLabel(value: string): string {
	if (!value || value === 'needs_review') return 'Not sorted';
	return CLASSIFICATIONS.find((c) => c.value === value)?.label || value;
}
