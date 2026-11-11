<script lang="ts">
	import { getOutboundPace, setOutboundPace, type OutboundPace } from '$lib/api';
	import { Button, Alert, LoadingSpinner, Input, Radio } from '$lib/components/ui';
	import { errorMessage } from './shared';

	// The pace dial: how many new prospects get their first email each day
	// across the mailbox pool. Past it, first emails wait for the next day's
	// window and follow-ups go on; webhooks hear outreach.paused once.

	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let notice = $state('');
	let pace = $state<OutboundPace | null>(null);
	let choice = $state('unlimited');
	let custom = $state('');

	const labels: Record<string, string> = {
		light: 'Light',
		typical: 'Typical',
		growth: 'Growth',
		unlimited: 'Unlimited',
		custom: 'Custom'
	};

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			show(await getOutboundPace());
		} catch (err) {
			// No mailbox pool carries outreach yet: the dial has nothing to set.
			error = errorMessage(err, 'Could not load the pace');
		} finally {
			loading = false;
		}
	}

	function show(p: OutboundPace) {
		pace = p;
		choice = p.current.pace;
		custom = p.current.pace === 'custom' ? String(p.current.daily_new_prospects) : '';
	}

	async function save() {
		saving = true;
		error = '';
		notice = '';
		try {
			const req =
				choice === 'custom' ? { daily_new_prospects: Math.max(0, Number(custom) || 0) } : { pace: choice };
			show(await setOutboundPace(req));
			notice = 'Pace saved.';
		} catch (err) {
			error = errorMessage(err, 'Could not save the pace');
		} finally {
			saving = false;
		}
	}

	function perDay(n: number): string {
		return n > 0 ? `${n} new prospects a day` : 'No daily limit';
	}
</script>

<section>
	<h2 class="text-lg font-semibold text-text mb-1">Daily pace</h2>
	<p class="form-hint mb-3">
		How many new people get their first email each day. Past it, first emails wait for the next day and follow-ups
		still go, so pausing never loses anything. Webhooks get <code>outreach.paused</code> the first time each day.
	</p>
	{#if error}
		<Alert type="warning" class="mb-3">
			<p>{error}</p>
		</Alert>
	{/if}
	{#if notice}
		<Alert type="success" dismissible onclose={() => (notice = '')} class="mb-3">
			<p>{notice}</p>
		</Alert>
	{/if}
	{#if loading}
		<div class="flex justify-center py-6">
			<LoadingSpinner />
		</div>
	{:else if pace}
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
			{#each [...pace.presets, { pace: 'custom', daily_new_prospects: 0, sends_per_day: 0, mailboxes_needed: 0 }] as p (p.pace)}
				<div class="rounded-lg border p-3" class:border-primary={choice === p.pace} class:border-border={choice !== p.pace}>
					<Radio name="pace" value={p.pace} checked={choice === p.pace} onchange={() => (choice = p.pace)} label={labels[p.pace] || p.pace} />
					{#if p.pace === 'custom'}
						<div class="mt-2">
							<Input type="number" min={0} bind:value={custom} placeholder="New prospects a day (0: no limit)" onfocus={() => (choice = 'custom')} />
						</div>
					{:else}
						<p class="text-sm text-text mt-1">{perDay(p.daily_new_prospects)}</p>
						{#if p.daily_new_prospects > 0}
							<p class="text-xs text-text-muted">
								Up to {p.sends_per_day} emails a day once follow-ups catch up{#if p.mailboxes_needed}, about
									{p.mailboxes_needed} mailboxes at {pace.mailbox_daily_limit} a day each{/if}.
							</p>
						{/if}
					{/if}
				</div>
			{/each}
		</div>
		<div class="grid grid-cols-1 sm:grid-cols-3 gap-3 mt-3 text-sm">
			<div><span class="text-text-muted">Today:</span> {pace.new_today} new{#if pace.current.daily_new_prospects} of {pace.current.daily_new_prospects}{/if}</div>
			<div><span class="text-text-muted">Active mailboxes:</span> {pace.mailboxes_active}</div>
			<div><span class="text-text-muted">They may send today:</span> {pace.capacity || 'no limit'}</div>
		</div>
		<p class="form-hint mt-2">Cost: {pace.cost}</p>
		<div class="flex justify-end mt-3">
			<Button type="primary" onclick={save} disabled={saving}>{saving ? 'Saving...' : 'Save pace'}</Button>
		</div>
	{/if}
</section>
