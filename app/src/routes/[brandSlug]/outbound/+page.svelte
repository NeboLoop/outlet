<script lang="ts">
	import { page } from '$app/stores';
	import {
		getOutboundOverview,
		pauseOutbound,
		explainOutbound,
		type OutboundOverview,
		type OutboundMailboxHealth
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Badge, EmptyState } from '$lib/components/ui';
	import { Send, Pause, Play } from '@lucide/svelte';
	import CampaignTable from './CampaignTable.svelte';
	import { errorMessage, mailboxLimit, percent } from './shared';

	let loading = $state(true);
	let error = $state('');
	let overview = $state<OutboundOverview | null>(null);
	let busy = $state('');
	let explained = $state<{ email: string; answer: string } | null>(null);

	let base = $derived(`/${$page.params.brandSlug}/outbound`);

	let sentToday = $derived((overview?.mailboxes || []).reduce((n, m) => n + m.sent_today, 0));
	let allowedToday = $derived((overview?.mailboxes || []).reduce((n, m) => n + m.allowed_today, 0));

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			overview = await getOutboundOverview();
		} catch (err) {
			error = errorMessage(err, 'Could not load outbound');
		} finally {
			loading = false;
		}
	}

	async function pause(scope: 'all' | 'mailbox', resume: boolean, id?: string) {
		busy = `${scope}:${id || ''}`;
		error = '';
		try {
			await pauseOutbound({ scope, id, resume });
			overview = await getOutboundOverview();
		} catch (err) {
			error = errorMessage(err, resume ? 'Could not resume' : 'Could not pause');
		} finally {
			busy = '';
		}
	}

	async function explain(m: OutboundMailboxHealth) {
		busy = `explain:${m.id}`;
		error = '';
		try {
			const res = await explainOutbound({ mailbox_id: m.id });
			explained = { email: m.email, answer: res.answer };
		} catch (err) {
			error = errorMessage(err, 'Could not explain this mailbox');
		} finally {
			busy = '';
		}
	}

	function mailboxFlagged(m: OutboundMailboxHealth): boolean {
		return m.status === 'paused' || m.spam_blocks_7d > 0 || m.complaints_7d > 0;
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if overview}
	<div class="rounded-lg border border-border p-4 mb-6">
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
			<div class="flex flex-wrap items-center gap-2">
				{#if !overview.enabled}
					<Badge variant="default">Off</Badge>
					<span class="text-sm text-text-muted">Outbound is off for this brand. <a href="{base}/setup" class="text-primary">Set it up</a></span>
				{:else if overview.paused}
					<Badge variant="error">Paused</Badge>
					<span class="text-sm text-error">{overview.paused}</span>
				{:else}
					<Badge variant="success">On</Badge>
					<span class="text-sm text-text-muted">Sending within each mailbox's daily limit</span>
				{/if}
			</div>
			{#if overview.enabled}
				{#if overview.paused}
					<Button type="secondary" size="sm" onclick={() => pause('all', true)} disabled={busy !== ''}>
						<Play class="mr-2 h-4 w-4" />
						Resume all
					</Button>
				{:else}
					<Button type="secondary" size="sm" onclick={() => pause('all', false)} disabled={busy !== ''}>
						<Pause class="mr-2 h-4 w-4" />
						Pause all
					</Button>
				{/if}
			{/if}
		</div>
		{#if overview.next?.length}
			<div class="mt-4">
				<p class="text-sm font-medium text-text mb-1">Next steps</p>
				<ul class="list-disc pl-5 text-sm text-text-muted space-y-1">
					{#each overview.next as step}
						<li>{step}</li>
					{/each}
				</ul>
			</div>
		{/if}
	</div>

	<div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Sent today</p>
			<p class="text-2xl font-semibold text-text">{sentToday} <span class="text-sm font-normal text-text-muted">of {allowedToday}</span></p>
		</div>
		<div class="rounded-lg border border-border p-4">
			<p class="text-xs text-text-muted">Reply rate</p>
			<p class="text-2xl font-semibold text-text">{percent(overview.reply_rate)}</p>
		</div>
		<a href="{base}/drafts" class="rounded-lg border border-border p-4 hover:border-primary/50 transition-colors">
			<p class="text-xs text-text-muted">Drafts waiting</p>
			<p class="text-2xl font-semibold text-text">{overview.drafts_waiting}</p>
		</a>
		<a href="{base}/replies" class="rounded-lg border border-border p-4 hover:border-primary/50 transition-colors">
			<p class="text-xs text-text-muted">Replies waiting</p>
			<p class="text-2xl font-semibold text-text">{overview.replies_waiting}</p>
		</a>
	</div>

	<h2 class="text-lg font-semibold text-text mb-3">Mailboxes</h2>
	{#if explained}
		<Alert type="info" title={explained.email} dismissible onclose={() => (explained = null)} class="mb-4">
			<p class="whitespace-pre-line">{explained.answer}</p>
		</Alert>
	{/if}
	{#if overview.mailboxes.length === 0}
		<div class="mb-6">
			<EmptyState
				icon="mail"
				title="No mailboxes yet"
				description="Add a mailbox on the Setup tab, or let the AI add one over MCP."
			/>
		</div>
	{:else}
		<div class="data-table overflow-x-auto mb-6">
			<table class="w-full text-sm">
				<thead>
					<tr>
						<th class="text-left">Mailbox</th>
						<th class="text-left">Status</th>
						<th class="text-right">Today</th>
						<th class="text-left">Daily limit</th>
						<th class="text-right">Sent 7d</th>
						<th class="text-right">Bounces</th>
						<th class="text-right">Spam blocks</th>
						<th class="text-right">Complaints</th>
						<th class="text-right"></th>
					</tr>
				</thead>
				<tbody>
					{#each overview.mailboxes as m (m.id)}
						<tr class={mailboxFlagged(m) ? 'bg-error/5' : ''}>
							<td>
								<span class="font-medium text-text block">{m.email}</span>
								<span class="text-xs text-text-muted block">{m.from_name}, on {m.domain}</span>
								{#if m.last_error}
									<span class="text-xs text-error block mt-1">{m.last_error}</span>
								{/if}
							</td>
							<td>
								{#if m.status === 'paused'}
									<Badge variant="error" size="sm">Paused</Badge>
									{#if m.reason}
										<span class="text-xs text-error block mt-1">{m.reason}</span>
									{/if}
								{:else}
									<Badge variant="success" size="sm">Active</Badge>
								{/if}
							</td>
							<td class="text-right">{m.sent_today}{m.limited ? `/${m.allowed_today}` : ''}</td>
							<td class="text-xs text-text-muted">{mailboxLimit(m)}</td>
							<td class="text-right">{m.sent_7d}</td>
							<td class="text-right">{m.bounces_7d}</td>
							<td class="text-right" class:text-error={m.spam_blocks_7d > 0}>{m.spam_blocks_7d}</td>
							<td class="text-right" class:text-error={m.complaints_7d > 0}>{m.complaints_7d}</td>
							<td class="text-right whitespace-nowrap">
								<Button
									type="ghost"
									size="xs"
									onclick={() => pause('mailbox', m.status === 'paused', m.id)}
									disabled={busy !== ''}
								>
									{m.status === 'paused' ? 'Resume' : 'Pause'}
								</Button>
								<Button type="link" size="xs" onclick={() => explain(m)} disabled={busy !== ''}>
									Explain
								</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	{#if overview.segments.some((s) => s.results.campaigns > 0)}
		<h2 class="text-lg font-semibold text-text mb-3">By segment</h2>
		<div class="data-table overflow-x-auto mb-8">
			<table class="w-full text-sm">
				<thead>
					<tr>
						<th class="text-left">Segment</th>
						<th class="text-right">Prospects</th>
						<th class="text-right">Campaigns</th>
						<th class="text-right">Reached</th>
						<th class="text-right">Replied</th>
						<th class="text-right">Reply rate</th>
						<th class="text-right">Interested</th>
					</tr>
				</thead>
				<tbody>
					{#each overview.segments as sg (sg.id)}
						<tr>
							<td><a href="{base}/targeting" class="link link-primary">{sg.name}</a></td>
							<td class="text-right">{sg.prospects.prospect || 0}</td>
							<td class="text-right">{sg.results.campaigns}</td>
							<td class="text-right">{sg.results.reached}</td>
							<td class="text-right">{sg.results.replied}</td>
							<td class="text-right">{percent(sg.results.reply_rate)}</td>
							<td class="text-right">{sg.results.interested}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<h2 class="text-lg font-semibold text-text mb-3">Campaigns</h2>
	{#if overview.campaigns.length === 0}
		<EmptyState icon={Send} title="No campaigns yet" description="Start one on the Campaigns tab, or let the AI start one over MCP." />
	{:else}
		<CampaignTable campaigns={overview.campaigns} />
	{/if}
{/if}
