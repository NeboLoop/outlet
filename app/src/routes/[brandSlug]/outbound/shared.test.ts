import { describe, expect, it } from 'vitest';
import type { OutboundCampaignResults, OutboundMailboxHealth, OutboundOverview, OutboundSetupStep } from '$lib/api';
import {
	askAIPrompt,
	dashboardOutbound,
	hotLeads,
	waitingOnYou,
	nextSetupStep,
	openingStep,
	setupDone,
	setupSkipped,
	skipSetup,
	authorLabel, authorVariant, classificationLabel, joinList, mailboxLimit, materialKindLabel, overBounceLimit, pageRange, splitList } from './shared';

describe('pageRange', () => {
	it('says which rows a page shows', () => {
		expect(pageRange(0, 50, 120)).toBe('1 to 50 of 120');
		expect(pageRange(100, 20, 120)).toBe('101 to 120 of 120');
	});

	it('says when a page shows none', () => {
		expect(pageRange(0, 0, 0)).toBe('0 of 0');
	});
});

describe('mailboxLimit', () => {
	const box = (m: Partial<OutboundMailboxHealth>): OutboundMailboxHealth => ({
		id: 'm1',
		email: 'sam@getexample.com',
		from_name: 'Sam',
		status: 'active',
		sent_today: 0,
		allowed_today: 30,
		sent_7d: 0,
		bounces_7d: 0,
		spam_blocks_7d: 0,
		complaints_7d: 0,
		limited: true,
		limit_in_use: 30,
		warming_up: false,
		domain: 'getexample.com',
		...m
	});

	it("says the pool's limit, or the mailbox's own", () => {
		expect(mailboxLimit(box({}))).toBe("30 a day (the pool's)");
		expect(mailboxLimit(box({ daily_limit: 80, limit_in_use: 80 }))).toBe('80 a day (its own)');
	});

	it('says the warm-up and where it climbs to', () => {
		expect(mailboxLimit(box({ warming_up: true, allowed_today: 7 }))).toBe(
			"Warming up: 7 today, climbing to 30 a day (the pool's)"
		);
	});

	it('says when nothing limits it', () => {
		expect(mailboxLimit(box({ limit_in_use: 0, limited: false }))).toBe('No daily limit');
	});
});

describe('overBounceLimit', () => {
	const campaign = (bounce_rate: number, bounce_limit: number) =>
		({ bounce_rate, bounce_limit }) as OutboundCampaignResults;

	it("uses the outreach pathway's threshold", () => {
		expect(overBounceLimit(campaign(4, 3))).toBe(true);
		expect(overBounceLimit(campaign(4, 5))).toBe(false);
		expect(overBounceLimit(campaign(3, 3))).toBe(false);
	});

	it('never flags when the pathway never pauses', () => {
		expect(overBounceLimit(campaign(40, 0))).toBe(false);
	});
});

describe('classificationLabel', () => {
	it('names a sorted reply and reads an unsorted one as not sorted', () => {
		expect(classificationLabel('interested')).toBe('Interested');
		expect(classificationLabel('needs_review')).toBe('Not sorted');
		expect(classificationLabel('')).toBe('Not sorted');
	});
});

describe('authorLabel', () => {
	it('says who wrote an email, and marks an AI\'s for the closest read', () => {
		expect(authorLabel('ai')).toBe('Written by AI');
		expect(authorLabel('ai_edited')).toBe('AI, edited by a person');
		expect(authorLabel('person')).toBe('Written by a person');
		expect(authorLabel('')).toBe('');
		expect(authorVariant('ai')).toBe('warning');
		expect(authorVariant('person')).toBe('default');
	});
});

describe('lists in a field', () => {
	it('splits on commas and drops blanks', () => {
		expect(splitList(' US, CA ,, ')).toEqual(['US', 'CA']);
		expect(joinList(['US', 'CA'])).toBe('US, CA');
		expect(joinList(undefined)).toBe('');
	});

	it('names material kinds', () => {
		expect(materialKindLabel('case_study')).toBe('Case study');
		expect(materialKindLabel('other')).toBe('other');
	});
});

describe('setup steps', () => {
	const steps = (...done: boolean[]): OutboundSetupStep[] =>
		['business', 'targeting', 'sending', 'do_not_contact', 'campaign'].map((step, i) => ({ step, done: done[i] }));

	it('names the first step not done as next, in order', () => {
		expect(nextSetupStep(steps(true, false, true, false, false))?.step).toBe('targeting');
		expect(nextSetupStep(steps(true, true, true, true, false))?.step).toBe('campaign');
	});

	it('is done only once every step is', () => {
		expect(setupDone(steps(true, true, true, true, true))).toBe(true);
		expect(setupDone(steps(true, true, true, true, false))).toBe(false);
		expect(setupDone([]), 'nothing read yet is not done').toBe(false);
	});

	it('follows the server, which can take a step back', () => {
		// A do-not-contact list deleted after setup: the step is open again.
		const after = steps(true, true, true, false, true);
		expect(setupDone(after)).toBe(false);
		expect(nextSetupStep(after)?.step).toBe('do_not_contact');
	});

	it('opens on the step asked for, else the next one, else the first', () => {
		const s = steps(true, false, false, false, false);
		expect(openingStep(s, 'sending')).toBe('sending');
		expect(openingStep(s, 'nonsense')).toBe('targeting');
		expect(openingStep(s, null)).toBe('targeting');
		expect(openingStep(steps(true, true, true, true, true), null)).toBe('business');
	});
});

describe('setup skipped', () => {
	it('is remembered per brand and can be undone', () => {
		expect(setupSkipped('acme')).toBe(false);
		skipSetup('acme', true);
		expect(setupSkipped('acme')).toBe(true);
		expect(setupSkipped('other')).toBe(false);
		skipSetup('acme', false);
		expect(setupSkipped('acme')).toBe(false);
	});
});

describe('askAIPrompt', () => {
	it('names the brand and what to read or write from', () => {
		expect(askAIPrompt('business', { brand: 'acme', website: 'https://acme.com' })).toContain('"acme": read https://acme.com');
		expect(askAIPrompt('campaign', { brand: 'acme', segment: 'Utah dental', approval: 'first' })).toContain(
			'segment "Utah dental"'
		);
		expect(askAIPrompt('campaign', { brand: 'acme', segment: 'Utah dental', approval: 'first' })).toContain(
			'approval_mode first'
		);
	});
});

describe('the dashboard outbound row', () => {
	const steps = (done: boolean): OutboundSetupStep[] =>
		['business', 'targeting', 'sending', 'do_not_contact', 'campaign'].map((step) => ({ step, done }));
	const overview = (o: Partial<OutboundOverview>) =>
		({ enabled: true, setup: steps(true), drafts_waiting: 0, replies_waiting: 0, campaigns: [], ...o }) as OutboundOverview;

	it('shows nothing while outbound is off or not loaded', () => {
		expect(dashboardOutbound(null)).toBe('off');
		expect(dashboardOutbound(overview({ enabled: false }))).toBe('off');
		expect(dashboardOutbound(overview({ enabled: false, setup: steps(false) }))).toBe('off');
	});

	it('shows the setup card until setup is done, then the numbers', () => {
		expect(dashboardOutbound(overview({ setup: steps(false) }))).toBe('setup');
		const oneLeft = steps(true);
		oneLeft[4].done = false;
		expect(dashboardOutbound(overview({ setup: oneLeft }))).toBe('setup');
		expect(dashboardOutbound(overview({}))).toBe('running');
	});

	it('adds hot leads across campaigns', () => {
		expect(hotLeads([])).toBe(0);
		expect(hotLeads([{ interested: 2 }, { interested: 3 }] as OutboundCampaignResults[])).toBe(5);
	});

	it('leaves out a waiting count of zero', () => {
		expect(waitingOnYou(overview({}), '/acme/outbound')).toEqual([]);
		expect(waitingOnYou(overview({ drafts_waiting: 3 }), '/acme/outbound')).toEqual([
			{ label: '3 drafts to approve', href: '/acme/outbound/drafts' }
		]);
		expect(waitingOnYou(overview({ replies_waiting: 1 }), '/acme/outbound')).toEqual([
			{ label: '1 reply to answer', href: '/acme/outbound/replies' }
		]);
		expect(waitingOnYou(overview({ drafts_waiting: 1, replies_waiting: 2 }), '/acme/outbound').map((w) => w.label)).toEqual([
			'1 draft to approve',
			'2 replies to answer'
		]);
	});
});
