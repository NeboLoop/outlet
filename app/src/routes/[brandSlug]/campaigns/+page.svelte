<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		listCampaigns,
		deleteCampaign,
		type CampaignInfo
	} from '$lib/api';
	import {
		Button,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		SearchInput,
		DropdownMenu,
		Tabs,
		AlertDialog
	} from '$lib/components/ui';
	import { Plus, Send, MoreVertical, Trash2, Edit, Copy, Play, Pause, BarChart3 } from '@lucide/svelte';

	let loading = $state(true);
	let campaigns = $state<CampaignInfo[]>([]);
	let error = $state('');
	let searchQuery = $state('');
	let activeTab = $state('all');

	// Build base path with brandSlug
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// Status tabs
	const tabs = [
		{ id: 'all', label: 'All' },
		{ id: 'draft', label: 'Drafts' },
		{ id: 'scheduled', label: 'Scheduled' },
		{ id: 'sending', label: 'Sending' },
		{ id: 'sent', label: 'Sent' },
	];

	// Filtered campaigns based on search and tab
	let filteredCampaigns = $derived(() => {
		let result = campaigns;

		// Filter by status tab
		if (activeTab !== 'all') {
			result = result.filter(c => c.status === activeTab);
		}

		// Filter by search
		if (searchQuery) {
			const query = searchQuery.toLowerCase();
			result = result.filter(c =>
				c.name.toLowerCase().includes(query) ||
				c.subject.toLowerCase().includes(query)
			);
		}

		return result;
	});

	// Count by status for tab badges
	let statusCounts = $derived(() => {
		const counts: Record<string, number> = { all: campaigns.length };
		for (const c of campaigns) {
			counts[c.status] = (counts[c.status] || 0) + 1;
		}
		return counts;
	});

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';

		try {
			const response = await listCampaigns({});
			campaigns = response.campaigns || [];
		} catch (err) {
			console.error('Failed to load campaigns:', err);
			error = 'Failed to load campaigns';
		} finally {
			loading = false;
		}
	}

	let showDeleteConfirm = $state(false);
	let deleteCampaignItem = $state<CampaignInfo | null>(null);
	let deleting = $state(false);

	function confirmDelete(campaign: CampaignInfo) {
		deleteCampaignItem = campaign;
		showDeleteConfirm = true;
	}

	async function executeDelete() {
		if (!deleteCampaignItem) return;
		deleting = true;
		try {
			await deleteCampaign({}, deleteCampaignItem.id);
			await loadData();
		} catch (err: any) {
			console.error('Failed to delete campaign:', err);
			error = err.message || 'Failed to delete campaign';
		} finally {
			deleting = false;
		}
	}

	function getStatusVariant(status: string): 'default' | 'info' | 'success' | 'warning' | 'error' {
		switch (status) {
			case 'draft': return 'default';
			case 'scheduled': return 'info';
			case 'sending': return 'warning';
			case 'sent': return 'success';
			case 'paused': return 'error';
			case 'cancelled': return 'error';
			default: return 'default';
		}
	}

	function formatDate(dateString?: string): string {
		if (!dateString) return '';
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		});
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return num.toString();
	}
</script>

<svelte:head>
	<title>Campaigns | Outlet</title>
</svelte:head>

<div class="p-6 max-w-6xl mx-auto">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<h1 class="text-2xl font-semibold text-text">Campaigns</h1>
			<p class="mt-1 text-sm text-text-muted">Send one-time email campaigns to your lists</p>
		</div>
		<Button type="primary" onclick={() => goto(`${basePath}/campaigns/new`)}>
			<Plus class="mr-2 h-4 w-4" />
			New Campaign
		</Button>
	</div>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if campaigns.length === 0}
		<EmptyState
			icon={Send}
			title="No campaigns yet"
			description="Create your first campaign to send emails to your subscribers."
		>
			<Button type="primary" onclick={() => goto(`${basePath}/campaigns/new`)}>
				<Plus class="mr-2 h-4 w-4" />
				Create Campaign
			</Button>
		</EmptyState>
	{:else}
		<!-- Status Tabs -->
		<div class="tab-nav mb-4">
			{#each tabs as tab}
				<button
					onclick={() => activeTab = tab.id}
					class="tab-nav-item"
					class:active={activeTab === tab.id}
				>
					{tab.label}
					{#if statusCounts()[tab.id]}
						<span class="tab-nav-count">{statusCounts()[tab.id]}</span>
					{/if}
				</button>
			{/each}
		</div>

		<!-- Search -->
		<div class="mb-4">
			<SearchInput
				bind:value={searchQuery}
				placeholder="Search campaigns..."
			/>
		</div>

		<!-- Campaigns Table -->
		<div class="data-table">
			<table class="w-full">
				<thead>
					<tr>
						<th class="text-left">Campaign</th>
						<th class="text-left">Status</th>
						<th class="text-right">Recipients</th>
						<th class="text-right">Opened</th>
						<th class="text-right">Clicked</th>
						<th class="text-right w-10"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredCampaigns() as campaign}
						<tr>
							<td>
								<a href="{basePath}/campaigns/{campaign.id}" class="block group">
									<div class="min-w-0">
										<span class="font-medium text-text group-hover:text-primary transition-colors block truncate">
											{campaign.name}
										</span>
										<span class="text-sm text-text-muted block truncate">
											{campaign.subject}
										</span>
									</div>
								</a>
							</td>
							<td>
								<div class="flex flex-col gap-1">
									<Badge variant={getStatusVariant(campaign.status)} size="sm">
										{campaign.status}
									</Badge>
									{#if campaign.scheduled_at && campaign.status === 'scheduled'}
										<span class="text-xs text-text-muted">
											{formatDate(campaign.scheduled_at)}
										</span>
									{:else if campaign.completed_at && campaign.status === 'sent'}
										<span class="text-xs text-text-muted">
											{formatDate(campaign.completed_at)}
										</span>
									{/if}
								</div>
							</td>
							<td class="text-right">
								<span class="font-medium text-text">{formatNumber(campaign.recipients_count)}</span>
							</td>
							<td class="text-right">
								{#if campaign.sent_count > 0}
									<div class="flex flex-col items-end">
										<span class="font-medium text-text">{formatNumber(campaign.opened_count)}</span>
										<span class="text-xs text-text-muted">
											{((campaign.opened_count / campaign.sent_count) * 100).toFixed(1)}%
										</span>
									</div>
								{:else}
									<span class="text-text-muted">-</span>
								{/if}
							</td>
							<td class="text-right">
								{#if campaign.sent_count > 0}
									<div class="flex flex-col items-end">
										<span class="font-medium text-text">{formatNumber(campaign.clicked_count)}</span>
										<span class="text-xs text-text-muted">
											{((campaign.clicked_count / campaign.sent_count) * 100).toFixed(1)}%
										</span>
									</div>
								{:else}
									<span class="text-text-muted">-</span>
								{/if}
							</td>
							<td class="text-right">
								<DropdownMenu
									trigger={{
										icon: MoreVertical,
										class: 'p-1.5 rounded-lg hover:bg-bg-secondary transition-colors text-text-muted hover:text-text'
									}}
									items={[
										{
											label: 'Edit',
											icon: Edit,
											onclick: () => goto(`${basePath}/campaigns/${campaign.id}`)
										},
										{
											label: 'View stats',
											icon: BarChart3,
											onclick: () => goto(`${basePath}/campaigns/${campaign.id}/stats`)
										},
										{
											label: 'Duplicate',
											icon: Copy,
											onclick: () => goto(`${basePath}/campaigns/new?duplicate=${campaign.id}`)
										},
										{ divider: true },
										{
											label: 'Delete',
											icon: Trash2,
											variant: 'danger',
											onclick: () => confirmDelete(campaign)
										}
									]}
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if searchQuery && filteredCampaigns().length === 0}
			<div class="text-center py-8 text-text-muted">
				No campaigns match "{searchQuery}"
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Campaign"
	description={`Delete "${deleteCampaignItem?.name || ''}"? This cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => showDeleteConfirm = false}
/>
