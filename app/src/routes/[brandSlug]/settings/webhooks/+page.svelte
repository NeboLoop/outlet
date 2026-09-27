<script lang="ts">
	import * as api from '$lib/api';
	import type { WebhookInfo } from '$lib/api';
	import {
		Card,
		Alert,
		LoadingSpinner,
		Badge,
		Button,
		Input,
		Modal,
		Checkbox,
		Table,
		AlertDialog
	} from '$lib/components/ui';
	import { Plus, Trash2, TestTube, Copy, Eye, EyeOff, Check, X, RefreshCw } from '@lucide/svelte';

	let loading = $state(true);
	let error = $state('');
	let webhooks = $state<WebhookInfo[]>([]);

	// Modal states
	let showCreateModal = $state(false);
	let showLogsModal = $state(false);
	let showSecretModal = $state(false);

	// Form state
	let newWebhook = $state({
		url: '',
		events: [] as string[],
		active: true
	});
	let createdSecret = $state('');
	let selectedWebhookId = $state('');
	let webhookLogs = $state<api.WebhookLogInfo[]>([]);
	let logsLoading = $state(false);

	// Delete confirmation
	let showDeleteConfirm = $state(false);
	let deleteWebhookId = $state('');
	let deleting = $state(false);

	// Testing state
	let testingWebhook = $state<string | null>(null);
	let testResult = $state<api.TestWebhookResponse | null>(null);

	// Available webhook events
	const availableEvents = [
		{ value: 'checkout.completed', label: 'Checkout Completed', category: 'Orders' },
		{ value: 'order.created', label: 'Order Created', category: 'Orders' },
		{ value: 'order.completed', label: 'Order Completed', category: 'Orders' },
		{ value: 'payment.succeeded', label: 'Payment Succeeded', category: 'Payments' },
		{ value: 'payment.failed', label: 'Payment Failed', category: 'Payments' },
		{ value: 'refund.created', label: 'Refund Created', category: 'Payments' },
		{ value: 'subscription.created', label: 'Subscription Created', category: 'Subscriptions' },
		{ value: 'subscription.updated', label: 'Subscription Updated', category: 'Subscriptions' },
		{ value: 'subscription.canceled', label: 'Subscription Canceled', category: 'Subscriptions' },
		{ value: 'subscription.renewed', label: 'Subscription Renewed', category: 'Subscriptions' },
		{ value: 'customer.created', label: 'Customer Created', category: 'Customers' },
		{ value: 'customer.updated', label: 'Customer Updated', category: 'Customers' },
		{ value: 'contact.created', label: 'Contact Created', category: 'Contacts' },
		{ value: 'contact.unsubscribed', label: 'Contact Unsubscribed', category: 'Contacts' },
		{ value: 'email.sent', label: 'Email Sent', category: 'Email' },
		{ value: 'email.bounced', label: 'Email Bounced', category: 'Email' },
		{ value: 'email.opened', label: 'Email Opened', category: 'Email' },
		{ value: 'email.clicked', label: 'Email Clicked', category: 'Email' }
	];

	// Group events by category
	let eventsByCategory = $derived(() => {
		const groups: Record<string, typeof availableEvents> = {};
		for (const event of availableEvents) {
			if (!groups[event.category]) {
				groups[event.category] = [];
			}
			groups[event.category].push(event);
		}
		return groups;
	});

	$effect(() => {
		loadWebhooks();
	});

	async function loadWebhooks() {
		loading = true;
		error = '';
		try {
			const resp = await api.adminListWebhooks();
			webhooks = resp.webhooks || [];
		} catch (err: any) {
			console.error('Failed to load webhooks:', err);
			error = err.message || 'Failed to load webhooks';
		} finally {
			loading = false;
		}
	}

	async function createWebhook() {
		if (!newWebhook.url || newWebhook.events.length === 0) {
			error = 'URL and at least one event are required';
			return;
		}

		try {
			const resp = await api.adminCreateWebhook({
				url: newWebhook.url,
				events: newWebhook.events,
				active: newWebhook.active
			});
			createdSecret = resp.secret;
			showCreateModal = false;
			showSecretModal = true;
			newWebhook = { url: '', events: [], active: true };
			await loadWebhooks();
		} catch (err: any) {
			console.error('Failed to create webhook:', err);
			error = err.message || 'Failed to create webhook';
		}
	}

	async function toggleWebhook(webhook: WebhookInfo) {
		try {
			await api.adminUpdateWebhook({}, { active: !webhook.active }, webhook.id);
			await loadWebhooks();
		} catch (err: any) {
			console.error('Failed to update webhook:', err);
			error = err.message || 'Failed to update webhook';
		}
	}

	function confirmDeleteWebhook(id: string) {
		deleteWebhookId = id;
		showDeleteConfirm = true;
	}

	async function executeDeleteWebhook() {
		deleting = true;
		try {
			await api.adminDeleteWebhook({}, deleteWebhookId);
			await loadWebhooks();
		} catch (err: any) {
			console.error('Failed to delete webhook:', err);
			error = err.message || 'Failed to delete webhook';
		} finally {
			deleting = false;
		}
	}

	async function testWebhook(id: string) {
		testingWebhook = id;
		testResult = null;
		try {
			testResult = await api.adminTestWebhook({}, id);
		} catch (err: any) {
			console.error('Failed to test webhook:', err);
			testResult = { success: false, status_code: 0, error: err.message || 'Test failed' };
		} finally {
			testingWebhook = null;
		}
	}

	async function viewLogs(id: string) {
		selectedWebhookId = id;
		logsLoading = true;
		showLogsModal = true;
		webhookLogs = [];

		try {
			const resp = await api.adminListWebhookLogs({ limit: 50 }, id);
			webhookLogs = resp.logs || [];
		} catch (err: any) {
			console.error('Failed to load webhook logs:', err);
		} finally {
			logsLoading = false;
		}
	}

	function toggleEvent(event: string) {
		const index = newWebhook.events.indexOf(event);
		if (index === -1) {
			newWebhook.events = [...newWebhook.events, event];
		} else {
			newWebhook.events = newWebhook.events.filter((e) => e !== event);
		}
	}

	function copySecret() {
		navigator.clipboard.writeText(createdSecret);
	}

	function formatDate(dateStr?: string): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleString();
	}
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-lg font-medium text-text">Webhooks</h2>
			<p class="mt-1 text-sm text-text-muted">
				Receive real-time notifications when events happen in your account.
			</p>
		</div>
		<Button onclick={() => (showCreateModal = true)}>
			<Plus size={16} class="mr-2" />
			Add Webhook
		</Button>
	</div>

	{#if error}
		<Alert variant="error">{error}</Alert>
	{/if}

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="lg" />
		</div>
	{:else if webhooks.length === 0}
		<Card class="py-12 text-center">
			<p class="text-text-muted">No webhooks configured yet.</p>
			<Button class="mt-4" onclick={() => (showCreateModal = true)}>
				<Plus size={16} class="mr-2" />
				Add Your First Webhook
			</Button>
		</Card>
	{:else}
		<Card>
			<Table>
				<thead>
					<tr>
						<th>URL</th>
						<th>Events</th>
						<th>Status</th>
						<th>Deliveries</th>
						<th>Last Delivery</th>
						<th class="text-right">Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each webhooks as webhook}
						<tr>
							<td class="font-mono text-sm max-w-xs truncate" title={webhook.url}>
								{webhook.url}
							</td>
							<td>
								<div class="flex flex-wrap gap-1">
									{#each webhook.events.slice(0, 2) as event}
										<Badge variant="secondary" class="text-xs">{event}</Badge>
									{/each}
									{#if webhook.events.length > 2}
										<Badge variant="secondary" class="text-xs">
											+{webhook.events.length - 2} more
										</Badge>
									{/if}
								</div>
							</td>
							<td>
								{#if webhook.active}
									<Badge variant="success">Active</Badge>
								{:else}
									<Badge variant="secondary">Inactive</Badge>
								{/if}
							</td>
							<td>
								<div class="text-sm">
									<span class="text-green-600">{webhook.deliveries_success || 0}</span>
									<span class="text-text-muted"> / </span>
									<span class="text-red-600">{webhook.deliveries_failed || 0}</span>
								</div>
							</td>
							<td class="text-sm text-text-muted">
								{formatDate(webhook.last_delivery_at)}
							</td>
							<td>
								<div class="flex items-center justify-end gap-2">
									<Button
										variant="ghost"
										size="sm"
										onclick={() => viewLogs(webhook.id)}
										title="View Logs"
									>
										<Eye size={16} />
									</Button>
									<Button
										variant="ghost"
										size="sm"
										onclick={() => testWebhook(webhook.id)}
										disabled={testingWebhook === webhook.id}
										title="Test Webhook"
									>
										{#if testingWebhook === webhook.id}
											<RefreshCw size={16} class="animate-spin" />
										{:else}
											<TestTube size={16} />
										{/if}
									</Button>
									<Button
										variant="ghost"
										size="sm"
										onclick={() => toggleWebhook(webhook)}
										title={webhook.active ? 'Disable' : 'Enable'}
									>
										{#if webhook.active}
											<EyeOff size={16} />
										{:else}
											<Eye size={16} />
										{/if}
									</Button>
									<Button
										variant="ghost"
										size="sm"
										class="text-red-600 hover:text-red-700"
										onclick={() => confirmDeleteWebhook(webhook.id)}
										title="Delete"
									>
										<Trash2 size={16} />
									</Button>
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		</Card>
	{/if}

	{#if testResult}
		<Alert variant={testResult.success ? 'success' : 'error'} dismissable onclose={() => (testResult = null)}>
			<strong>Test {testResult.success ? 'Successful' : 'Failed'}</strong>
			{#if testResult.status_code}
				- Status: {testResult.status_code}
			{/if}
			{#if testResult.error}
				<p class="mt-1 text-sm">{testResult.error}</p>
			{/if}
		</Alert>
	{/if}
</div>

<!-- Create Webhook Modal -->
<Modal bind:open={showCreateModal} title="Add Webhook">
	<div class="space-y-4">
		<div>
			<label class="form-label" for="webhook-url">Endpoint URL</label>
			<Input
				id="webhook-url"
				type="url"
				bind:value={newWebhook.url}
				placeholder="https://your-server.com/webhook"
			/>
		</div>

		<div>
			<label class="form-label">Events to Subscribe</label>
			<div class="mt-2 space-y-4 max-h-64 overflow-y-auto">
				{#each Object.entries(eventsByCategory()) as [category, events]}
					<div>
						<h4 class="text-sm font-medium text-text-muted mb-2">{category}</h4>
						<div class="space-y-2 pl-2">
							{#each events as event}
								<Checkbox
									label={event.label}
									checked={newWebhook.events.includes(event.value)}
									onchange={() => toggleEvent(event.value)}
								/>
							{/each}
						</div>
					</div>
				{/each}
			</div>
		</div>

		<div>
			<Checkbox label="Active" bind:checked={newWebhook.active} />
		</div>
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-3">
			<Button variant="secondary" onclick={() => (showCreateModal = false)}>Cancel</Button>
			<Button onclick={createWebhook}>Create Webhook</Button>
		</div>
	{/snippet}
</Modal>

<!-- Secret Display Modal -->
<Modal bind:open={showSecretModal} title="Webhook Created">
	<div class="space-y-4">
		<Alert variant="warning">
			Save this secret now. You won't be able to see it again.
		</Alert>

		<div>
			<label class="form-label">Signing Secret</label>
			<div class="flex gap-2">
				<Input value={createdSecret} readonly class="font-mono text-sm" />
				<Button variant="secondary" onclick={copySecret}>
					<Copy size={16} />
				</Button>
			</div>
			<p class="mt-2 text-sm text-text-muted">
				Use this secret to verify that webhook payloads are from Outlet. Check the
				<code>X-Webhook-Signature</code> header.
			</p>
		</div>
	</div>

	{#snippet footer()}
		<Button onclick={() => (showSecretModal = false)}>Done</Button>
	{/snippet}
</Modal>

<!-- Logs Modal -->
<Modal bind:open={showLogsModal} title="Webhook Delivery Logs" size="lg">
	{#if logsLoading}
		<div class="flex justify-center py-8">
			<LoadingSpinner />
		</div>
	{:else if webhookLogs.length === 0}
		<p class="text-center text-text-muted py-8">No delivery logs yet.</p>
	{:else}
		<div class="max-h-96 overflow-y-auto">
			<Table>
				<thead>
					<tr>
						<th>Event</th>
						<th>Status</th>
						<th>Duration</th>
						<th>Time</th>
					</tr>
				</thead>
				<tbody>
					{#each webhookLogs as log}
						<tr>
							<td>
								<Badge variant="secondary">{log.event}</Badge>
							</td>
							<td>
								{#if log.status_code >= 200 && log.status_code < 300}
									<span class="flex items-center gap-1 text-green-600">
										<Check size={14} />
										{log.status_code}
									</span>
								{:else if log.error}
									<span class="flex items-center gap-1 text-red-600">
										<X size={14} />
										Error
									</span>
								{:else}
									<span class="text-red-600">{log.status_code}</span>
								{/if}
							</td>
							<td class="text-sm text-text-muted">
								{log.duration_ms || 0}ms
							</td>
							<td class="text-sm text-text-muted">
								{formatDate(log.delivered_at)}
							</td>
						</tr>
					{/each}
				</tbody>
			</Table>
		</div>
	{/if}

	{#snippet footer()}
		<Button variant="secondary" onclick={() => (showLogsModal = false)}>Close</Button>
	{/snippet}
</Modal>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Webhook"
	description="Are you sure you want to delete this webhook?"
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDeleteWebhook}
	onCancel={() => showDeleteConfirm = false}
/>
