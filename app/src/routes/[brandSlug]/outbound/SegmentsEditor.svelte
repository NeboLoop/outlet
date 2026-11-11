<script lang="ts">
	import {
		listOutboundSegments,
		createOutboundSegment,
		updateOutboundSegment,
		deleteOutboundSegment,
		type OutboundSegment,
		type OutboundSegmentFilters
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState, Input, Textarea, Badge } from '$lib/components/ui';
	import { Target, Plus } from '@lucide/svelte';
	import { errorMessage, joinList, percent, splitList } from './shared';

	// The brand's segments: who a campaign targets. onchange hears each
	// segment saved or removed.

	let { onchange }: { onchange?: () => void } = $props();

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');

	let segments = $state<OutboundSegment[]>([]);
	// The segment being edited: its id, 'new' for a new one, or ''.
	let editing = $state('');
	let saving = $state(false);
	let removing = $state('');
	let form = $state({ name: '', description: '', countries: '', regions: '', industries: '', sizes: '', titles: '' });

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			segments = (await listOutboundSegments()).segments || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load the segments');
		} finally {
			loading = false;
		}
	}

	function edit(sg: OutboundSegment | null) {
		const f = sg?.filters || {};
		form = {
			name: sg?.name || '',
			description: sg?.description || '',
			countries: joinList(f.countries),
			regions: joinList(f.regions),
			industries: joinList(f.industries),
			sizes: joinList(f.company_sizes),
			titles: joinList(f.title_keywords)
		};
		editing = sg?.id || 'new';
	}

	async function saveSegment() {
		saving = true;
		error = '';
		notice = '';
		const filters: OutboundSegmentFilters = {
			countries: splitList(form.countries),
			regions: splitList(form.regions),
			industries: splitList(form.industries),
			company_sizes: splitList(form.sizes),
			title_keywords: splitList(form.titles),
			fields: segments.find((s) => s.id === editing)?.filters.fields
		};
		const req = { name: form.name.trim(), description: form.description.trim(), filters };
		try {
			if (editing === 'new') {
				await createOutboundSegment(req);
			} else {
				await updateOutboundSegment({}, req, editing);
			}
			notice = `Segment ${req.name} saved.`;
			editing = '';
			await load();
			onchange?.();
		} catch (err) {
			error = errorMessage(err, 'Could not save the segment');
		} finally {
			saving = false;
		}
	}

	async function remove(sg: OutboundSegment) {
		removing = sg.id;
		error = '';
		notice = '';
		try {
			await deleteOutboundSegment({}, sg.id);
			notice = `Segment ${sg.name} removed.`;
			await load();
			onchange?.();
		} catch (err) {
			error = errorMessage(err, 'Could not remove the segment');
		} finally {
			removing = '';
		}
	}

	function filterSummary(f: OutboundSegmentFilters): string[] {
		return [
			...(f.countries || []),
			...(f.regions || []),
			...(f.industries || []),
			...(f.company_sizes || []).map((s) => `${s} staff`),
			...(f.title_keywords || []).map((t) => `title: ${t}`),
			...Object.entries(f.fields || {}).map(([k, v]) => `${k}: ${v}`)
		];
	}

	// Only a segment nothing depends on can be removed: no one in it and no
	// campaign on it (the server also refuses one with materials of its own).
	function removable(sg: OutboundSegment): boolean {
		return sg.results.campaigns === 0 && Object.values(sg.prospects).every((n) => !n);
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

{#if loading}
	<div class="flex justify-center py-6">
		<LoadingSpinner />
	</div>
{:else}
	{#if editing === ''}
		<div class="flex justify-end mb-3">
			<Button type="secondary" size="sm" onclick={() => edit(null)}>
				<Plus class="mr-2 h-4 w-4" />
				New segment
			</Button>
		</div>
	{:else}
		<div class="rounded-lg border border-border p-4 mb-4 space-y-4">
			<div>
				<label for="sg-name" class="form-label">Name</label>
				<Input id="sg-name" bind:value={form.name} placeholder="Dental clinics in Utah" />
			</div>
			<div>
				<label for="sg-desc" class="form-label">Who it targets</label>
				<Textarea id="sg-desc" bind:value={form.description} rows={2} placeholder="Independent dental clinics in Utah with 10 to 50 staff" />
			</div>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div>
					<label for="sg-countries" class="form-label">Countries</label>
					<Input id="sg-countries" bind:value={form.countries} placeholder="US, CA" />
				</div>
				<div>
					<label for="sg-regions" class="form-label">Regions</label>
					<Input id="sg-regions" bind:value={form.regions} placeholder="UT, ID" />
				</div>
				<div>
					<label for="sg-industries" class="form-label">Industries</label>
					<Input id="sg-industries" bind:value={form.industries} placeholder="dental" />
				</div>
				<div>
					<label for="sg-sizes" class="form-label">Company sizes</label>
					<Input id="sg-sizes" bind:value={form.sizes} placeholder="11-50, 51-200" />
				</div>
				<div class="sm:col-span-2">
					<label for="sg-titles" class="form-label">Title keywords</label>
					<Input id="sg-titles" bind:value={form.titles} placeholder="owner, office manager" />
				</div>
			</div>
			<p class="form-hint">Separate values with commas.</p>
			<div class="flex justify-end gap-2">
				<Button type="secondary" onclick={() => (editing = '')} disabled={saving}>Cancel</Button>
				<Button type="primary" onclick={saveSegment} disabled={saving || form.name.trim() === ''}>
					{saving ? 'Saving...' : 'Save segment'}
				</Button>
			</div>
		</div>
	{/if}

	{#if segments.length === 0}
		<EmptyState icon={Target} title="No segments yet" description="Create one here, or let the AI propose them from your website over MCP." />
	{:else}
		<div class="space-y-3">
			{#each segments as sg (sg.id)}
				<div class="rounded-lg border border-border p-4">
					<div class="flex flex-wrap items-start justify-between gap-2">
						<div class="min-w-0">
							<p class="font-medium text-text">{sg.name}</p>
							{#if sg.description}
								<p class="text-sm text-text">{sg.description}</p>
							{/if}
							<div class="flex flex-wrap gap-1 mt-2">
								{#each filterSummary(sg.filters) as f}
									<Badge variant="default" size="sm">{f}</Badge>
								{/each}
							</div>
						</div>
						<div class="flex gap-1">
							<Button type="ghost" size="sm" onclick={() => edit(sg)} disabled={editing !== '' || removing !== ''}>Edit</Button>
							{#if removable(sg)}
								<Button type="ghost" size="sm" onclick={() => remove(sg)} disabled={editing !== '' || removing !== ''}>
									{removing === sg.id ? 'Removing...' : 'Remove'}
								</Button>
							{/if}
						</div>
					</div>
					<div class="grid grid-cols-2 sm:grid-cols-5 gap-2 mt-3 text-sm">
						<div><span class="text-text-muted">Prospects:</span> {sg.prospects.prospect || 0}</div>
						<div><span class="text-text-muted">Verifying:</span> {sg.prospects.verifying || 0}</div>
						<div><span class="text-text-muted">Campaigns:</span> {sg.results.campaigns}</div>
						<div><span class="text-text-muted">Reply rate:</span> {percent(sg.results.reply_rate)}</div>
						<div><span class="text-text-muted">Interested:</span> {sg.results.interested}</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
{/if}
