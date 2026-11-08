import { describe, expect, it } from 'vitest';
import type { SendingPathway } from '$lib/api';
import { moveTargets } from './pathways';

function pathway(id: string, provider: string): SendingPathway {
	return {
		id,
		name: id,
		provider,
		kinds: [],
		aws_keys_set: false,
		smtp_password_set: false,
		mailbox_service_key_set: false,
		limits: {
			daily_limit: 0,
			warmup_start: 0,
			warmup_step: 0,
			min_gap_seconds: 0,
			max_gap_seconds: 0,
			pause_bounce_rate: 0,
			pause_min_bounces: 0,
			pause_spam_blocks: 0,
			pause_complaints: 0
		},
		domains: [],
		mailboxes: 0,
		created_at: ''
	};
}

const all = [
	pathway('platform', 'platform'),
	pathway('ses', 'ses'),
	pathway('smtp', 'smtp'),
	pathway('pool-a', 'mailboxes'),
	pathway('pool-b', 'mailboxes')
];

describe('moveTargets', () => {
	it('moves an outreach domain only to another mailbox pool', () => {
		expect(moveTargets(all, 'pool-a', 'outreach').map((p) => p.id)).toEqual(['pool-b']);
	});

	it('moves a sending domain only to another platform, ses or smtp pathway', () => {
		expect(moveTargets(all, 'platform', 'sending').map((p) => p.id)).toEqual(['ses', 'smtp']);
	});
});
