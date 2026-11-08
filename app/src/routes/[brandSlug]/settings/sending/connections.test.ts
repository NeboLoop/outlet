import { describe, expect, it } from 'vitest';
import type { MailboxConnection } from '$lib/api';
import { connectionTargets, limitText, parsePeople, serviceName } from './connections';

function connection(id: string, pool: string, service: string, limit?: number): MailboxConnection {
	return { id, pool_id: pool, service, name: id, key_set: service !== 'manual', daily_limit: limit, domains: [], mailboxes: 0, created_at: '' };
}

describe('connections', () => {
	const all = [
		connection('md-a', 'pool-1', 'maildoso', 15),
		connection('md-b', 'pool-1', 'maildoso', 15),
		connection('hand', 'pool-1', 'manual'),
		connection('elsewhere', 'pool-2', 'cheapinboxes')
	];

	it('moves a domain only to another connection of its pool', () => {
		expect(connectionTargets(all, all[0]).map((c) => c.id)).toEqual(['md-b', 'hand']);
	});

	it("says the limit a connection's mailboxes send under", () => {
		expect(limitText(all[0])).toBe('15 a day per mailbox');
		expect(limitText(all[2])).toBe("The pool's daily limit");
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
