import { describe, expect, it } from 'vitest';
import type { MailboxConnection } from '$lib/api';
import { connectionTargets, limitText, limitValue, parsePeople, serviceName } from './connections';

function connection(id: string, pool: string, service: string, limit?: number, start?: number, step?: number): MailboxConnection {
	return {
		id,
		pool_id: pool,
		service,
		name: id,
		key_set: service !== 'manual',
		daily_limit: limit,
		warmup_start: start,
		warmup_step: step,
		domains: [],
		mailboxes: 0,
		created_at: ''
	};
}

describe('connections', () => {
	const all = [
		connection('md-a', 'pool-1', 'maildoso', 15, 5, 1),
		connection('md-b', 'pool-1', 'maildoso', 15, 5, 1),
		connection('hand', 'pool-1', 'manual'),
		connection('elsewhere', 'pool-2', 'cheapinboxes')
	];

	it('moves a domain only to another connection of its pool', () => {
		expect(connectionTargets(all, all[0]).map((c) => c.id)).toEqual(['md-b', 'hand']);
	});

	it("says the limit a connection's mailboxes send under", () => {
		expect(limitText(all[0])).toBe('15 a day per mailbox, warming up from 5 by 1 a day');
		expect(limitText(all[2])).toBe("The pool's daily limit, the pool's warm-up");
		expect(limitText(connection('x', 'pool-1', 'maildoso', 15, 0))).toBe('15 a day per mailbox, no warm-up');
	});

	it('reads a limit field, empty for the pool', () => {
		expect(limitValue(15, 1)).toBe(15);
		expect(limitValue(0, 1)).toBeUndefined();
		expect(limitValue(0, 0)).toBe(0);
		expect(limitValue(Number.NaN, 0)).toBeUndefined();
		expect(limitValue(undefined, 1)).toBeUndefined();
	});

	it('names services and reads people', () => {
		expect(serviceName('manual')).toBe('By hand');
		expect(serviceName('cheapinboxes')).toBe('CheapInboxes');
		expect(parsePeople('Sam Rivera\nPat Lee Rivera, pat\n')).toEqual([
			{ first_name: 'Sam', last_name: 'Rivera', local: undefined },
			{ first_name: 'Pat', last_name: 'Lee Rivera', local: 'pat' }
		]);
	});
});
