<script lang="ts">
	import {
		listOutboundMaterials,
		createOutboundMaterial,
		updateOutboundMaterial,
		deleteOutboundMaterial,
		listOutboundSegments,
		type OutboundMaterial,
		type OutboundSegment
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState, Input, Textarea, Select, Badge } from '$lib/components/ui';
	import { BookOpen, Plus } from '@lucide/svelte';
	import { errorMessage, MATERIAL_KINDS, materialKindLabel } from '../shared';

	// What outreach can cite: case studies, proof points, offers, links and
	// assets, each with a short summary. Drafting hands them to the AI that
	// writes; a campaign pins the ones it used.

	const MAX_SUMMARY = 600;

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let materials = $state<OutboundMaterial[]>([]);
	let segments = $state<OutboundSegment[]>([]);
	let kind = $state('');

	// The material being edited: its id, 'new', or ''.
	let editing = $state('');
	let saving = $state(false);
	let busy = $state('');
	let form = $state({ kind: 'case_study', title: '', summary: '', details: '', url: '', segment: '' });

	$effect(() => {
		load(kind);
	});

	async function load(k: string) {
		loading = true;
		error = '';
		try {
			const [m, s] = await Promise.all([listOutboundMaterials({ kind: k || undefined }), listOutboundSegments()]);
			materials = m.materials || [];
			segments = s.segments || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load materials');
		} finally {
			loading = false;
		}
	}

	function edit(m: OutboundMaterial | null) {
		form = {
			kind: m?.kind || 'case_study',
			title: m?.title || '',
			summary: m?.summary || '',
			details: m?.details || '',
			url: m?.url || '',
			segment: m?.segment_id || ''
		};
		editing = m?.id || 'new';
	}

	async function save() {
		saving = true;
		error = '';
		notice = '';
		const req = {
			kind: form.kind,
			title: form.title.trim(),
			summary: form.summary.trim(),
			details: form.details.trim(),
			url: form.url.trim(),
			segment: form.segment
		};
		try {
			if (editing === 'new') {
				await createOutboundMaterial({ ...req, segment: req.segment || undefined });
			} else {
				// An empty segment makes it brand-wide again.
				await updateOutboundMaterial({}, { ...req, segment: req.segment || 'any' }, editing);
			}
			notice = `${req.title} saved.`;
			editing = '';
			await load(kind);
		} catch (err) {
			error = errorMessage(err, 'Could not save the material');
		} finally {
			saving = false;
		}
	}

	async function remove(m: OutboundMaterial) {
		busy = m.id;
		error = '';
		try {
			await deleteOutboundMaterial({}, m.id);
			notice = `${m.title} deleted.`;
			await load(kind);
		} catch (err) {
			error = errorMessage(err, 'Could not delete the material');
		} finally {
			busy = '';
		}
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

<div class="flex flex-wrap items-end justify-between gap-3 mb-4">
	<div class="w-full sm:w-56">
		<label for="mat-filter" class="form-label">Kind</label>
		<Select id="mat-filter" bind:value={kind} options={[{ value: '', label: 'All kinds' }, ...MATERIAL_KINDS]} />
	</div>
	{#if editing === ''}
		<Button type="secondary" size="sm" onclick={() => edit(null)}>
			<Plus class="mr-2 h-4 w-4" />
			New material
		</Button>
	{/if}
</div>

{#if editing !== ''}
	<div class="rounded-lg border border-border p-4 mb-4 space-y-4">
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
			<div>
				<label for="mat-kind" class="form-label">Kind</label>
				<Select id="mat-kind" bind:value={form.kind} options={MATERIAL_KINDS} />
			</div>
			<div>
				<label for="mat-segment" class="form-label">For segment</label>
				<Select id="mat-segment" bind:value={form.segment} options={[{ value: '', label: 'Any segment' }, ...segments.map((s) => ({ value: s.id, label: s.name }))]} />
			</div>
		</div>
		<div>
			<label for="mat-title" class="form-label">Title</label>
			<Input id="mat-title" bind:value={form.title} placeholder="Clinic Z doubled its bookings" />
		</div>
		<div>
			<label for="mat-summary" class="form-label">Summary</label>
			<Textarea id="mat-summary" bind:value={form.summary} rows={2} />
			<p class="form-hint">A sentence or two: what it shows. {form.summary.length} of {MAX_SUMMARY}.</p>
		</div>
		<div>
			<label for="mat-details" class="form-label">Details</label>
			<Textarea id="mat-details" bind:value={form.details} rows={4} />
		</div>
		<div>
			<label for="mat-url" class="form-label">Link</label>
			<Input id="mat-url" bind:value={form.url} placeholder="https://example.com/case-studies/clinic-z" />
		</div>
		<div class="flex justify-end gap-2">
			<Button type="secondary" onclick={() => (editing = '')} disabled={saving}>Cancel</Button>
			<Button type="primary" onclick={save} disabled={saving || !form.title.trim() || !form.summary.trim() || form.summary.length > MAX_SUMMARY}>
				{saving ? 'Saving...' : 'Save material'}
			</Button>
		</div>
	</div>
{/if}

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if materials.length === 0}
	<EmptyState icon={BookOpen} title="No materials yet" description="Add case studies, proof points and offers here, or let the AI add them from your website over MCP." />
{:else}
	<div class="space-y-3">
		{#each materials as m (m.id)}
			<div class="rounded-lg border border-border p-4">
				<div class="flex flex-wrap items-start justify-between gap-2">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<Badge variant="default" size="sm">{materialKindLabel(m.kind)}</Badge>
							<p class="font-medium text-text">{m.title}</p>
						</div>
						<p class="text-sm text-text mt-1">{m.summary}</p>
						<p class="text-xs text-text-muted mt-1">
							{m.segment ? `For ${m.segment}` : 'Any segment'}{#if m.url},
								<a href={m.url} class="link link-primary" target="_blank" rel="noopener noreferrer">{m.url}</a>{/if}
						</p>
					</div>
					<div class="flex gap-2">
						<Button type="ghost" size="sm" onclick={() => edit(m)} disabled={editing !== '' || busy !== ''}>Edit</Button>
						<Button type="ghost" size="sm" onclick={() => remove(m)} disabled={busy !== ''}>Delete</Button>
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}
