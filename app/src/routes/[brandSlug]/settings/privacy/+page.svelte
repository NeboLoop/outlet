<script lang="ts">
	import { formatDate } from '$lib/dates';
	import * as api from '$lib/api';
	import type { GDPRConsentInfo, GDPRExportResponse, GDPRDeleteResponse, GDPRProviderResult } from '$lib/api';
	import { Button, Card, Input, Alert, LoadingSpinner, Badge, Toggle, AlertDialog } from '$lib/components/ui';
	import { Search, Download, Trash2, Shield, UserX, FileText } from '@lucide/svelte';

	let searchEmail = $state('');
	let searching = $state(false);
	let contactId = $state('');
	let consentInfo = $state<GDPRConsentInfo | null>(null);
	let error = $state('');
	let success = $state('');

	// Export state
	let exporting = $state(false);
	let exportResult = $state<GDPRExportResponse | null>(null);

	// Delete state
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);
	let deleteAlsoEmail = $state(false);
	// What each email provider did, when they were asked to forget the address
	let providerResults = $state<GDPRProviderResult[]>([]);

	// Update state
	let saving = $state(false);
	let editGdprConsent = $state(false);
	let editMarketingConsent = $state(false);

	async function searchContact() {
		if (!searchEmail.trim()) {
			error = 'Please enter an email address';
			return;
		}

		searching = true;
		error = '';
		success = '';
		consentInfo = null;
		exportResult = null;

		try {
			// First, look up the contact by email using the GDPR lookup endpoint
			const lookup = await api.lookupContact({ email: searchEmail.trim() });
			if (!lookup.found || !lookup.contact_id) {
				error = 'No contact found with that email address';
				return;
			}

			contactId = lookup.contact_id;

			// Get the consent info
			const info = await api.getContactConsent({}, contactId);
			consentInfo = info;
			editGdprConsent = info.gdpr_consent;
			editMarketingConsent = info.marketing_consent;
		} catch (err: any) {
			console.error('Failed to search contact:', err);
			error = err.message || 'Failed to find contact';
		} finally {
			searching = false;
		}
	}

	async function updateConsent() {
		if (!contactId) return;

		saving = true;
		error = '';
		success = '';

		try {
			const updated = await api.updateContactConsent({}, {
				gdpr_consent: editGdprConsent,
				marketing_consent: editMarketingConsent
			}, contactId);

			consentInfo = updated;
			success = 'Consent preferences updated successfully';
			setTimeout(() => { success = ''; }, 3000);
		} catch (err: any) {
			console.error('Failed to update consent:', err);
			error = err.message || 'Failed to update consent';
		} finally {
			saving = false;
		}
	}

	// Saves the export the server returned as a file on this computer
	function downloadExport() {
		if (!exportResult) return;
		const url = URL.createObjectURL(new Blob([exportResult.data], { type: 'application/json' }));
		const a = document.createElement('a');
		a.href = url;
		a.download = exportResult.filename;
		a.click();
		URL.revokeObjectURL(url);
	}

	async function exportData() {
		if (!contactId) return;

		exporting = true;
		error = '';
		success = '';

		try {
			const result = await api.exportContactData({}, contactId);
			exportResult = result;
			downloadExport();
			success = 'Data export downloaded';
		} catch (err: any) {
			console.error('Failed to export data:', err);
			error = err.message || 'Failed to export contact data';
		} finally {
			exporting = false;
		}
	}

	function confirmDelete() {
		showDeleteConfirm = true;
	}

	async function executeDelete() {
		if (!contactId) return;

		deleting = true;
		error = '';
		providerResults = [];

		try {
			const res = await api.deleteContactData({}, {
				confirm: true,
				delete_email: deleteAlsoEmail
			}, contactId);

			showDeleteConfirm = false;
			consentInfo = null;
			contactId = '';
			searchEmail = '';
			providerResults = res.provider_results ?? [];
			success = `Contact data deleted. Audit record ${res.audit_record_id}.`;
		} catch (err) {
			console.error('Failed to delete data:', err);
			error = err instanceof Error ? err.message : 'Failed to delete contact data';
		} finally {
			deleting = false;
		}
	}

</script>

<svelte:head>
	<title>Privacy & GDPR - Settings</title>
</svelte:head>

{#if error}
	<div class="mb-6">
		<Alert type="error" title="Error">
			<p>{error}</p>
		</Alert>
	</div>
{/if}

{#if success}
	<div class="mb-6">
		<Alert type="success" title="Success">
			<p>{success}</p>
			{#if providerResults.length > 0}
				<p class="mt-2 font-medium">What each email provider did:</p>
				<ul class="mt-1 list-disc pl-5 space-y-1">
					{#each providerResults as r (r.pathway)}
						<li>
							<span class="font-medium">{r.pathway} ({r.provider}):</span>
							<span class={r.failed ? 'text-error' : ''}>{r.result}</span>
						</li>
					{/each}
				</ul>
			{/if}
		</Alert>
	</div>
{/if}

<div class="space-y-6">
	<Card>
		<div class="flex items-center gap-3 mb-4">
			<Shield class="h-5 w-5 text-primary" />
			<h2 class="text-lg font-medium text-text">GDPR Compliance Tools</h2>
		</div>
		<p class="text-sm text-text-muted mb-4">
			Manage contact consent preferences, export personal data, or delete contact information to comply with GDPR and privacy regulations.
		</p>

		<div class="flex gap-3">
			<div class="flex-1">
				<Input
					type="email"
					bind:value={searchEmail}
					placeholder="Enter contact email address"
					onkeydown={(e: KeyboardEvent) => e.key === 'Enter' && searchContact()}
				/>
			</div>
			<Button
				type="primary"
				onclick={searchContact}
				disabled={searching}
			>
				<Search class="mr-2 h-4 w-4" />
				{searching ? 'Searching...' : 'Search Contact'}
			</Button>
		</div>
	</Card>

	{#if searching}
		<div class="flex justify-center py-8">
			<LoadingSpinner size="large" />
		</div>
	{:else if consentInfo}
		<Card>
			<h2 class="text-lg font-medium text-text mb-4">Contact Information</h2>
			<div class="space-y-4">
				<div class="flex items-center justify-between py-2 border-b border-border">
					<span class="text-sm text-text-muted">Email</span>
					<span class="text-sm font-medium text-text">{consentInfo.email}</span>
				</div>
				<div class="flex items-center justify-between py-2 border-b border-border">
					<span class="text-sm text-text-muted">Contact ID</span>
					<code class="text-sm font-mono text-text">{consentInfo.contact_id}</code>
				</div>
				<div class="flex items-center justify-between py-2 border-b border-border">
					<span class="text-sm text-text-muted">Account Created</span>
					<span class="text-sm text-text">{formatDate(consentInfo.created_at, 'datetime') || 'Never'}</span>
				</div>
				{#if consentInfo.data_retention_policy}
					<div class="flex items-center justify-between py-2 border-b border-border">
						<span class="text-sm text-text-muted">Data Retention</span>
						<span class="text-sm text-text">{consentInfo.data_retention_policy}</span>
					</div>
				{/if}
			</div>
		</Card>

		<Card>
			<h2 class="text-lg font-medium text-text mb-4">Consent Preferences</h2>
			<div class="space-y-4">
				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm font-medium text-text">GDPR Consent</p>
						<p class="text-xs text-text-muted">
							Consented: {formatDate(consentInfo.gdpr_consent_at, 'datetime') || 'Never'}
						</p>
					</div>
					<div class="flex items-center gap-3">
						<Badge type={editGdprConsent ? 'success' : 'secondary'}>
							{editGdprConsent ? 'Granted' : 'Not Granted'}
						</Badge>
						<Toggle
							bind:checked={editGdprConsent}
							label=""
						/>
					</div>
				</div>

				<div class="flex items-center justify-between">
					<div>
						<p class="text-sm font-medium text-text">Marketing Consent</p>
						<p class="text-xs text-text-muted">
							Consented: {formatDate(consentInfo.marketing_consent_at, 'datetime') || 'Never'}
						</p>
					</div>
					<div class="flex items-center gap-3">
						<Badge type={editMarketingConsent ? 'success' : 'secondary'}>
							{editMarketingConsent ? 'Granted' : 'Not Granted'}
						</Badge>
						<Toggle
							bind:checked={editMarketingConsent}
							label=""
						/>
					</div>
				</div>

				<div class="pt-4 border-t border-border">
					<Button
						type="primary"
						onclick={updateConsent}
						disabled={saving}
					>
						{saving ? 'Saving...' : 'Save Consent Changes'}
					</Button>
				</div>
			</div>
		</Card>

		<Card>
			<h2 class="text-lg font-medium text-text mb-4">Data Actions</h2>
			<div class="space-y-4">
				<div class="flex items-center justify-between p-4 bg-bg-secondary rounded-lg">
					<div class="flex items-center gap-3">
						<FileText class="h-5 w-5 text-primary" />
						<div>
							<p class="text-sm font-medium text-text">Export Personal Data</p>
							<p class="text-xs text-text-muted">Download all data associated with this contact (GDPR Article 20)</p>
						</div>
					</div>
					<Button
						type="secondary"
						onclick={exportData}
						disabled={exporting}
					>
						<Download class="mr-2 h-4 w-4" />
						{exporting ? 'Exporting...' : 'Export Data'}
					</Button>
				</div>

				{#if exportResult}
					<div class="p-4 bg-green-500/10 border border-green-500/20 rounded-lg">
						<p class="text-sm font-medium text-text mb-2">Export Ready</p>
						<p class="text-xs text-text-muted mb-3">
							{exportResult.filename} ({exportResult.format})
						</p>
						<Button type="primary" onclick={downloadExport}>
							<Download class="mr-2 h-4 w-4" />
							Download Export
						</Button>
					</div>
				{/if}

				<div class="flex items-center justify-between p-4 bg-red-500/10 border border-red-500/20 rounded-lg">
					<div class="flex items-center gap-3">
						<UserX class="h-5 w-5 text-red-500" />
						<div>
							<p class="text-sm font-medium text-text">Delete All Data</p>
							<p class="text-xs text-text-muted">Permanently delete this contact and all associated data (GDPR Article 17)</p>
						</div>
					</div>
					<Button
						type="danger"
						onclick={confirmDelete}
					>
						<Trash2 class="mr-2 h-4 w-4" />
						Delete Data
					</Button>
				</div>
			</div>
		</Card>
	{/if}
</div>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Contact Data"
	description="This will permanently delete all data associated with this contact. This action cannot be undone."
	actionLabel={deleting ? 'Deleting...' : 'Delete All Data'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => { showDeleteConfirm = false; }}
>
	<div class="mt-4">
		<label class="flex items-center gap-2">
			<input type="checkbox" bind:checked={deleteAlsoEmail} class="rounded" />
			<span class="text-sm text-text">Also ask the brand's email providers to delete what they keep about this address</span>
		</label>
		<ul class="mt-2 ml-6 list-disc space-y-1 text-xs text-text-muted">
			<li>
				Amazon SES: the address comes off the SES account's suppression list, the only per-recipient record SES's
				API can delete. SES keeps no sending log that can be deleted.
			</li>
			<li>
				SMTP relays and mailboxes have no way to ask, so nothing is sent to them. The result names each one, so you
				can ask its operator yourself.
			</li>
			<li>Outlet keeps its own bounce and suppression records for the address, so it isn't mailed again.</li>
			<li>The request is recorded in the GDPR audit log either way.</li>
		</ul>
	</div>
</AlertDialog>
