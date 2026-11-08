// Helpers for a mailbox pool's connections on the Sending settings screen.

import type { MailboxConnection, MailboxServicePerson } from '$lib/api';

export const SERVICES = [
	{ value: 'manual', label: 'By hand (Google Workspace, Microsoft 365, any SMTP and IMAP)' },
	{ value: 'maildoso', label: 'Maildoso' },
	{ value: 'cheapinboxes', label: 'CheapInboxes' },
	{ value: 'infraforge', label: 'Infraforge' }
];

// serviceName is a service as people say it.
export function serviceName(service: string): string {
	if (service === 'manual') return 'By hand';
	return SERVICES.find((s) => s.value === service)?.label || service;
}

// The connections a domain can move to: the others of its own pool.
export function connectionTargets(all: MailboxConnection[], from: MailboxConnection): MailboxConnection[] {
	return all.filter((c) => c.pool_id === from.pool_id && c.id !== from.id);
}

// The daily limit a connection's mailboxes send under, in words.
export function limitText(c: MailboxConnection): string {
	return c.daily_limit ? `${c.daily_limit} a day per mailbox` : "The pool's daily limit";
}

// Each line is "First Last", or "First Last, local" to choose the address.
export function parsePeople(text: string): MailboxServicePerson[] {
	return text
		.split('\n')
		.map((line) => line.trim())
		.filter(Boolean)
		.map((line) => {
			const [name, local] = line.split(',').map((s) => s.trim());
			const parts = name.split(/\s+/);
			return { first_name: parts[0] || '', last_name: parts.slice(1).join(' '), local: local || undefined };
		});
}
