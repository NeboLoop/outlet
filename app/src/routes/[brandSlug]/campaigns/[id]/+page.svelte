<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getCampaign,
		getCampaignStats,
		sendCampaignNow,
		scheduleCampaign,
		deleteCampaign,
		listLists,
		type CampaignInfo,
		type CampaignLinkStat,
		type ListInfo
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Badge, Card, Input, AlertDialog } from '$lib/components/ui';
	import CampaignEditor from '$lib/components/campaigns/CampaignEditor.svelte';
	import { ArrowLeft, Send, Clock, Edit, Trash2 } from '@lucide/svelte';

	let id = $derived($page.params.id ?? '');
	let basePath = $derived(`/${$page.params.brandSlug}`);

	let loading = $state(true);
	let error = $state('');
	let campaign = $state<CampaignInfo | null>(null);
	let links = $state<CampaignLinkStat[]>([]);
	let lists = $state<ListInfo[]>([]);

	let editing = $state(false);
	let confirmSend = $state(false);
	let confirmDelete = $state(false);
	let busy = $state(false);
	let scheduleAt = $state('');

	let editable = $derived(campaign?.status === 'draft' || campaign?.status === 'scheduled');
	let audience = $derived(
		lists.filter((l) => campaign?.list_ids?.includes(l.id)).reduce((sum, l) => sum + l.subscriber_count, 0)
	);

	$effect(() => {
		load(id);
	});

	// Refresh while the campaign is going out, so progress shows live
	$effect(() => {
		if (campaign?.status !== 'sending' && campaign?.status !== 'scheduled') return;
		const timer = setInterval(() => refresh(id), 5000);
		return () => clearInterval(timer);
	});

	async function load(campaignId: string) {
		loading = true;
		error = '';
		try {
			const [c, l] = await Promise.all([getCampaign({}, campaignId), listLists()]);
			campaign = c;
			lists = l.lists || [];
			await loadLinks(campaignId);
		} catch (err: any) {
			error = err.message || 'Failed to load campaign';
		} finally {
			loading = false;
		}
	}

	async function refresh(campaignId: string) {
		try {
			campaign = await getCampaign({}, campaignId);
			await loadLinks(campaignId);
		} catch {
			// keep showing the last state; the next refresh tries again
		}
	}

	async function loadLinks(campaignId: string) {
		if (campaign && campaign.status !== 'draft') {
			const stats = await getCampaignStats({}, campaignId).catch(() => null);
			links = stats?.links || [];
		}
	}

	async function doSendNow() {
		busy = true;
		error = '';
		try {
			campaign = await sendCampaignNow({}, id);
			confirmSend = false;
		} catch (err: any) {
			error = err.message || 'Failed to send';
		} finally {
			busy = false;
		}
	}

	async function doSchedule() {
		if (!scheduleAt) return;
		busy = true;
		error = '';
		try {
			campaign = await scheduleCampaign({}, { scheduled_at: new Date(scheduleAt).toISOString() }, id);
			scheduleAt = '';
		} catch (err: any) {
			error = err.message || 'Failed to schedule';
		} finally {
			busy = false;
		}
	}

	async function doDelete() {
		busy = true;
		try {
			await deleteCampaign({}, id);
			goto(`${basePath}/campaigns`);
		} catch (err: any) {
			error = err.message || 'Failed to delete';
			busy = false;
		}
	}

	function rate(n: number, of: number): string {
		return of > 0 ? `${((n / of) * 100).toFixed(1)}%` : '–';
	}

	function when(iso?: string): string {
		return iso ? new Date(iso.includes('T') ? iso : iso.replace(' ', 'T') + 'Z').toLocaleString() : '';
	}

	const statusVariant: Record<string, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
		draft: 'default',
		scheduled: 'info',
		sending: 'warning',
		sent: 'success',
		paused: 'warning',
		cancelled: 'error'
	};
</script>

<svelte:head>
	<title>{campaign?.name || 'Campaign'} | Outlet</title>
</svelte:head>

{#if editing && campaign}
	<CampaignEditor
		{campaign}
		onsaved={(c) => {
			campaign = c;
			editing = false;
		}}
		onclose={() => (editing = false)}
	/>
{/if}

<div class="p-4 sm:p-6 max-w-5xl mx-auto">
	<a href="{basePath}/campaigns" class="inline-flex items-center gap-1 text-sm text-text-muted hover:text-text mb-4">
		<ArrowLeft class="h-4 w-4" /> Campaigns
	</a>

	{#if error}
		<Alert type="error" title="Error" class="mb-4"><p>{error}</p></Alert>
	{/if}

	{#if loading}
		<div class="flex justify-center py-12"><LoadingSpinner size="large" /></div>
	{:else if campaign}
		<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
			<div class="min-w-0">
				<div class="flex items-center gap-2 flex-wrap">
					<h1 class="text-2xl font-semibold text-text truncate">{campaign.name}</h1>
					<Badge variant={statusVariant[campaign.status] || 'default'} size="sm">{campaign.status}</Badge>
				</div>
				<p class="mt-1 text-sm text-text-muted truncate">{campaign.subject}</p>
				{#if campaign.status === 'scheduled' && campaign.scheduled_at}
					<p class="mt-1 text-sm text-text-muted">Goes out {when(campaign.scheduled_at)}</p>
				{:else if campaign.completed_at}
					<p class="mt-1 text-sm text-text-muted">Finished {when(campaign.completed_at)}</p>
				{:else if campaign.started_at}
					<p class="mt-1 text-sm text-text-muted">Started {when(campaign.started_at)}</p>
				{/if}
			</div>
			{#if editable}
				<div class="flex flex-wrap gap-2">
					<Button type="secondary" onclick={() => (editing = true)}>
						<Edit class="mr-2 h-4 w-4" /> Edit
					</Button>
					<Button type="primary" onclick={() => (confirmSend = true)} disabled={busy || campaign.list_ids.length === 0}>
						<Send class="mr-2 h-4 w-4" /> Send now
					</Button>
				</div>
			{/if}
		</div>

		{#if editable}
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
				<Card title="Audience">
					<p class="text-3xl font-semibold text-text">{audience.toLocaleString()}</p>
					<p class="text-sm text-text-muted mt-1">
						subscribers on {campaign.list_ids.length} list{campaign.list_ids.length === 1 ? '' : 's'}. Unsubscribed, bounced and
						suppressed addresses are skipped.
					</p>
				</Card>
				<Card title="Schedule">
					<div class="flex flex-col sm:flex-row gap-2">
						<Input type="datetime-local" bind:value={scheduleAt} class="flex-1" />
						<Button type="secondary" onclick={doSchedule} disabled={!scheduleAt || busy}>
							<Clock class="mr-2 h-4 w-4" /> Schedule
						</Button>
					</div>
					<p class="text-xs text-text-muted mt-2">Your local time: {Intl.DateTimeFormat().resolvedOptions().timeZone}</p>
				</Card>
			</div>
		{/if}

		{#if campaign.status !== 'draft'}
			{@const c = campaign}
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
				{#each [
					['Recipients', c.recipients_count.toLocaleString(), ''],
					['Sent', c.sent_count.toLocaleString(), rate(c.sent_count, c.recipients_count)],
					['Opened', c.opened_count.toLocaleString(), rate(c.opened_count, c.sent_count)],
					['Clicked', c.clicked_count.toLocaleString(), rate(c.clicked_count, c.sent_count)],
					['Bounced', c.bounced_count.toLocaleString(), rate(c.bounced_count, c.sent_count)],
					['Complaints', c.complained_count.toLocaleString(), rate(c.complained_count, c.sent_count)],
					['Unsubscribed', c.unsubscribed_count.toLocaleString(), rate(c.unsubscribed_count, c.sent_count)],
					['Delivered', c.delivered_count.toLocaleString(), rate(c.delivered_count, c.sent_count)]
				] as [label, value, pct]}
					<div class="rounded-lg border border-border p-3">
						<div class="text-xs text-text-muted">{label}</div>
						<div class="text-lg font-semibold text-text">{value}</div>
						{#if pct}<div class="text-xs text-text-muted">{pct}</div>{/if}
					</div>
				{/each}
			</div>
			<p class="text-xs text-text-muted -mt-4 mb-6">Apple Mail opens emails automatically, so open rates run high. Clicks are the reliable signal.</p>

			{#if links.length > 0}
				<Card title="Links clicked">
					<div class="overflow-x-auto">
						<table class="w-full text-sm">
							<tbody>
								{#each links as link}
									<tr class="border-b border-border/50">
										<td class="py-2 pr-4 break-all">{link.url}</td>
										<td class="py-2 text-right whitespace-nowrap">{link.click_count}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</Card>
			{/if}
		{/if}

		<Card title="Email">
			<iframe title="Campaign preview" sandbox="" srcdoc={campaign.html_body} class="w-full h-96 rounded border border-border bg-white"></iframe>
		</Card>

		{#if campaign.status === 'draft'}
			<div class="mt-6">
				<Button type="danger" onclick={() => (confirmDelete = true)}>
					<Trash2 class="mr-2 h-4 w-4" /> Delete draft
				</Button>
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={confirmSend}
	title="Send this campaign now?"
	description={`It goes to about ${audience.toLocaleString()} subscribers within a minute. This can't be undone.`}
	actionLabel={busy ? 'Sending...' : 'Send now'}
	actionType="primary"
	onAction={doSendNow}
	onCancel={() => (confirmSend = false)}
/>

<AlertDialog
	bind:open={confirmDelete}
	title="Delete draft"
	description={`Delete "${campaign?.name || ''}"? This can't be undone.`}
	actionLabel="Delete"
	actionType="danger"
	onAction={doDelete}
	onCancel={() => (confirmDelete = false)}
/>
