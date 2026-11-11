<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		createOutboundCampaign,
		listOutboundMaterials,
		listOutboundProspects,
		type OutboundCampaignRequest,
		type OutboundCampaignResponse,
		type OutboundListRef,
		type OutboundMaterial,
		type OutboundProspectInfo
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Input, Textarea, Toggle, Select, Checkbox, EmptyState } from '$lib/components/ui';
	import { ArrowLeft, Plus, Trash2, Users } from '@lucide/svelte';
	import { errorMessage, materialKindLabel } from '../../shared';

	const MAX_STEPS = 5;
	const MAX_PROSPECTS = 100;

	let base = $derived(`/${$page.params.brandSlug}/outbound`);

	let name = $state('');
	let steps = $state<{ subject: string; body: string; delay_days: number }[]>([{ subject: '', body: '', delay_days: 0 }]);
	let approvalMode = $state('every');
	let trackOpens = $state(false);
	let trackClicks = $state(false);
	let windowStart = $state(8);
	let windowEnd = $state(17);
	// One key per visit to this form, so a double click never makes two campaigns.
	const idempotencyKey = crypto.randomUUID();

	let loadingProspects = $state(true);
	let lists = $state<OutboundListRef[]>([]);
	let listId = $state('');
	// Only people the chosen list's campaign can take now.
	let prospects = $state<OutboundProspectInfo[]>([]);
	let totalFree = $state(0);
	let selected = $state<string[]>([]);
	// Enroll everyone in the segment free to enroll (up to 100), or pick.
	let wholeSegment = $state(false);
	let materials = $state<OutboundMaterial[]>([]);
	let pinned = $state<string[]>([]);

	let listName = $derived(lists.find((l) => l.id === listId)?.name || '');

	let error = $state('');
	let working = $state(false);
	let preview = $state<OutboundCampaignResponse | null>(null);

	let emailById = $derived(new Map(prospects.map((p) => [p.id, p.email])));
	let allSelected = $derived(
		prospects.length > 0 && selected.length === Math.min(prospects.length, MAX_PROSPECTS)
	);
	let canSubmit = $derived(
		name.trim() !== '' &&
			listName !== '' &&
			steps[0].subject.trim() !== '' &&
			steps.every((s) => s.body.trim() !== '') &&
			(wholeSegment ? totalFree > 0 : selected.length > 0) &&
			!working
	);

	$effect(() => {
		loadProspects(listId);
	});

	$effect(() => {
		loadMaterials(listName);
	});

	// The segment's own materials and the brand-wide ones.
	async function loadMaterials(segment: string) {
		if (!segment) return;
		try {
			const res = await listOutboundMaterials({ segment });
			materials = res.materials || [];
			pinned = pinned.filter((id) => materials.some((m) => m.id === id));
		} catch (err) {
			error = errorMessage(err, 'Could not load materials');
		}
	}

	function togglePinned(id: string, on: boolean) {
		pinned = on ? [...pinned.filter((p) => p !== id), id] : pinned.filter((p) => p !== id);
	}

	async function loadProspects(id: string) {
		loadingProspects = true;
		try {
			const res = await listOutboundProspects({ list_id: id || undefined, enrollable: true, limit: 500 });
			lists = res.lists || [];
			if (!id && lists.length) {
				// Start on the default outreach list; choosing it loads its people.
				listId = (lists.find((l) => l.name === 'Outreach') || lists[0]).id;
				return;
			}
			prospects = res.prospects || [];
			totalFree = res.total;
			selected = selected.filter((s) => prospects.some((p) => p.id === s));
		} catch (err) {
			error = errorMessage(err, 'Could not load prospects');
		} finally {
			loadingProspects = false;
		}
	}

	function addStep() {
		if (steps.length < MAX_STEPS) steps.push({ subject: '', body: '', delay_days: 3 });
	}

	function removeStep(i: number) {
		steps.splice(i, 1);
	}

	function toggleProspect(id: string, on: boolean) {
		if (on && !selected.includes(id) && selected.length < MAX_PROSPECTS) {
			selected = [...selected, id];
		} else if (!on) {
			selected = selected.filter((s) => s !== id);
		}
	}

	function toggleAll(on: boolean) {
		selected = on ? prospects.slice(0, MAX_PROSPECTS).map((p) => p.id) : [];
	}

	function request(dryRun: boolean): OutboundCampaignRequest {
		return {
			name: name.trim(),
			list: listName,
			steps: steps.map((s, i) => ({
				subject: s.subject.trim() || undefined,
				body: s.body,
				delay_days: i === 0 ? undefined : Number(s.delay_days)
			})),
			prospect_ids: wholeSegment ? undefined : selected,
			material_ids: pinned.length ? pinned : undefined,
			approval_mode: approvalMode,
			track_opens: trackOpens,
			track_clicks: trackClicks,
			window_start_hour: Number(windowStart),
			window_end_hour: Number(windowEnd),
			idempotency_key: dryRun ? undefined : idempotencyKey,
			dry_run: dryRun
		};
	}

	async function runPreview() {
		working = true;
		error = '';
		try {
			preview = await createOutboundCampaign(request(true));
		} catch (err) {
			error = errorMessage(err, 'Could not preview the campaign');
		} finally {
			working = false;
		}
	}

	async function create() {
		working = true;
		error = '';
		try {
			const res = await createOutboundCampaign(request(false));
			if (res.campaign_id) {
				goto(`${base}/campaigns/${res.campaign_id}`);
			} else {
				error = res.next || 'The campaign was not created';
			}
		} catch (err) {
			error = errorMessage(err, 'Could not create the campaign');
		} finally {
			working = false;
		}
	}
</script>

<a href="{base}/campaigns" class="inline-flex items-center text-sm text-text-muted hover:text-text mb-4">
	<ArrowLeft class="mr-1 h-4 w-4" />
	Campaigns
</a>

<h2 class="text-xl font-semibold text-text mb-4">New campaign</h2>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

<div class="space-y-6">
	<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
		<div>
			<label for="oc-name" class="form-label">Name</label>
			<Input id="oc-name" bind:value={name} placeholder="Creator partners, spring" />
		</div>
		<div>
			<label for="oc-list" class="form-label">Segment</label>
			<Select id="oc-list" bind:value={listId} options={lists.map((l) => ({ value: l.id, label: l.name }))} />
		</div>
	</div>

	<div>
		<h3 class="text-base font-semibold text-text mb-1">Emails</h3>
		<p class="form-hint mb-3">
			Merge tags: <code>{'{{first_name}}'}</code>, <code>{'{{name}}'}</code>, <code>{'{{company}}'}</code>,
			<code>{'{{title}}'}</code>, <code>{'{{recent_video}}'}</code>,
			<code>{'{{personal_line}}'}</code>, <code>{'{{channel_url}}'}</code>, <code>{'{{sender_first}}'}</code>, or any
			custom field. Add a fallback with <code>{'{{first_name,fallback=there}}'}</code>. Vary wording with
			<code>{'{Hi|Hey}'}</code>. The first email may have at most one link.
		</p>
		<div class="space-y-4">
			{#each steps as step, i}
				<div class="rounded-lg border border-border p-4">
					<div class="flex items-center justify-between mb-3">
						<p class="text-sm font-medium text-text">Step {i + 1}</p>
						{#if i > 0}
							<Button type="ghost" size="xs" onclick={() => removeStep(i)}>
								<Trash2 class="h-4 w-4" />
							</Button>
						{/if}
					</div>
					<div class="grid grid-cols-1 sm:grid-cols-4 gap-4">
						<div class={i === 0 ? 'sm:col-span-4' : 'sm:col-span-3'}>
							<label for="oc-subject-{i}" class="form-label">Subject</label>
							<Input
								id="oc-subject-{i}"
								bind:value={step.subject}
								placeholder={i === 0 ? 'Loved your video, {{first_name}}' : 'Leave empty to reply in the same thread'}
							/>
						</div>
						{#if i > 0}
							<div>
								<label for="oc-delay-{i}" class="form-label">Days after previous</label>
								<Input id="oc-delay-{i}" type="number" min={1} bind:value={step.delay_days} />
							</div>
						{/if}
						<div class="sm:col-span-4">
							<label for="oc-body-{i}" class="form-label">Body</label>
							<Textarea id="oc-body-{i}" bind:value={step.body} rows={6} />
						</div>
					</div>
				</div>
			{/each}
		</div>
		{#if steps.length < MAX_STEPS}
			<Button type="secondary" size="sm" class="mt-3" onclick={addStep}>
				<Plus class="mr-2 h-4 w-4" />
				Add follow-up
			</Button>
		{/if}
	</div>

	<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
		<div>
			<label for="oc-approval" class="form-label">Approval</label>
			<Select
				id="oc-approval"
				bind:value={approvalMode}
				options={[
					{ value: 'every', label: 'Approve every email' },
					{ value: 'first', label: 'Approve the first email only' },
					{ value: 'automatic', label: 'Send without approval' }
				]}
			/>
		</div>
		<div>
			<label for="oc-start" class="form-label">Send from (hour)</label>
			<Input id="oc-start" type="number" min={0} max={23} bind:value={windowStart} />
		</div>
		<div>
			<label for="oc-end" class="form-label">Send until (hour)</label>
			<Input id="oc-end" type="number" min={1} max={24} bind:value={windowEnd} />
		</div>
	</div>
	<p class="form-hint">The hours of the day emails may go out. Weekends are skipped.</p>

	<div class="flex flex-col sm:flex-row gap-4">
		<Toggle bind:checked={trackOpens} label="Track opens" />
		<Toggle bind:checked={trackClicks} label="Track clicks" />
	</div>

	{#if materials.length}
		<div>
			<h3 class="text-base font-semibold text-text mb-1">Written from</h3>
			<p class="form-hint mb-3">Pin the materials the emails draw on, so the campaign shows what to check claims against.</p>
			<div class="space-y-2">
				{#each materials as m (m.id)}
					<Checkbox
						checked={pinned.includes(m.id)}
						onchange={(on) => togglePinned(m.id, on)}
						label={`${materialKindLabel(m.kind)}: ${m.title}`}
						size="sm"
					/>
				{/each}
			</div>
		</div>
	{/if}

	<div>
		<h3 class="text-base font-semibold text-text mb-1">Prospects</h3>
		<div class="mb-3">
			<Toggle bind:checked={wholeSegment} label={`Everyone in ${listName || 'the segment'} free to enroll (up to ${MAX_PROSPECTS})`} />
		</div>
		<p class="form-hint mb-3">
			People on {listName || 'the list'} free to enroll: prospects in no other campaign and not suppressed. Pick up to
			{MAX_PROSPECTS}. {selected.length} selected.{#if totalFree > prospects.length}
				Showing the newest {prospects.length} of {totalFree}.{/if}
		</p>
		{#if wholeSegment}
			<p class="text-sm text-text">{Math.min(totalFree, MAX_PROSPECTS)} of {totalFree} free to enroll will be enrolled. Anyone the brand already knows is skipped, with the reason.</p>
		{:else if loadingProspects}
			<div class="flex justify-center py-6">
				<LoadingSpinner />
			</div>
		{:else if prospects.length === 0}
			<EmptyState
				icon={Users}
				title="No prospects free to enroll"
				description="Add prospects on the Prospects tab, or let the AI add them over MCP. People already in a campaign, held or suppressed aren't listed."
			/>
		{:else}
			<div class="data-table overflow-x-auto max-h-96 overflow-y-auto">
				<table class="w-full text-sm">
					<thead>
						<tr>
							<th class="text-left w-10">
								<Checkbox checked={allSelected} onchange={toggleAll} size="sm" />
							</th>
							<th class="text-left">Email</th>
							<th class="text-left">Name</th>
						</tr>
					</thead>
					<tbody>
						{#each prospects as p (p.id)}
							<tr>
								<td>
									<Checkbox
										checked={selected.includes(p.id)}
										disabled={!selected.includes(p.id) && selected.length >= MAX_PROSPECTS}
										onchange={(on) => toggleProspect(p.id, on)}
										size="sm"
									/>
								</td>
								<td>{p.email}</td>
								<td>{p.name}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	<div class="flex justify-end gap-3">
		<Button type="secondary" onclick={runPreview} disabled={!canSubmit}>
			{working ? 'Working...' : 'Preview'}
		</Button>
		<Button type="primary" onclick={create} disabled={!canSubmit}>
			{working ? 'Working...' : 'Create'}
		</Button>
	</div>

	{#if preview}
		<div>
			<h3 class="text-base font-semibold text-text mb-1">Preview</h3>
			{#if preview.next}
				<p class="form-hint mb-3">{preview.next}</p>
			{/if}
			{#if preview.skipped && Object.keys(preview.skipped).length}
				<Alert type="warning" title="Skipped" class="mb-3">
					<ul class="list-disc pl-5 text-sm">
						{#each Object.entries(preview.skipped) as [id, reason]}
							<li>{emailById.get(id) || id}: {reason}</li>
						{/each}
					</ul>
				</Alert>
			{/if}
			<div class="space-y-3">
				{#each preview.previews || [] as d}
					<div class="rounded-lg border border-border p-4">
						<p class="text-xs text-text-muted">
							From {d.from} to {d.to}{#if d.sends_after}, sends after {formatDate(d.sends_after, 'datetime')}{/if}
						</p>
						<p class="font-medium text-text mt-1">{d.subject}</p>
						<p class="text-sm text-text mt-2 whitespace-pre-line">{d.body}</p>
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>
