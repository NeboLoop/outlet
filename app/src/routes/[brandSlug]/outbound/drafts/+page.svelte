<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { page } from '$app/stores';
	import { listOutboundDrafts, reviewOutboundDrafts, editOutboundDraft, type OutboundDraft } from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState, Input, Textarea, Checkbox, Badge } from '$lib/components/ui';
	import { FileText } from '@lucide/svelte';
	import { authorLabel, authorVariant, errorMessage } from '../shared';

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let drafts = $state<OutboundDraft[]>([]);
	let selected = $state<string[]>([]);
	let refused = $state<{ [key: string]: string }>({});
	let busy = $state(false);

	let editing = $state('');
	let editSubject = $state('');
	let editBody = $state('');

	let base = $derived(`/${$page.params.brandSlug}/outbound`);
	let ids = $derived(drafts.map((d) => d.message_id));
	let allSelected = $derived(ids.length > 0 && selected.length === ids.length);

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const res = await listOutboundDrafts({ limit: 100 });
			drafts = res.drafts || [];
			selected = selected.filter((id) => drafts.some((d) => d.message_id === id));
		} catch (err) {
			error = errorMessage(err, 'Could not load drafts');
		} finally {
			loading = false;
		}
	}

	async function review(action: 'approve' | 'skip', target: string[]) {
		if (target.length === 0) return;
		busy = true;
		error = '';
		notice = '';
		try {
			const res = await reviewOutboundDrafts({ action, ids: target });
			refused = res.refused || {};
			notice = `${res.done} ${action === 'approve' ? 'approved' : 'skipped'}`;
			selected = selected.filter((id) => !target.includes(id));
			await load();
		} catch (err) {
			error = errorMessage(err, 'Could not review the drafts');
		} finally {
			busy = false;
		}
	}

	function startEdit(d: OutboundDraft) {
		editing = d.message_id;
		editSubject = d.subject;
		editBody = d.body;
	}

	async function saveEdit() {
		if (!editing) return;
		busy = true;
		error = '';
		try {
			await editOutboundDraft({}, { subject: editSubject, body: editBody }, editing);
			editing = '';
			await load();
		} catch (err) {
			error = errorMessage(err, 'Could not save the draft');
		} finally {
			busy = false;
		}
	}

	function toggle(id: string, on: boolean) {
		selected = on ? [...selected, id] : selected.filter((s) => s !== id);
	}
</script>

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
{#if Object.keys(refused).length}
	<Alert type="warning" title="Not done" dismissible onclose={() => (refused = {})} class="mb-4">
		<ul class="list-disc pl-5 text-sm">
			{#each Object.entries(refused) as [id, reason]}
				<li>{drafts.find((d) => d.message_id === id)?.to || id}: {reason}</li>
			{/each}
		</ul>
	</Alert>
{/if}

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if drafts.length === 0}
	<EmptyState
		icon={FileText}
		title="No drafts waiting"
		description="Emails that need approval show up here when a campaign writes them."
	/>
{:else}
	<div class="flex flex-wrap items-center justify-between gap-3 mb-4">
		<Checkbox
			checked={allSelected}
			onchange={(on) => (selected = on ? [...ids] : [])}
			label={allSelected ? 'Clear selection' : `Select all (${ids.length})`}
			size="sm"
		/>
		<div class="flex gap-2">
			<Button type="primary" size="sm" onclick={() => review('approve', selected)} disabled={busy || selected.length === 0}>
				Approve selected
			</Button>
			<Button type="secondary" size="sm" onclick={() => review('skip', selected)} disabled={busy || selected.length === 0}>
				Skip selected
			</Button>
		</div>
	</div>

	<div class="space-y-3">
		{#each drafts as d (d.message_id)}
			{@const id = d.message_id}
			<div class="rounded-lg border border-border p-4">
				<div class="flex items-start gap-3">
					<Checkbox checked={selected.includes(id)} onchange={(on) => toggle(id, on)} size="sm" />
					<div class="flex-1 min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<p class="text-xs text-text-muted">
								<a href="{base}/campaigns/{d.campaign_id}" class="link link-primary">{d.campaign_name}</a>, step {d.step}
							</p>
							{#if d.authored_by}
								<Badge variant={authorVariant(d.authored_by)} size="sm">{authorLabel(d.authored_by)}</Badge>
							{/if}
						</div>
						<p class="text-xs text-text-muted">
							To {d.to} from {d.from}{#if d.sends_after}, sends after {formatDate(d.sends_after, 'datetime')}{/if}
						</p>
						{#if editing === id}
							<div class="mt-2 space-y-3">
								<div>
									<label for="draft-subject-{id}" class="form-label">Subject</label>
									<Input id="draft-subject-{id}" bind:value={editSubject} />
								</div>
								<div>
									<label for="draft-body-{id}" class="form-label">Body</label>
									<Textarea id="draft-body-{id}" bind:value={editBody} rows={8} />
								</div>
								<div class="flex justify-end gap-2">
									<Button type="secondary" size="sm" onclick={() => (editing = '')} disabled={busy}>Cancel</Button>
									<Button type="primary" size="sm" onclick={saveEdit} disabled={busy}>Save</Button>
								</div>
							</div>
						{:else}
							<p class="font-medium text-text mt-1">{d.subject}</p>
							<p class="text-sm text-text mt-2 whitespace-pre-line">{d.body}</p>
						{/if}
					</div>
				</div>
				{#if editing !== id}
					<div class="flex justify-end gap-2 mt-3">
						<Button type="ghost" size="sm" onclick={() => startEdit(d)} disabled={busy}>Edit</Button>
						<Button type="secondary" size="sm" onclick={() => review('skip', [id])} disabled={busy}>Skip</Button>
						<Button type="primary" size="sm" onclick={() => review('approve', [id])} disabled={busy}>Approve</Button>
					</div>
				{/if}
			</div>
		{/each}
	</div>
{/if}
