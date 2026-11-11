<script lang="ts">
	import {
		getOutboundProfile,
		updateOutboundProfile,
		listOutboundSegments,
		createOutboundSegment,
		updateOutboundSegment,
		type OutboundProfile,
		type OutboundSegment,
		type OutboundSegmentFilters
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState, Input, Textarea, Badge } from '$lib/components/ui';
	import { Target, Plus } from '@lucide/svelte';
	import { errorMessage, joinList, percent, splitList } from '../shared';

	// Who outreach is for: what the business sells and to whom (the brand
	// profile), and its segments. An AI reading the website fills these in
	// over MCP; this is where a person reads and corrects them.

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');

	let profile = $state<OutboundProfile>({});
	let savingProfile = $state(false);

	let segments = $state<OutboundSegment[]>([]);
	// The segment being edited: its id, 'new' for a new one, or ''.
	let editing = $state('');
	let saving = $state(false);
	let form = $state({ name: '', description: '', countries: '', regions: '', industries: '', sizes: '', titles: '' });

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const [p, s] = await Promise.all([getOutboundProfile(), listOutboundSegments()]);
			profile = p;
			segments = s.segments || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load targeting');
		} finally {
			loading = false;
		}
	}

	async function saveProfile() {
		savingProfile = true;
		error = '';
		notice = '';
		try {
			profile = await updateOutboundProfile({
				website: profile.website?.trim(),
				sells: profile.sells?.trim(),
				audience: profile.audience?.trim(),
				proof: profile.proof?.trim(),
				voice: profile.voice?.trim()
			});
			notice = 'Profile saved.';
		} catch (err) {
			error = errorMessage(err, 'Could not save the profile');
		} finally {
			savingProfile = false;
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
		} catch (err) {
			error = errorMessage(err, 'Could not save the segment');
		} finally {
			saving = false;
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
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else}
	<div class="space-y-8">
		<section>
			<h2 class="text-lg font-semibold text-text mb-1">Brand profile</h2>
			<p class="form-hint mb-3">
				What the business sells, to whom, and its proof. Outreach is written from it: an AI reading your website fills it
				in, and you correct it here.
			</p>
			<div class="space-y-4">
				<div>
					<label for="tp-website" class="form-label">Website</label>
					<Input id="tp-website" bind:value={profile.website} placeholder="https://example.com" />
				</div>
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label for="tp-sells" class="form-label">What it sells</label>
						<Textarea id="tp-sells" bind:value={profile.sells} rows={3} />
					</div>
					<div>
						<label for="tp-audience" class="form-label">Who buys it</label>
						<Textarea id="tp-audience" bind:value={profile.audience} rows={3} />
					</div>
					<div>
						<label for="tp-proof" class="form-label">Proof</label>
						<Textarea id="tp-proof" bind:value={profile.proof} rows={3} placeholder="Customers, results, numbers" />
					</div>
					<div>
						<label for="tp-voice" class="form-label">How its emails sound</label>
						<Textarea id="tp-voice" bind:value={profile.voice} rows={3} placeholder="Plain, friendly, no hype" />
					</div>
				</div>
				<div class="flex justify-end">
					<Button type="primary" onclick={saveProfile} disabled={savingProfile}>
						{savingProfile ? 'Saving...' : 'Save profile'}
					</Button>
				</div>
			</div>
		</section>

		<section>
			<div class="flex items-center justify-between gap-2 mb-1">
				<h2 class="text-lg font-semibold text-text">Segments</h2>
				{#if editing === ''}
					<Button type="secondary" size="sm" onclick={() => edit(null)}>
						<Plus class="mr-2 h-4 w-4" />
						New segment
					</Button>
				{/if}
			</div>
			<p class="form-hint mb-3">
				Who a campaign targets. Prospects are added into a segment, a campaign runs on one, and results break down by it.
				Filters guide whoever finds the prospects; anyone outside them is noted when added, never refused.
			</p>

			{#if editing !== ''}
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
								<Button type="ghost" size="sm" onclick={() => edit(sg)} disabled={editing !== ''}>Edit</Button>
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
		</section>
	</div>
{/if}
