<script lang="ts">
	import { page } from '$app/stores';
	import { getOutboundCampaign, pauseOutbound, type OutboundCampaignResults } from '$lib/api';
	import { Button, Alert, LoadingSpinner, Badge } from '$lib/components/ui';
	import { ArrowLeft, Pause, Play } from '@lucide/svelte';
	import { authorLabel, authorVariant, classificationLabel, errorMessage, materialKindLabel, overBounceLimit, percent } from '../../shared';

	let loading = $state(true);
	let error = $state('');
	let busy = $state(false);
	let campaign = $state<OutboundCampaignResults | null>(null);

	let base = $derived(`/${$page.params.brandSlug}/outbound`);
	let id = $derived($page.params.id || '');
	let kinds = $derived(Object.entries(campaign?.replies_by_kind || {}).filter(([, n]) => n > 0));
	let stopped = $derived(Object.entries(campaign?.stopped_by_reason || {}).filter(([, n]) => n > 0));

	$effect(() => {
		if (id) load(id);
	});

	async function load(campaignId: string) {
		loading = true;
		error = '';
		try {
			campaign = await getOutboundCampaign({}, campaignId);
		} catch (err) {
			error = errorMessage(err, 'Could not load the campaign');
		} finally {
			loading = false;
		}
	}

	async function togglePause() {
		if (!campaign) return;
		busy = true;
		error = '';
		try {
			await pauseOutbound({ scope: 'campaign', id: campaign.id, resume: campaign.status === 'paused' });
			campaign = await getOutboundCampaign({}, campaign.id);
		} catch (err) {
			error = errorMessage(err, 'Could not change the campaign');
		} finally {
			busy = false;
		}
	}
</script>

<a href="{base}/campaigns" class="inline-flex items-center text-sm text-text-muted hover:text-text mb-4">
	<ArrowLeft class="mr-1 h-4 w-4" />
	Campaigns
</a>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if campaign}
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
		<div class="flex items-center gap-3">
			<h2 class="text-xl font-semibold text-text">{campaign.name}</h2>
			<Badge variant={campaign.status === 'active' ? 'success' : 'warning'}>{campaign.status}</Badge>
			{#if campaign.authored_by}
				<Badge variant={authorVariant(campaign.authored_by)}>{authorLabel(campaign.authored_by)}</Badge>
			{/if}
		</div>
		<Button type="secondary" size="sm" onclick={togglePause} disabled={busy}>
			{#if campaign.status === 'paused'}
				<Play class="mr-2 h-4 w-4" />
				Resume
			{:else}
				<Pause class="mr-2 h-4 w-4" />
				Pause
			{/if}
		</Button>
	</div>

	<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Reached</p>
			<p class="text-2xl font-semibold text-text">{campaign.reached} <span class="text-sm font-normal text-text-muted">of {campaign.enrolled}</span></p>
		</div>
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Replied</p>
			<p class="text-2xl font-semibold text-text">{campaign.replied} <span class="text-sm font-normal text-text-muted">{percent(campaign.reply_rate)}</span></p>
		</div>
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Interested</p>
			<p class="text-2xl font-semibold text-text">{campaign.interested}</p>
		</div>
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Partners</p>
			<p class="text-2xl font-semibold text-text">{campaign.partners}</p>
		</div>
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Opened</p>
			{#if campaign.track_opens}
				<p class="text-2xl font-semibold text-text">{campaign.opened} <span class="text-sm font-normal text-text-muted">{percent(campaign.open_rate)}</span></p>
			{:else}
				<p class="text-2xl font-semibold text-text-muted">Off</p>
			{/if}
		</div>
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Clicked</p>
			{#if campaign.track_clicks}
				<p class="text-2xl font-semibold text-text">{campaign.clicked} <span class="text-sm font-normal text-text-muted">{percent(campaign.click_rate)}</span></p>
			{:else}
				<p class="text-2xl font-semibold text-text-muted">Off</p>
			{/if}
		</div>
		<div class="rounded-lg border p-4" class:border-border={!overBounceLimit(campaign)} class:border-error={overBounceLimit(campaign)}>
			<p class="text-xs text-text-muted">Bounced</p>
			<p class="text-2xl font-semibold" class:text-text={!overBounceLimit(campaign)} class:text-error={overBounceLimit(campaign)}>
				{campaign.bounced} <span class="text-sm font-normal">{percent(campaign.bounce_rate)}</span>
			</p>
		</div>
		<div class="rounded-lg border p-4" class:border-border={campaign.spam_blocks === 0} class:border-error={campaign.spam_blocks > 0}>
			<p class="text-xs text-text-muted">Spam blocks</p>
			<p class="text-2xl font-semibold" class:text-text={campaign.spam_blocks === 0} class:text-error={campaign.spam_blocks > 0}>{campaign.spam_blocks}</p>
		</div>
	</div>

	<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6 text-sm">
		<div><span class="text-text-muted">Opted out:</span> {campaign.opted_out}</div>
		<div><span class="text-text-muted">In progress:</span> {campaign.in_progress}</div>
		<div><span class="text-text-muted">Finished, no reply:</span> {campaign.finished_no_reply}</div>
		<div><a href="{base}/drafts" class="text-primary">Drafts waiting: {campaign.drafts_waiting}</a></div>
		{#if campaign.segment}
			<div><span class="text-text-muted">Segment:</span> <a href="{base}/targeting" class="text-primary">{campaign.segment}</a></div>
		{/if}
	</div>
	{#if campaign.track_opens}
		<p class="form-hint mb-6">
			Apple Mail opens every email automatically, so open rates run high. Reply rate is the measure that counts.
		</p>
	{/if}

	<h3 class="text-lg font-semibold text-text mb-3">Steps</h3>
	{#if campaign.steps?.length}
		<div class="data-table overflow-x-auto mb-6">
			<table class="w-full text-sm">
				<thead>
					<tr>
						<th class="text-left">Step</th>
						<th class="text-right">Sent</th>
						<th class="text-right">Replies</th>
						<th class="text-right">Reply rate</th>
						<th class="text-right">Opened</th>
						<th class="text-right">Clicked</th>
					</tr>
				</thead>
				<tbody>
					{#each campaign.steps as s (s.step)}
						<tr>
							<td>Step {s.step}</td>
							<td class="text-right">{s.sent}</td>
							<td class="text-right">{s.replies}</td>
							<td class="text-right">{percent(s.reply_rate)}</td>
							<td class="text-right">{campaign.track_opens ? (s.opened ?? 0) : 'Off'}</td>
							<td class="text-right">{campaign.track_clicks ? (s.clicked ?? 0) : 'Off'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else}
		<p class="text-sm text-text-muted mb-6">Nothing sent yet.</p>
	{/if}

	<h3 class="text-lg font-semibold text-text mb-3">Replies by kind</h3>
	{#if kinds.length}
		<div class="flex flex-wrap gap-2">
			{#each kinds as [kind, n]}
				<Badge variant="default">{classificationLabel(kind)}: {n}</Badge>
			{/each}
		</div>
	{:else}
		<p class="text-sm text-text-muted">No replies yet.</p>
	{/if}

	{#if stopped.length}
		<h3 class="text-lg font-semibold text-text mt-6 mb-3">Stopped early</h3>
		<p class="form-hint mb-2">People whose emails stopped before the last step, and why.</p>
		<div class="flex flex-wrap gap-2">
			{#each stopped as [reason, n]}
				<Badge variant={reason.startsWith('do not contact') ? 'warning' : 'default'}>{reason}: {n}</Badge>
			{/each}
		</div>
	{/if}

	{#if campaign.materials?.length}
		<h3 class="text-lg font-semibold text-text mt-6 mb-3">Written from</h3>
		<ul class="space-y-2">
			{#each campaign.materials as m (m.id)}
				<li class="text-sm">
					<Badge variant="default" size="sm">{materialKindLabel(m.kind)}</Badge>
					<span class="font-medium text-text ml-1">{m.title}</span>
					<span class="text-text-muted">: {m.summary}</span>
				</li>
			{/each}
		</ul>
	{/if}
{/if}
