<script lang="ts">
	import { page } from '$app/stores';
	import {
		housekeepingInactive,
		housekeepingUnconfirmed,
		type HousekeepingResponse
	} from '$lib/api';
	import {
		Button,
		Card,
		Alert,
		LoadingSpinner,
		Badge,
		AlertDialog
	} from '$lib/components/ui';
	import {
		Trash2,
		UserX,
		Clock,
		AlertTriangle,
		CheckCircle
	} from '@lucide/svelte';

	let basePath = $derived(`/${$page.params.brandSlug}`);

	// Inactive cleanup state
	let noOpensDays = $state(90);
	let noClicksDays = $state(180);
	let inactiveLoading = $state(false);
	let inactiveResult = $state<HousekeepingResponse | null>(null);

	// Unconfirmed cleanup state
	let unconfirmedDays = $state(30);
	let unconfirmedLoading = $state(false);
	let unconfirmedResult = $state<HousekeepingResponse | null>(null);

	// Error state
	let error = $state('');

	// Confirmation dialogs
	let showInactiveConfirm = $state(false);
	let showUnconfirmedConfirm = $state(false);

	async function previewInactive() {
		inactiveLoading = true;
		error = '';
		inactiveResult = null;
		try {
			inactiveResult = await housekeepingInactive({
				no_opens_days: noOpensDays,
				no_clicks_days: noClicksDays,
				dry_run: true
			});
		} catch (err: any) {
			error = err.message || 'Failed to preview inactive contacts';
		} finally {
			inactiveLoading = false;
		}
	}

	async function executeInactive() {
		inactiveLoading = true;
		error = '';
		try {
			inactiveResult = await housekeepingInactive({
				no_opens_days: noOpensDays,
				no_clicks_days: noClicksDays,
				dry_run: false
			});
		} catch (err: any) {
			error = err.message || 'Failed to delete inactive contacts';
		} finally {
			inactiveLoading = false;
		}
	}

	async function previewUnconfirmed() {
		unconfirmedLoading = true;
		error = '';
		unconfirmedResult = null;
		try {
			unconfirmedResult = await housekeepingUnconfirmed({
				older_than_days: unconfirmedDays,
				dry_run: true
			});
		} catch (err: any) {
			error = err.message || 'Failed to preview unconfirmed contacts';
		} finally {
			unconfirmedLoading = false;
		}
	}

	async function executeUnconfirmed() {
		unconfirmedLoading = true;
		error = '';
		try {
			unconfirmedResult = await housekeepingUnconfirmed({
				older_than_days: unconfirmedDays,
				dry_run: false
			});
		} catch (err: any) {
			error = err.message || 'Failed to delete unconfirmed contacts';
		} finally {
			unconfirmedLoading = false;
		}
	}
</script>

<svelte:head>
	<title>Housekeeping | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<!-- Header -->
	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-text">Housekeeping</h1>
		<p class="mt-1 text-sm text-text-muted">
			Clean up your subscriber lists by removing inactive or unconfirmed contacts.
		</p>
	</div>

	{#if error}
		<Alert type="error" title="Error" class="mb-6">
			<p>{error}</p>
		</Alert>
	{/if}

	<div class="space-y-6">
		<!-- Inactive Contacts -->
		<Card>
			<div class="flex items-start gap-4">
				<div class="p-3 rounded-lg bg-orange-100 text-orange-600">
					<UserX class="h-6 w-6" />
				</div>
				<div class="flex-1">
					<h2 class="text-lg font-medium text-text">Inactive Contacts</h2>
					<p class="mt-1 text-sm text-text-muted">
						Remove contacts who haven't opened or clicked any emails in a long time. This helps improve your deliverability and open rates.
					</p>

					<div class="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
						<div>
							<label for="no-opens" class="form-label">No opens in (days)</label>
							<input
								id="no-opens"
								type="number"
								bind:value={noOpensDays}
								min={30}
								max={365}
								class="input input-bordered w-full"
							/>
						</div>
						<div>
							<label for="no-clicks" class="form-label">No clicks in (days)</label>
							<input
								id="no-clicks"
								type="number"
								bind:value={noClicksDays}
								min={30}
								max={365}
								class="input input-bordered w-full"
							/>
						</div>
					</div>

					{#if inactiveResult}
						<div class="mt-4 p-4 rounded-lg {inactiveResult.dry_run ? 'bg-yellow-50 border border-yellow-200' : 'bg-green-50 border border-green-200'}">
							<div class="flex items-center gap-2">
								{#if inactiveResult.dry_run}
									<AlertTriangle class="h-5 w-5 text-yellow-600" />
									<span class="font-medium text-yellow-800">Preview:</span>
								{:else}
									<CheckCircle class="h-5 w-5 text-green-600" />
									<span class="font-medium text-green-800">Completed:</span>
								{/if}
								<span class="{inactiveResult.dry_run ? 'text-yellow-700' : 'text-green-700'}">
									{inactiveResult.message}
								</span>
							</div>
							{#if inactiveResult.dry_run && inactiveResult.affected_count > 0}
								<p class="mt-2 text-sm text-yellow-700">
									{inactiveResult.affected_count} contact{inactiveResult.affected_count === 1 ? '' : 's'} will be deleted.
								</p>
							{/if}
						</div>
					{/if}

					<div class="mt-4 flex gap-3">
						<Button
							type="secondary"
							onclick={previewInactive}
							disabled={inactiveLoading}
						>
							{#if inactiveLoading && !inactiveResult}
								Checking...
							{:else}
								Preview
							{/if}
						</Button>
						{#if inactiveResult?.dry_run && inactiveResult.affected_count > 0}
							<Button
								type="danger"
								onclick={() => showInactiveConfirm = true}
								disabled={inactiveLoading}
							>
								{#if inactiveLoading}
									Deleting...
								{:else}
									<Trash2 class="mr-2 h-4 w-4" />
									Delete {inactiveResult.affected_count} Contact{inactiveResult.affected_count === 1 ? '' : 's'}
								{/if}
							</Button>
						{/if}
					</div>
				</div>
			</div>
		</Card>

		<!-- Unconfirmed Subscriptions -->
		<Card>
			<div class="flex items-start gap-4">
				<div class="p-3 rounded-lg bg-blue-100 text-blue-600">
					<Clock class="h-6 w-6" />
				</div>
				<div class="flex-1">
					<h2 class="text-lg font-medium text-text">Unconfirmed Subscriptions</h2>
					<p class="mt-1 text-sm text-text-muted">
						Remove pending subscriptions from contacts who never confirmed their email. This cleans up contacts who signed up but didn't complete double opt-in.
					</p>

					<div class="mt-4 max-w-xs">
						<label for="unconfirmed-days" class="form-label">Older than (days)</label>
						<input
							id="unconfirmed-days"
							type="number"
							bind:value={unconfirmedDays}
							min={7}
							max={90}
							class="input input-bordered w-full"
						/>
					</div>

					{#if unconfirmedResult}
						<div class="mt-4 p-4 rounded-lg {unconfirmedResult.dry_run ? 'bg-yellow-50 border border-yellow-200' : 'bg-green-50 border border-green-200'}">
							<div class="flex items-center gap-2">
								{#if unconfirmedResult.dry_run}
									<AlertTriangle class="h-5 w-5 text-yellow-600" />
									<span class="font-medium text-yellow-800">Preview:</span>
								{:else}
									<CheckCircle class="h-5 w-5 text-green-600" />
									<span class="font-medium text-green-800">Completed:</span>
								{/if}
								<span class="{unconfirmedResult.dry_run ? 'text-yellow-700' : 'text-green-700'}">
									{unconfirmedResult.message}
								</span>
							</div>
							{#if unconfirmedResult.dry_run && unconfirmedResult.affected_count > 0}
								<p class="mt-2 text-sm text-yellow-700">
									{unconfirmedResult.affected_count} subscription{unconfirmedResult.affected_count === 1 ? '' : 's'} will be deleted.
								</p>
							{/if}
						</div>
					{/if}

					<div class="mt-4 flex gap-3">
						<Button
							type="secondary"
							onclick={previewUnconfirmed}
							disabled={unconfirmedLoading}
						>
							{#if unconfirmedLoading && !unconfirmedResult}
								Checking...
							{:else}
								Preview
							{/if}
						</Button>
						{#if unconfirmedResult?.dry_run && unconfirmedResult.affected_count > 0}
							<Button
								type="danger"
								onclick={() => showUnconfirmedConfirm = true}
								disabled={unconfirmedLoading}
							>
								{#if unconfirmedLoading}
									Deleting...
								{:else}
									<Trash2 class="mr-2 h-4 w-4" />
									Delete {unconfirmedResult.affected_count} Subscription{unconfirmedResult.affected_count === 1 ? '' : 's'}
								{/if}
							</Button>
						{/if}
					</div>
				</div>
			</div>
		</Card>

		<!-- Warning -->
		<Alert type="warning" title="Permanent Deletion">
			<p>
				Housekeeping operations permanently delete data. Always preview changes before executing.
				Deleted contacts and subscriptions cannot be recovered.
			</p>
		</Alert>
	</div>
</div>

<AlertDialog
	bind:open={showInactiveConfirm}
	title="Delete Inactive Contacts"
	description={`This will permanently delete ${inactiveResult?.affected_count || 0} contacts. This action cannot be undone.`}
	actionLabel={inactiveLoading ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeInactive}
	onCancel={() => showInactiveConfirm = false}
/>

<AlertDialog
	bind:open={showUnconfirmedConfirm}
	title="Delete Unconfirmed Subscriptions"
	description={`This will permanently delete ${unconfirmedResult?.affected_count || 0} unconfirmed subscriptions. This action cannot be undone.`}
	actionLabel={unconfirmedLoading ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeUnconfirmed}
	onCancel={() => showUnconfirmedConfirm = false}
/>
