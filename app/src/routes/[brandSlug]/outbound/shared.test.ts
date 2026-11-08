import { describe, expect, it } from 'vitest';
import type { OutboundCampaignResults, OutboundMailboxHealth } from '$lib/api';
import { mailboxLimit, overBounceLimit, pageRange } from './shared';

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
