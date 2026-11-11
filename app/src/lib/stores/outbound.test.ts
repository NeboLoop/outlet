import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { OutboundOverview } from '$lib/api';

const getOutboundOverview = vi.fn();
vi.mock('$lib/api', () => ({ getOutboundOverview: () => getOutboundOverview() }));

const { outbound, refreshOutbound, repliesWaitingLabel } = await import('./outbound.svelte');

const overview = (replies_waiting: number) => ({ enabled: true, replies_waiting }) as OutboundOverview;

describe('the outbound store', () => {
	beforeEach(() => getOutboundOverview.mockReset());

	it('holds the overview it loads', async () => {
		getOutboundOverview.mockResolvedValueOnce(overview(3));
		await refreshOutbound('org1');
		expect(outbound.overview?.replies_waiting).toBe(3);
	});

	it('keeps the newest answer when an older one lands after it', async () => {
		let answerOld!: (o: OutboundOverview) => void;
		getOutboundOverview.mockReturnValueOnce(new Promise((r) => (answerOld = r)));
		getOutboundOverview.mockResolvedValueOnce(overview(7));
		const older = refreshOutbound('org1');
		await refreshOutbound('org1');
		answerOld(overview(1));
		await older;
		expect(outbound.overview?.replies_waiting).toBe(7);
	});

	it("clears the last brand's numbers on a switch", async () => {
		getOutboundOverview.mockResolvedValueOnce(overview(4));
		await refreshOutbound('org1');
		getOutboundOverview.mockReturnValueOnce(new Promise(() => {}));
		refreshOutbound('org2');
		expect(outbound.overview).toBeNull();
	});

	it('holds nothing when the overview fails to load', async () => {
		getOutboundOverview.mockResolvedValueOnce(overview(2));
		await refreshOutbound('org3');
		getOutboundOverview.mockRejectedValueOnce(new Error('no'));
		await refreshOutbound('org3');
		expect(outbound.overview).toBeNull();
	});
});

describe('repliesWaitingLabel', () => {
	it('says how many replies wait', () => {
		expect(repliesWaitingLabel(1)).toBe('1 reply waiting');
		expect(repliesWaitingLabel(3)).toBe('3 replies waiting');
	});
});
