<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { page } from '$app/stores';
	import {
		exportOutboundLeads,
		listOutboundReplies,
		answerOutboundReply,
		classifyOutboundReply,
		markOutboundPartner,
		type OutboundReplyInfo
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState, Badge, DropdownMenu, Textarea, Toggle } from '$lib/components/ui';
	import { MessageSquare, ChevronDown, Download } from '@lucide/svelte';
	import { CLASSIFICATIONS, classificationLabel, errorMessage } from '../shared';

	let base = $derived(`/${$page.params.brandSlug}/outbound`);
	let handled = $state(false);
	// Hot leads are the replies sorted as interested.
	let hotOnly = $state(false);
	let exporting = $state(false);
	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let replies = $state<OutboundReplyInfo[]>([]);
	let busy = $state('');

	let answering = $state('');
	let answerBody = $state('');

	const badgeVariant: Record<string, 'success' | 'error' | 'warning' | 'info' | 'default'> = {
		interested: 'success',
		question: 'info',
		not_interested: 'default',
		out_of_office: 'default',
		wrong_person: 'warning',
		unsubscribe: 'warning',
		complaint: 'error'
	};

	$effect(() => {
		load(handled, hotOnly);
	});

	async function load(showHandled: boolean, hot: boolean) {
		loading = true;
		error = '';
		try {
			const res = await listOutboundReplies({ handled: showHandled, classification: hot ? 'interested' : undefined, limit: 100 });
			replies = res.replies || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load replies');
		} finally {
			loading = false;
		}
	}

	async function act(key: string, done: string, fallback: string, call: () => Promise<unknown>) {
		busy = key;
		error = '';
		notice = '';
		try {
			await call();
			notice = done;
			await load(handled, hotOnly);
		} catch (err) {
			error = errorMessage(err, fallback);
		} finally {
			busy = '';
		}
	}

	function classify(r: OutboundReplyInfo, classification: string) {
		if (!classification || classification === r.classification) return;
		act(`classify:${r.id}`, `Marked as ${classificationLabel(classification).toLowerCase()}`, 'Could not sort the reply', () =>
			classifyOutboundReply({}, { classification }, r.id)
		);
	}

	function sendAnswer(r: OutboundReplyInfo) {
		const body = answerBody.trim();
		if (!body) return;
		act(`answer:${r.id}`, `Answer sent to ${r.from_email}`, 'Could not send the answer', async () => {
			await answerOutboundReply({}, { body }, r.id);
			answering = '';
			answerBody = '';
		});
	}

	async function exportLeads() {
		exporting = true;
		error = '';
		try {
			await exportOutboundLeads({ classification: hotOnly ? 'interested' : undefined });
		} catch (err) {
			error = errorMessage(err, 'Could not export the leads');
		} finally {
			exporting = false;
		}
	}

	function markPartner(r: OutboundReplyInfo) {
		act(`partner:${r.id}`, `${r.from_email} is now a partner`, 'Could not mark as partner', () =>
			markOutboundPartner({}, r.contact_id)
		);
	}
</script>

<div class="flex flex-wrap items-center justify-between gap-2 mb-4">
	<div class="flex flex-wrap gap-2">
		<Button type={handled ? 'ghost' : 'primary'} size="sm" onclick={() => (handled = false)}>Needs attention</Button>
		<Button type={handled ? 'primary' : 'ghost'} size="sm" onclick={() => (handled = true)}>Handled</Button>
		<Toggle bind:checked={hotOnly} label="Hot leads only" />
	</div>
	<Button type="secondary" size="sm" onclick={exportLeads} disabled={exporting}>
		<Download class="mr-2 h-4 w-4" />
		{exporting ? 'Exporting...' : hotOnly ? 'Export hot leads' : 'Export leads'}
	</Button>
</div>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}
{#if notice}
	<Alert type="success" dismissible onclose={() => (notice = '')} class="mb-4">
		<p>{notice}</p>
	</Alert>
{/if}

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if replies.length === 0}
	<EmptyState
		icon={MessageSquare}
		title={hotOnly ? 'No hot leads here' : handled ? 'No handled replies' : 'Nothing needs attention'}
		description={handled ? 'Replies you or the AI have dealt with show up here.' : 'New replies to your outreach show up here.'}
	/>
{:else}
	<div class="space-y-3">
		{#each replies as r (r.id)}
			<div class="rounded-lg border border-border p-4">
				<div class="flex flex-wrap items-start justify-between gap-2">
					<div class="min-w-0">
						<p class="font-medium text-text">{r.from_email}</p>
						<p class="text-sm text-text">{r.subject}</p>
						<p class="text-xs text-text-muted">
							{formatDate(r.received_at, 'datetime')}{#if r.campaign_name}, to
								<a href="{base}/campaigns/{r.sequence_id}" class="link link-primary">{r.campaign_name}</a>{/if}{#if r.segment_name}
								in {r.segment_name}{/if}
						</p>
					</div>
					<Badge variant={badgeVariant[r.classification] || 'default'} size="sm">
						{classificationLabel(r.classification)}
					</Badge>
				</div>
				<p class="text-sm text-text mt-3 whitespace-pre-line">{r.body}</p>

				{#if answering === r.id}
					<div class="mt-3">
						<label for="answer-{r.id}" class="form-label">Your answer</label>
						<Textarea id="answer-{r.id}" bind:value={answerBody} rows={5} />
						<p class="form-hint">Sends in the same thread from the mailbox that got the reply.</p>
						<div class="flex justify-end gap-2 mt-2">
							<Button type="secondary" size="sm" onclick={() => (answering = '')} disabled={busy !== ''}>Cancel</Button>
							<Button type="primary" size="sm" onclick={() => sendAnswer(r)} disabled={busy !== '' || !answerBody.trim()}>
								Send
							</Button>
						</div>
					</div>
				{:else}
					<div class="flex flex-wrap items-center justify-end gap-2 mt-3">
						<DropdownMenu
							items={CLASSIFICATIONS.map((c) => ({
								label: c.label,
								disabled: busy !== '' || c.value === r.classification,
								onclick: () => classify(r, c.value)
							}))}
						>
							{#snippet trigger()}
								<span class="btn btn-ghost btn-sm">
									Sort as
									<ChevronDown class="ml-1 h-4 w-4" />
								</span>
							{/snippet}
						</DropdownMenu>
						{#if r.contact_id}
							<Button type="ghost" size="sm" onclick={() => markPartner(r)} disabled={busy !== ''}>Mark as partner</Button>
						{/if}
						<Button
							type="primary"
							size="sm"
							onclick={() => {
								answering = r.id;
								answerBody = '';
							}}
							disabled={busy !== ''}
						>
							Answer
						</Button>
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
