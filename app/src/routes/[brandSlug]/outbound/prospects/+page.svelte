<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { page } from '$app/stores';
	import {
		listOutboundProspects,
		addOutboundProspects,
		type OutboundListRef,
		type OutboundProspectInfo,
		type OutboundProspectInput,
		type OutboundProspectResult
	} from '$lib/api';
	import {
		Button,
		Alert,
		LoadingSpinner,
		EmptyState,
		Badge,
		Input,
		Textarea,
		SlideForm,
		Select,
		SearchInput,
		Checkbox
	} from '$lib/components/ui';
	import { Plus, Users } from '@lucide/svelte';
	import { errorMessage, pageRange } from '../shared';

	const PAGE_SIZE = 50;
	const STATUSES = [
		{ value: '', label: 'Any status' },
		{ value: 'prospect', label: 'Prospect' },
		{ value: 'held', label: 'Held' }
	];

	let base = $derived(`/${$page.params.brandSlug}/outbound`);

	let loading = $state(true);
	let error = $state('');
	let prospects = $state<OutboundProspectInfo[]>([]);
	let total = $state(0);
	let lists = $state<OutboundListRef[]>([]);

	let listId = $state('');
	let status = $state('');
	let search = $state('');
	// The search as applied, a moment after typing stops.
	let applied = $state('');
	let enrollable = $state(false);
	let offset = $state(0);
	let searchTimer: ReturnType<typeof setTimeout> | undefined;

	let listOptions = $derived([{ value: '', label: 'All lists' }, ...lists.map((l) => ({ value: l.id, label: l.name }))]);
	let filtered = $derived(listId !== '' || status !== '' || applied !== '' || enrollable);

	let showAdd = $state(false);
	let adding = $state(false);
	let lines = $state('');
	let listName = $state('');
	let lineErrors = $state<string[]>([]);
	let results = $state<OutboundProspectResult[]>([]);

	const resultVariant: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
		added: 'success',
		held: 'warning',
		duplicate: 'default',
		suppressed: 'default',
		refused: 'error'
	};

	// A new filter starts again from the first page.
	let lastFilter = '';
	$effect(() => {
		const filter = `${listId}|${status}|${applied}|${enrollable}`;
		if (filter !== lastFilter) {
			lastFilter = filter;
			if (offset !== 0) {
				offset = 0;
				return;
			}
		}
		load(listId, status, applied, enrollable, offset);
	});

	async function load(list: string, withStatus: string, query: string, onlyFree: boolean, from: number) {
		loading = true;
		error = '';
		try {
			const res = await listOutboundProspects({
				list_id: list || undefined,
				status: withStatus || undefined,
				search: query || undefined,
				enrollable: onlyFree || undefined,
				limit: PAGE_SIZE,
				offset: from
			});
			prospects = res.prospects || [];
			total = res.total;
			lists = res.lists || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load prospects');
		} finally {
			loading = false;
		}
	}

	function searchSoon() {
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => (applied = search.trim()), 300);
	}

	function parse(): { inputs: OutboundProspectInput[]; problems: string[] } {
		const inputs: OutboundProspectInput[] = [];
		const problems: string[] = [];
		lines
			.split('\n')
			.map((l) => l.trim())
			.filter((l) => l !== '')
			.forEach((line, i) => {
				const parts = line.split(',').map((p) => p.trim());
				if (parts.length < 3 || !parts[0] || !parts[parts.length - 1]) {
					problems.push(`Line ${i + 1}: write it as email, name, source`);
					return;
				}
				inputs.push({
					email: parts[0],
					name: parts.slice(1, -1).join(', ') || undefined,
					email_source: parts[parts.length - 1]
				});
			});
		return { inputs, problems };
	}

	async function submit() {
		const { inputs, problems } = parse();
		lineErrors = problems;
		results = [];
		if (problems.length || inputs.length === 0) return;
		adding = true;
		error = '';
		try {
			const res = await addOutboundProspects({ list: listName.trim() || undefined, prospects: inputs });
			results = res.prospects || [];
			if (results.every((r) => r.status !== 'refused')) lines = '';
			await load(listId, status, applied, enrollable, offset);
		} catch (err) {
			error = errorMessage(err, 'Could not add prospects');
		} finally {
			adding = false;
		}
	}
</script>

<div class="flex items-center justify-between gap-4 mb-4">
	<h2 class="text-lg font-semibold text-text">Prospects</h2>
	<Button type="primary" onclick={() => (showAdd = true)} disabled={showAdd}>
		<Plus class="mr-2 h-4 w-4" />
		Add prospects
	</Button>
</div>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

<SlideForm bind:show={showAdd} title="Add prospects">
	<div class="space-y-4">
		<div>
			<label for="pr-lines" class="form-label">One per line: email, name, source</label>
			<Textarea
				id="pr-lines"
				bind:value={lines}
				rows={6}
				placeholder={'jane@example.com, Jane Doe, YouTube about page\nsam@example.com, Sam Lee, channel website'}
			/>
			<p class="form-hint">The source says where you found the address. It is required for every prospect.</p>
		</div>
		<div>
			<label for="pr-list" class="form-label">List (optional)</label>
			<Input id="pr-list" bind:value={listName} placeholder="Outreach" />
		</div>
		{#if lineErrors.length}
			<Alert type="error">
				<ul class="list-disc pl-5">
					{#each lineErrors as problem}
						<li>{problem}</li>
					{/each}
				</ul>
			</Alert>
		{/if}
		{#if results.length}
			<div class="data-table overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr>
							<th class="text-left">Email</th>
							<th class="text-left">Result</th>
							<th class="text-left">Why</th>
						</tr>
					</thead>
					<tbody>
						{#each results as r}
							<tr>
								<td>{r.email}</td>
								<td><Badge variant={resultVariant[r.status] || 'default'} size="sm">{r.status}</Badge></td>
								<td class="text-text-muted">{r.reason || ''}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	{#snippet footer()}
		<Button type="secondary" onclick={() => (showAdd = false)} disabled={adding}>Close</Button>
		<Button type="primary" onclick={submit} disabled={adding || !lines.trim()}>
			{adding ? 'Adding...' : 'Add'}
		</Button>
	{/snippet}
</SlideForm>

<div class="flex flex-col lg:flex-row lg:items-center gap-3 mb-4">
	<SearchInput bind:value={search} placeholder="Search email or name" size="sm" class="lg:w-72" onsearch={searchSoon} onclear={searchSoon} />
	<div class="flex flex-wrap items-center gap-3">
		<Select size="sm" bind:value={listId} options={listOptions} class="w-44" />
		<Select size="sm" bind:value={status} options={STATUSES} class="w-36" />
		<Checkbox bind:checked={enrollable} label="Free to enroll" size="sm" />
	</div>
</div>

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if prospects.length === 0}
	{#if filtered}
		<p class="text-sm text-text-muted py-6">No prospects match.</p>
	{:else}
		<EmptyState icon={Users} title="No prospects yet" description="Add them here, or let the AI find and add them over MCP." />
	{/if}
{:else}
	<div class="data-table overflow-x-auto">
		<table class="w-full text-sm">
			<thead>
				<tr>
					<th class="text-left">Email</th>
					<th class="text-left">Name</th>
					<th class="text-left">Lists</th>
					<th class="text-left">Campaign</th>
					<th class="text-left hidden sm:table-cell">Checked</th>
					<th class="text-left hidden md:table-cell">Source</th>
					<th class="text-left hidden md:table-cell">Added</th>
				</tr>
			</thead>
			<tbody>
				{#each prospects as p (p.id)}
					<tr>
						<td>
							{p.email}
							{#if p.suppressed}
								<Badge variant="error" size="sm">Suppressed</Badge>
							{/if}
						</td>
						<td>{p.name}</td>
						<td>
							<div class="flex flex-wrap gap-1">
								{#each p.lists as l (l.id)}
									<Badge variant={l.status === 'held' ? 'warning' : 'default'} size="sm">{l.name}: {l.status}</Badge>
								{/each}
							</div>
						</td>
						<td>
							{#each p.campaigns as c (c.id)}
								<a href="{base}/campaigns/{c.id}" class="link link-primary block">{c.name}</a>
							{:else}
								<span class="text-text-muted">{p.enrollable ? 'Free to enroll' : 'None'}</span>
							{/each}
						</td>
						<td class="hidden sm:table-cell text-text-muted">{p.validation_status || 'Not yet'}</td>
						<td class="hidden md:table-cell text-text-muted">{p.email_source}</td>
						<td class="hidden md:table-cell text-text-muted whitespace-nowrap">{formatDate(p.created_at, 'datetime')}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<div class="flex items-center justify-between gap-3 mt-3 text-sm text-text-muted">
		<span>{pageRange(offset, prospects.length, total)}</span>
		<div class="flex gap-2">
			<Button type="secondary" size="sm" onclick={() => (offset = Math.max(0, offset - PAGE_SIZE))} disabled={offset === 0}>
				Previous
			</Button>
			<Button type="secondary" size="sm" onclick={() => (offset += PAGE_SIZE)} disabled={offset + prospects.length >= total}>
				Next
			</Button>
		</div>
	</div>
{/if}
