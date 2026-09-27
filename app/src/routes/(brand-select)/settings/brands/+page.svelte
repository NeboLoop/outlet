<script lang="ts">
	import { listOrganizations, updateOrganization, getDashboardStats, type OrgInfo } from '$lib/api';
	import { Button, Card, Input, Badge, Modal, Alert, LoadingSpinner, SaveButton, Toggle } from '$lib/components/ui';
	import { Building2 } from '@lucide/svelte';

	interface OrgWithStats extends OrgInfo {
		current_contacts?: number;
		loading_stats?: boolean;
	}

	let loading = $state(true);
	let organizations = $state<OrgWithStats[]>([]);
	let error = $state('');

	// Edit modal state
	let showEditModal = $state(false);
	let editing = $state(false);
	let editSaved = $state(false);
	let editingOrg = $state<OrgWithStats | null>(null);
	let editMaxContacts = $state(0);
	let editUnlimited = $state(false);

	$effect(() => {
		loadOrganizations();
	});

	async function loadOrganizations() {
		loading = true;
		error = '';

		try {
			const response = await listOrganizations();
			organizations = (response.organizations || []).map(org => ({
				...org,
				loading_stats: true
			}));

			// Load stats for each organization in parallel
			await Promise.all(
				organizations.map(async (org, index) => {
					try {
						const stats = await getDashboardStats({}, org.id);
						organizations[index] = {
							...organizations[index],
							current_contacts: stats.total_subscribers,
							loading_stats: false
						};
					} catch {
						organizations[index] = {
							...organizations[index],
							loading_stats: false
						};
					}
				})
			);
		} catch (err) {
			console.error('Failed to load organizations:', err);
			error = 'Failed to load brands';
		} finally {
			loading = false;
		}
	}

	function openEditModal(org: OrgWithStats) {
		editingOrg = org;
		editMaxContacts = org.max_contacts || 0;
		editUnlimited = org.max_contacts === 0;
		showEditModal = true;
	}

	function closeEditModal() {
		showEditModal = false;
		editingOrg = null;
	}

	async function submitEdit() {
		if (!editingOrg) return;

		editing = true;
		editSaved = false;
		error = '';

		try {
			const maxContacts = editUnlimited ? 0 : editMaxContacts;
			await updateOrganization({}, { max_contacts: maxContacts }, editingOrg.id);
			editSaved = true;
			setTimeout(() => {
				editSaved = false;
				closeEditModal();
			}, 1500);
			await loadOrganizations();
		} catch (err: any) {
			console.error('Failed to update brand:', err);
			error = err.message || 'Failed to update brand';
		} finally {
			editing = false;
		}
	}

	function formatLimit(maxContacts: number): string {
		if (maxContacts === 0) return 'Unlimited';
		return maxContacts.toLocaleString();
	}

	function formatDate(dateString: string): string {
		if (!dateString) return 'N/A';
		return new Date(dateString).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function getUsageVariant(current: number | undefined, max: number): 'default' | 'success' | 'warning' | 'error' {
		if (max === 0) return 'default'; // Unlimited
		if (current === undefined) return 'default';
		const percent = (current / max) * 100;
		if (percent >= 90) return 'error';
		if (percent >= 75) return 'warning';
		return 'success';
	}
</script>

<svelte:head>
	<title>Brands - Outlet</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex justify-between items-center">
		<p class="text-sm text-text-secondary">Manage limits and quotas for all brands</p>
	</div>

	{#if error}
		<Alert type="error" title="Error">
			<p>{error}</p>
		</Alert>
	{/if}

	{#if loading}
		<LoadingSpinner size="large" />
	{:else if organizations.length === 0}
		<Card>
			<div class="text-center py-12">
				<Building2 class="mx-auto h-12 w-12 text-text-muted" />
				<h3 class="mt-2 text-sm font-medium text-text">No brands</h3>
				<p class="mt-1 text-sm text-text-muted">No brands found. Create a brand to get started.</p>
			</div>
		</Card>
	{:else}
		<Card>
			<div class="divide-y divide-border">
				{#each organizations as org}
					<button
						type="button"
						onclick={() => openEditModal(org)}
						class="w-full flex items-center justify-between py-4 first:pt-0 last:pb-0 hover:bg-surface-secondary transition-colors cursor-pointer text-left -mx-4 px-4 first:-mt-4 first:pt-4 last:-mb-4 last:pb-4 rounded-lg"
					>
						<div class="flex items-center gap-3">
							<div class="h-10 w-10 rounded-full bg-primary flex items-center justify-center">
								<span class="text-sm font-medium text-white">
									{org.name[0].toUpperCase()}
								</span>
							</div>
							<div>
								<div class="flex items-center gap-2">
									<span class="font-medium text-text">{org.name}</span>
									<Badge variant="default" size="sm">{org.slug}</Badge>
								</div>
								<p class="text-sm text-text-muted">Created {formatDate(org.created_at)}</p>
							</div>
						</div>
						<div class="text-right">
							<div class="flex items-center gap-2">
								{#if org.loading_stats}
									<span class="text-sm text-text-muted">Loading...</span>
								{:else}
									<span class="text-sm font-medium text-text">
										{org.current_contacts?.toLocaleString() ?? '—'}
									</span>
									<span class="text-sm text-text-muted">/</span>
									<Badge variant={getUsageVariant(org.current_contacts, org.max_contacts)} size="sm">
										{formatLimit(org.max_contacts)}
									</Badge>
								{/if}
							</div>
							<p class="text-xs text-text-muted">Contacts</p>
						</div>
					</button>
				{/each}
			</div>
		</Card>

		<Card>
			<div class="p-4 bg-surface-secondary rounded-lg">
				<h3 class="text-sm font-medium text-text mb-2">About Contact Limits</h3>
				<p class="text-sm text-text-muted">
					Contact limits control how many contacts each brand can store.
					Setting a limit to 0 or leaving it empty means unlimited contacts.
					When a brand reaches its limit, new contact imports will be blocked.
				</p>
			</div>
		</Card>
	{/if}
</div>

<!-- Edit Limit Modal -->
<Modal bind:show={showEditModal} title="Edit Contact Limit">
	{#if editingOrg}
		<div class="space-y-4">
			<div>
				<p class="text-sm text-text-muted mb-4">
					Configure the maximum number of contacts for <strong>{editingOrg.name}</strong>.
				</p>
			</div>

			<div class="flex items-center justify-between p-3 bg-surface-secondary rounded-lg">
				<div>
					<span class="text-sm font-medium text-text">Unlimited contacts</span>
					<p class="text-xs text-text-muted">No limit on the number of contacts</p>
				</div>
				<Toggle bind:checked={editUnlimited} />
			</div>

			{#if !editUnlimited}
				<div>
					<label for="edit-max-contacts" class="form-label">Maximum Contacts</label>
					<Input
						id="edit-max-contacts"
						type="number"
						bind:value={editMaxContacts}
						min={1}
						placeholder="Enter max contacts (e.g., 10000)"
					/>
					<p class="mt-1 text-xs text-text-muted">
						Current usage: {editingOrg.current_contacts?.toLocaleString() ?? 'Unknown'} contacts
					</p>
				</div>
			{/if}
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end gap-3">
			<Button type="secondary" onclick={closeEditModal} disabled={editing}>
				Cancel
			</Button>
			<SaveButton
				label="Save Changes"
				saving={editing}
				saved={editSaved}
				onclick={submitEdit}
			/>
		</div>
	{/snippet}
</Modal>
