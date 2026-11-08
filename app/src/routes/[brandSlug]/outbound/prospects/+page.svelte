<script lang="ts">
	import { formatDate } from '$lib/dates';
	import {
		listOutboundProspects,
		addOutboundProspects,
		type OutboundProspectInfo,
		type OutboundProspectInput,
		type OutboundProspectResult
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState, Badge, Input, Textarea, SlideForm } from '$lib/components/ui';
	import { Plus, Users } from '@lucide/svelte';
	import { errorMessage } from '../shared';

	let loading = $state(true);
	let error = $state('');
	let prospects = $state<OutboundProspectInfo[]>([]);

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

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const res = await listOutboundProspects({ limit: 500 });
			prospects = res.prospects || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load prospects');
		} finally {
			loading = false;
		}
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
			await load();
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

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if prospects.length === 0}
	<EmptyState icon={Users} title="No prospects yet" description="Add them here, or let the AI find and add them over MCP." />
{:else}
	<div class="data-table overflow-x-auto">
		<table class="w-full text-sm">
			<thead>
				<tr>
					<th class="text-left">Email</th>
					<th class="text-left">Name</th>
					<th class="text-left hidden sm:table-cell">List</th>
					<th class="text-left">Status</th>
					<th class="text-left hidden sm:table-cell">Checked</th>
					<th class="text-left hidden md:table-cell">Source</th>
					<th class="text-left hidden md:table-cell">Added</th>
				</tr>
			</thead>
			<tbody>
				{#each prospects as p (`${p.id}:${p.list_id}`)}
					<tr>
						<td>{p.email}</td>
						<td>{p.name}</td>
						<td class="hidden sm:table-cell text-text-muted">{p.list_name}</td>
						<td>
							<Badge variant={p.status === 'held' ? 'warning' : 'default'} size="sm">{p.status}</Badge>
						</td>
						<td class="hidden sm:table-cell text-text-muted">{p.validation_status || 'Not yet'}</td>
						<td class="hidden md:table-cell text-text-muted">{p.email_source}</td>
						<td class="hidden md:table-cell text-text-muted whitespace-nowrap">{formatDate(p.created_at, 'datetime')}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}
