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
	// The windows the preview counted: the delete removes exactly those.
	let previewedWindows = $state<{ no_opens_days: number; no_clicks_days: number } | null>(null);

	// Unconfirmed cleanup state
	let unconfirmedDays = $state(30);
	let unconfirmedLoading = $state(false);
	let unconfirmedResult = $state<HousekeepingResponse | null>(null);

	// Error state
	let error = $state('');

	// Confirmation dialogs
	let showInactiveConfirm = $state(false);
	let showUnconfirmedConfirm = $state(false);

	// A changed window makes the preview stale: preview again before deleting.
	function clearInactivePreview() {
		inactiveResult = null;
		previewedWindows = null;
	}

	async function previewInactive() {
		inactiveLoading = true;
		error = '';
		clearInactivePreview();
		const windows = { no_opens_days: noOpensDays, no_clicks_days: noClicksDays };
		try {
			inactiveResult = await housekeepingInactive({ ...windows, dry_run: true });
			previewedWindows = windows;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to preview inactive contacts';
		} finally {
			inactiveLoading = false;
		}
	}

	async function executeInactive() {
		if (!previewedWindows) return;
		inactiveLoading = true;
		error = '';
		try {
			inactiveResult = await housekeepingInactive({ ...previewedWindows, dry_run: false });
			previewedWindows = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete inactive contacts';
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
				<div class="p-3 rounded-lg bg-warning/10 text-warning">
					<UserX class="h-6 w-6" />
				</div>
				<div class="flex-1">
					<h2 class="text-lg font-medium text-text">Inactive Contacts</h2>
					<p class="mt-1 text-sm text-text-muted">
						Remove subscribers (added, imported or confirmed, and still subscribed) who haven't opened any email in the first window and haven't clicked any in the second: campaigns, sequences, outreach and transactional emails all count. Contacts newer than either window are kept. This helps improve your deliverability and open rates.
					</p>

					<div class="mt-4 grid grid-cols-1 sm:grid-cols-2 gap-4">
						<div>
							<label for="no-opens" class="form-label">No opens in (days)</label>
							<input
								id="no-opens"
								type="number"
								bind:value={noOpensDays}
								oninput={clearInactivePreview}
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
								oninput={clearInactivePreview}
								min={30}
								max={365}
								class="input input-bordered w-full"
							/>
						</div>
					</div>

					{#if inactiveResult}
						<div class="mt-4 p-4 rounded-lg {inactiveResult.dry_run ? 'bg-warning/10 border border-warning/30' : 'bg-success/10 border border-success/30'}">
							<div class="flex items-center gap-2">
								{#if inactiveResult.dry_run}
									<AlertTriangle class="h-5 w-5 text-warning" />
									<span class="font-medium text-warning">Preview:</span>
								{:else}
									<CheckCircle class="h-5 w-5 text-success" />
									<span class="font-medium text-success">Completed:</span>
								{/if}
								<span class="{inactiveResult.dry_run ? 'text-warning' : 'text-success'}">
									{inactiveResult.message}
								</span>
							</div>
							{#if inactiveResult.dry_run && inactiveResult.affected_count > 0}
								<p class="mt-2 text-sm text-warning">
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
				<div class="p-3 rounded-lg bg-info/10 text-info">
					<Clock class="h-6 w-6" />
				</div>
				<div class="flex-1">
					<h2 class="text-lg font-medium text-text">Unconfirmed Subscriptions</h2>
					<p class="mt-1 text-sm text-text-muted">
						Remove contacts who signed up on a double opt-in list and never confirmed, waiting longer than the days below. Anyone on a list as a subscriber, or found for outreach, is kept.
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
						<div class="mt-4 p-4 rounded-lg {unconfirmedResult.dry_run ? 'bg-warning/10 border border-warning/30' : 'bg-success/10 border border-success/30'}">
							<div class="flex items-center gap-2">
								{#if unconfirmedResult.dry_run}
									<AlertTriangle class="h-5 w-5 text-warning" />
									<span class="font-medium text-warning">Preview:</span>
								{:else}
									<CheckCircle class="h-5 w-5 text-success" />
									<span class="font-medium text-success">Completed:</span>
								{/if}
								<span class="{unconfirmedResult.dry_run ? 'text-warning' : 'text-success'}">
									{unconfirmedResult.message}
								</span>
							</div>
							{#if unconfirmedResult.dry_run && unconfirmedResult.affected_count > 0}
								<p class="mt-2 text-sm text-warning">
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
