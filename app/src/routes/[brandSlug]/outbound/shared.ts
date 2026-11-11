// Helpers shared by the Outbound screens.

import type { OutboundCampaignResults, OutboundMailboxHealth, OutboundOverview, OutboundSetupStep } from '$lib/api';

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

// Who wrote an outreach email, for the reviewer: an AI's email needs the
// closest read, one a person edited less, a person's own the least.
export function authorLabel(authoredBy: string): string {
	switch (authoredBy) {
		case 'ai':
			return 'Written by AI';
		case 'ai_edited':
			return 'AI, edited by a person';
		case 'person':
			return 'Written by a person';
	}
	return '';
}

export function authorVariant(authoredBy: string): 'warning' | 'info' | 'default' {
	if (authoredBy === 'ai') return 'warning';
	if (authoredBy === 'ai_edited') return 'info';
	return 'default';
}

export const MATERIAL_KINDS: { value: string; label: string }[] = [
	{ value: 'case_study', label: 'Case study' },
	{ value: 'proof_point', label: 'Proof point' },
	{ value: 'offer', label: 'Offer' },
	{ value: 'link', label: 'Link' },
	{ value: 'asset', label: 'Asset' }
];

// How a campaign's emails are approved before they go.
export const APPROVAL_MODES: { value: string; label: string }[] = [
	{ value: 'every', label: 'Approve every email' },
	{ value: 'first', label: 'Approve the first email only' },
	{ value: 'automatic', label: 'Send without approval' }
];

export function materialKindLabel(kind: string): string {
	return MATERIAL_KINDS.find((k) => k.value === kind)?.label || kind;
}

// A comma-separated field as a list, and back.
export function splitList(value: string): string[] {
	return value
		.split(',')
		.map((v) => v.trim())
		.filter(Boolean);
}

export function joinList(values?: string[]): string {
	return (values || []).join(', ');
}

// The guided setup's five steps, in order. Whether each is done comes from
// the server (the overview's setup), read from the brand's data.
export const SETUP_STEPS: { step: string; label: string; title: string }[] = [
	{ step: 'business', label: 'Business', title: 'Your business' },
	{ step: 'targeting', label: 'Targeting', title: 'Who to target' },
	{ step: 'sending', label: 'Sending', title: 'Sending' },
	{ step: 'do_not_contact', label: 'Do not contact', title: 'Who never to contact' },
	{ step: 'campaign', label: 'Campaign', title: 'First campaign' }
];

export function setupTitle(step: string): string {
	return SETUP_STEPS.find((s) => s.step === step)?.title || step;
}

// The first step not done; undefined once setup is done.
export function nextSetupStep(setup: OutboundSetupStep[]): OutboundSetupStep | undefined {
	return SETUP_STEPS.map((s) => setup.find((x) => x.step === s.step)).find((x) => x && !x.done);
}

export function setupDone(setup: OutboundSetupStep[]): boolean {
	return setup.length > 0 && nextSetupStep(setup) === undefined;
}

// Which step the setup opens on: the one asked for, else the first not
// done, else the first.
export function openingStep(setup: OutboundSetupStep[], asked: string | null): string {
	if (asked && SETUP_STEPS.some((s) => s.step === asked)) return asked;
	return nextSetupStep(setup)?.step || SETUP_STEPS[0].step;
}

// Whether this browser was told to stop opening the guided setup for a
// brand. It only stops the setup opening by itself: whether setup is done
// never comes from here.
const skipKey = (brand: string) => `outboundSetupSkipped_${brand}`;

export function setupSkipped(brand: string): boolean {
	try {
		return localStorage.getItem(skipKey(brand)) === '1';
	} catch {
		return false;
	}
}

export function skipSetup(brand: string, skipped: boolean): void {
	try {
		if (skipped) localStorage.setItem(skipKey(brand), '1');
		else localStorage.removeItem(skipKey(brand));
	} catch {
		// Storage is off: the setup opens again next time, nothing else changes.
	}
}

// What to ask an AI connected to Outlet over MCP for, in each step it can
// do. Outlet itself doesn't browse or write: the AI does, through Outlet's
// tools, and the setup shows what it saved.
export function askAIPrompt(
	step: 'business' | 'targeting' | 'campaign',
	o: { brand: string; website?: string; segment?: string; approval?: string }
): string {
	const brand = `In Outlet, on the brand "${o.brand}"`;
	switch (step) {
		case 'business':
			return `${brand}: read ${o.website || 'our website'} (home, pricing, customers, about) and fill in the brand profile: what we sell, who buys it, our proof, and how our emails sound. explain(topic: start_from_website) has the steps.`;
		case 'targeting':
			return `${brand}: from the brand profile, propose 2 to 4 segments of who buys from us, in plain words with filters, and create them. I'll keep, edit or remove them in Outlet.`;
		case 'campaign':
			return `${brand}: draft our first outreach campaign on the segment "${o.segment || 'Outreach'}" with draft_outreach. Read its brief, write from its materials, and show me a dry run of the first emails. If the segment has no prospects free to enroll, find people who fit it and add them with add_prospects first, saying where each address came from. Then save it with approval_mode ${o.approval || 'every'}.`;
	}
}

// What the brand dashboard shows for outbound: nothing while it is off, the
// "Finish setup" card until setup is done, then the numbers.
export function dashboardOutbound(o: OutboundOverview | null): 'off' | 'setup' | 'running' {
	if (!o || !o.enabled) return 'off';
	return nextSetupStep(o.setup || []) ? 'setup' : 'running';
}

// Hot leads are people who replied interested, across every campaign.
export function hotLeads(campaigns: OutboundCampaignResults[]): number {
	return campaigns.reduce((n, c) => n + c.interested, 0);
}

// What waits on the person, each with the tab it is done on. A count of
// zero is left out.
export function waitingOnYou(o: OutboundOverview, base: string): { label: string; href: string }[] {
	const out: { label: string; href: string }[] = [];
	if (o.drafts_waiting > 0) {
		out.push({ label: `${o.drafts_waiting} ${o.drafts_waiting === 1 ? 'draft' : 'drafts'} to approve`, href: `${base}/drafts` });
	}
	if (o.replies_waiting > 0) {
		out.push({ label: `${o.replies_waiting} ${o.replies_waiting === 1 ? 'reply' : 'replies'} to answer`, href: `${base}/replies` });
	}
	return out;
}
