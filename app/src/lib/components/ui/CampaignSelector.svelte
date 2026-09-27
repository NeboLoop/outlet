<!--
  Campaign Selector Modal
  Multi-select campaign picker with search and status filters
-->

<script lang="ts">
	import { Check, Hash, Mail, Eye, MousePointerClick } from '@lucide/svelte';
	import Modal from './Modal.svelte';
	import Button from './Button.svelte';
	import SearchInput from './SearchInput.svelte';
	import Select from './Select.svelte';
	import Checkbox from './Checkbox.svelte';
	import LoadingSpinner from './LoadingSpinner.svelte';
	import EmptyState from './EmptyState.svelte';
	import Alert from './Alert.svelte';
	import { listCampaigns, type CampaignInfo } from '$lib/api';

	interface Props {
		show: boolean;
		onselect?: (selectedIds: string[]) => void;
		oncancel?: () => void;
	}

	let {
		show = $bindable(false),
		onselect,
		oncancel
	}: Props = $props();

	// State
	let campaigns = $state<CampaignInfo[]>([]);
	let selectedIds = $state<Set<string>>(new Set());
	let searchQuery = $state('');
	let statusFilter = $state('all');
	let loading = $state(false);
	let error = $state<string | null>(null);

	// Filtered campaigns
	let filteredCampaigns = $derived.by(() => {
		let filtered = campaigns;

		// Status filter
		if (statusFilter !== 'all') {
			filtered = filtered.filter(c => c.status === statusFilter);
		}

		// Search filter
		if (searchQuery.trim()) {
			const query = searchQuery.toLowerCase();
			filtered = filtered.filter(c =>
				c.name.toLowerCase().includes(query) ||
				c.subject.toLowerCase().includes(query)
			);
		}

		// Sort by created_at (newest first)
		return filtered.sort((a, b) =>
			new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
		);
	});

	// All selected
	let allSelected = $derived(
		filteredCampaigns.length > 0 &&
		filteredCampaigns.every(c => selectedIds.has(c.id))
	);

	// Load campaigns when modal opens
	$effect(() => {
		if (show && campaigns.length === 0) {
			loadCampaigns();
		}
	});

	async function loadCampaigns() {
		try {
			loading = true;
			error = null;

			const response = await listCampaigns({});
			if (response.campaigns) {
				campaigns = response.campaigns;
			}
		} catch (err) {
			console.error('Failed to load campaigns:', err);
			error = 'Failed to load campaigns';
		} finally {
			loading = false;
		}
	}

	function toggleCampaign(campaignId: string) {
		if (selectedIds.has(campaignId)) {
			selectedIds.delete(campaignId);
		} else {
			selectedIds.add(campaignId);
		}
		selectedIds = selectedIds; // Trigger reactivity
	}

	function toggleAll() {
		if (allSelected) {
			// Deselect all visible
			filteredCampaigns.forEach(c => selectedIds.delete(c.id));
		} else {
			// Select all visible
			filteredCampaigns.forEach(c => selectedIds.add(c.id));
		}
		selectedIds = selectedIds; // Trigger reactivity
	}

	function handleApply() {
		if (onselect) {
			onselect(Array.from(selectedIds));
		}
		handleCancel();
	}

	function handleCancel() {
		show = false;
		if (oncancel) {
			oncancel();
		}
	}

	function formatDate(dateStr: string): string {
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function getOpenRate(campaign: CampaignInfo): string {
		if (campaign.sent_count === 0) return '0%';
		return ((campaign.opened_count / campaign.sent_count) * 100).toFixed(1) + '%';
	}

	function getClickRate(campaign: CampaignInfo): string {
		if (campaign.sent_count === 0) return '0%';
		return ((campaign.clicked_count / campaign.sent_count) * 100).toFixed(1) + '%';
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'draft': return 'bg-bg-secondary text-text-muted';
			case 'scheduled': return 'bg-blue-500/20 text-blue-600';
			case 'sending': return 'bg-yellow-500/20 text-yellow-600';
			case 'sent': return 'bg-green-500/20 text-green-600';
			case 'paused': return 'bg-yellow-500/20 text-yellow-600';
			case 'cancelled': return 'bg-red-500/20 text-red-600';
			default: return 'bg-bg-secondary text-text-muted';
		}
	}
</script>

<Modal bind:show title="Select Campaigns" size="xl">
	{#snippet children()}
		<!-- Filters -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
			<SearchInput
				bind:value={searchQuery}
				placeholder="Search campaigns..."
			/>

			<Select bind:value={statusFilter}>
				<option value="all">All Statuses</option>
				<option value="draft">Draft</option>
				<option value="scheduled">Scheduled</option>
				<option value="sending">Sending</option>
				<option value="sent">Sent</option>
				<option value="paused">Paused</option>
				<option value="cancelled">Cancelled</option>
			</Select>
		</div>

		{#if error}
			<Alert type="error" title="Error" class="mb-4">
				{error}
			</Alert>
		{/if}

		{#if loading}
			<div class="flex justify-center py-12">
				<LoadingSpinner size="large" />
			</div>
		{:else if filteredCampaigns.length === 0}
			<EmptyState
				icon={Mail}
				title="No campaigns found"
				message="Try adjusting your filters or search query"
			/>
		{:else}
			<!-- Select All -->
			<div class="flex items-center justify-between border-b border-border pb-4 mb-4">
				<div class="flex items-center gap-3">
					<Checkbox
						checked={allSelected}
						onchange={toggleAll}
						label="Select all ({filteredCampaigns.length} campaigns)"
					/>
				</div>
				<div class="text-sm text-text-muted">
					{selectedIds.size} selected
				</div>
			</div>

			<!-- Campaign List -->
			<div class="space-y-2 max-h-96 overflow-y-auto">
				{#each filteredCampaigns as campaign}
					<button
						type="button"
						onclick={() => toggleCampaign(campaign.id)}
						class="w-full flex items-start gap-4 p-4 rounded-lg border-2 transition-all
							{selectedIds.has(campaign.id)
								? 'border-indigo-500 bg-indigo-500/10'
								: 'border-border hover:border-text-muted hover:bg-bg-secondary'}"
					>
						<!-- Checkbox -->
						<div class="flex-shrink-0 mt-1">
							<div class="w-5 h-5 rounded border-2 flex items-center justify-center transition-all
								{selectedIds.has(campaign.id)
									? 'border-indigo-500 bg-indigo-500'
									: 'border-text-muted'}">
								{#if selectedIds.has(campaign.id)}
									<Check class="w-3 h-3 text-white" />
								{/if}
							</div>
						</div>

						<!-- Campaign Info -->
						<div class="flex-1 text-left">
							<div class="font-medium text-text mb-1">{campaign.name}</div>
							<div class="text-sm text-text-muted mb-2">{campaign.subject}</div>
							<div class="flex items-center gap-4 text-xs text-text-muted">
								<span class="flex items-center gap-1">
									<Hash class="w-3 h-3" />
									{campaign.id}
								</span>
								<span class="px-2 py-0.5 rounded-full {getStatusColor(campaign.status)}">
									{campaign.status}
								</span>
								<span class="text-text-muted">
									{formatDate(campaign.created_at)}
								</span>
							</div>
						</div>

						<!-- Metrics -->
						<div class="flex-shrink-0 text-right">
							<div class="text-sm font-semibold text-text mb-1">
								{campaign.sent_count} sent
							</div>
							<div class="flex items-center gap-3 text-xs">
								<span class="flex items-center gap-1 text-blue-600">
									<Eye class="w-3 h-3" />
									{getOpenRate(campaign)}
								</span>
								<span class="flex items-center gap-1 text-green-600">
									<MousePointerClick class="w-3 h-3" />
									{getClickRate(campaign)}
								</span>
							</div>
						</div>
					</button>
				{/each}
			</div>
		{/if}
	{/snippet}

	{#snippet footer()}
		<div class="flex items-center justify-between">
			<div class="text-sm text-text-muted">
				{selectedIds.size} campaign{selectedIds.size !== 1 ? 's' : ''} selected
			</div>
			<div class="flex gap-3">
				<Button type="secondary" onclick={handleCancel}>
					Cancel
				</Button>
				<Button type="primary" onclick={handleApply} disabled={selectedIds.size === 0}>
					<Check class="w-4 h-4 mr-2" />
					Apply Selection
				</Button>
			</div>
		</div>
	{/snippet}
</Modal>
