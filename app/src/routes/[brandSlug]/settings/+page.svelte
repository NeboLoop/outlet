<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import * as api from '$lib/api';
	import type { OrgInfo } from '$lib/api';
	import { Button, Card, Input, Modal, AlertDialog, Alert, LoadingSpinner, SaveButton } from '$lib/components/ui';
	import { Copy, Check, RefreshCw, Trash2 } from '@lucide/svelte';

	let loading = $state(true);
	let org = $state<OrgInfo | null>(null);
	let error = $state('');

	// Edit state
	let editName = $state('');
	let editAppUrl = $state('');
	let saving = $state(false);
	let saved = $state(false);

	// API key copy feedback
	let copied = $state(false);

	// Confirm dialogs
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);

	let showRegenerateConfirm = $state(false);
	let regenerating = $state(false);

	$effect(() => {
		loadOrg();
	});

	async function loadOrg() {
		loading = true;
		error = '';
		const brandSlug = $page.params.brandSlug;
		if (!brandSlug) return;

		try {
			org = await api.getOrganizationBySlug({}, brandSlug);
			editName = org.name;
			editAppUrl = org.app_url || '';
		} catch (err) {
			console.error('Failed to load organization:', err);
			error = 'Failed to load organization';
		} finally {
			loading = false;
		}
	}

	async function saveChanges() {
		if (!org || !editName.trim()) return;

		saving = true;
		saved = false;
		error = '';
		try {
			await api.updateOrganization({}, { name: editName.trim(), app_url: editAppUrl.trim() }, org.id);
			org.name = editName.trim();
			org.app_url = editAppUrl.trim();
			localStorage.setItem('currentOrgName', editName.trim());
			saved = true;
			setTimeout(() => { saved = false; }, 2000);
		} catch (err: any) {
			console.error('Failed to update organization:', err);
			error = err.message || 'Failed to update organization';
		} finally {
			saving = false;
		}
	}

	function copyApiKey() {
		if (!org) return;
		navigator.clipboard.writeText(org.api_key);
		copied = true;
		setTimeout(() => { copied = false; }, 2000);
	}

	async function executeRegenerate() {
		if (!org) return;

		regenerating = true;
		try {
			await api.regenerateOrgApiKey({}, org.id);
			showRegenerateConfirm = false;
			await loadOrg();
		} catch (err: any) {
			console.error('Failed to regenerate API key:', err);
			error = err.message || 'Failed to regenerate API key';
		} finally {
			regenerating = false;
		}
	}

	async function executeDelete() {
		if (!org) return;

		deleting = true;
		try {
			await api.deleteOrganization({}, org.id);
			localStorage.removeItem('currentOrgId');
			localStorage.removeItem('currentOrgName');
			localStorage.removeItem('currentOrgSlug');
			goto('/');
		} catch (err: any) {
			console.error('Failed to delete organization:', err);
			error = err.message || 'Failed to delete organization';
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>Brand Settings - {org?.name || 'Loading'}</title>
</svelte:head>

{#if error}
	<div class="mb-6">
		<Alert type="error" title="Error">
			<p>{error}</p>
		</Alert>
	</div>
{/if}

{#if loading}
	<LoadingSpinner size="large" />
{:else if org}
	<div class="space-y-6">
		<!-- General Settings -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-4">General</h2>
					<div class="space-y-4">
						<div>
							<label for="org-name" class="form-label">Brand Name</label>
							<Input
								id="org-name"
								type="text"
								bind:value={editName}
							/>
						</div>
						<div>
							<label class="form-label">Slug</label>
							<p class="text-sm text-text-muted bg-bg-secondary px-3 py-2 rounded-md">{org.slug}</p>
							<p class="mt-1 text-xs text-text-muted">The slug cannot be changed after creation</p>
						</div>
						<div>
							<label for="app-url" class="form-label">App URL</label>
							<Input
								id="app-url"
								type="text"
								bind:value={editAppUrl}
								placeholder="https://yourapp.com"
							/>
							<p class="mt-1 text-xs text-text-muted">Base URL of your application for confirmation links</p>
						</div>
						<div class="pt-2">
							<SaveButton
								label="Save Changes"
								{saving}
								{saved}
								onclick={saveChanges}
								disabled={editName === org.name && editAppUrl === (org.app_url || '')}
							/>
						</div>
					</div>
				</Card>

				<!-- API Key -->
				<Card>
					<h2 class="text-lg font-medium text-text mb-4">API Key</h2>
					<p class="text-sm text-text-muted mb-4">Use this key to authenticate API requests for this brand.</p>
					<div class="flex items-center gap-2 bg-bg-secondary px-3 py-2 rounded-md">
						<code class="flex-1 text-sm font-mono text-text break-all">{org.api_key}</code>
						<Button
							type="ghost"
							size="icon"
							onclick={copyApiKey}
							class="flex-shrink-0 p-1.5 text-text-muted hover:text-text rounded transition-colors"
						>
							{#if copied}
								<Check class="h-4 w-4 text-green-500" />
							{:else}
								<Copy class="h-4 w-4" />
							{/if}
						</Button>
					</div>
					<div class="mt-4">
						<Button
							type="secondary"
							onclick={() => showRegenerateConfirm = true}
						>
							<RefreshCw class="mr-2 h-4 w-4" />
							Regenerate Key
						</Button>
					</div>
				</Card>

				<!-- Danger Zone -->
				<Card>
					<h2 class="text-lg font-medium text-red-600 mb-4">Danger Zone</h2>
					<p class="text-sm text-text-muted mb-4">
						Permanently delete this brand and all of its data including contacts, lists, and campaigns.
						This action cannot be undone.
					</p>
					<Button
						type="danger"
						onclick={() => showDeleteConfirm = true}
					>
						<Trash2 class="mr-2 h-4 w-4" />
						Delete Brand
					</Button>
		</Card>
	</div>
{/if}

<!-- Regenerate API Key Confirmation -->
<AlertDialog
	bind:open={showRegenerateConfirm}
	title="Regenerate API Key"
	description="Are you sure you want to regenerate the API key? The current key will immediately stop working. Any applications using the old key will need to be updated."
	actionLabel={regenerating ? 'Regenerating...' : 'Regenerate'}
	actionType="danger"
	onAction={executeRegenerate}
	onCancel={() => showRegenerateConfirm = false}
/>

<!-- Delete Confirmation -->
<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Brand"
	description={`Are you sure you want to delete ${org?.name}? This will permanently delete all associated data including contacts, lists, and campaigns. This action cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => showDeleteConfirm = false}
/>
