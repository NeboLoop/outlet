<script lang="ts">
	import { page } from '$app/stores';
	import { getOutboundOverview, type OutboundCampaignResults } from '$lib/api';
	import { Button, Alert, LoadingSpinner, EmptyState } from '$lib/components/ui';
	import { Plus, Send } from '@lucide/svelte';
	import CampaignTable from '../CampaignTable.svelte';
	import { errorMessage } from '../shared';

	let loading = $state(true);
	let error = $state('');
	let campaigns = $state<OutboundCampaignResults[]>([]);

	let base = $derived(`/${$page.params.brandSlug}/outbound`);

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			const overview = await getOutboundOverview();
			campaigns = overview.campaigns || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load campaigns');
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex items-center justify-between gap-4 mb-4">
	<h2 class="text-lg font-semibold text-text">Campaigns</h2>
	<Button type="primary" href="{base}/campaigns/new">
		<Plus class="mr-2 h-4 w-4" />
		New campaign
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
	<EmptyState icon={Send} title="No campaigns yet" description="Create one here, or let the AI start one over MCP." />
{:else}
	<CampaignTable {campaigns} />
{/if}
