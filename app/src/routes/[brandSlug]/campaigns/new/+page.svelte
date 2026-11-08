<script lang="ts">
	import { page } from '$app/stores';
	import { getCampaign, type CampaignInfo } from '$lib/api';
	import { Alert, LoadingSpinner } from '$lib/components/ui';
	import CampaignEditor from '$lib/components/campaigns/CampaignEditor.svelte';

	// ?duplicate=<id> starts the new draft from that campaign's content
	let duplicateId = $derived($page.url.searchParams.get('duplicate') ?? '');
	let copyOf = $state<CampaignInfo | undefined>(undefined);
	let loading = $state(false);
	let error = $state('');

	$effect(() => {
		if (!duplicateId) return;
		loading = true;
		error = '';
		getCampaign({}, duplicateId)
			.then((c) => (copyOf = c))
			.catch((err: Error) => (error = err.message || 'Failed to load the campaign to duplicate'))
			.finally(() => (loading = false));
	});
</script>

<svelte:head>
	<title>New Campaign | Outlet</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-12"><LoadingSpinner size="large" /></div>
{:else if error}
	<div class="p-6"><Alert type="error" title="Error"><p>{error}</p></Alert></div>
{:else if !duplicateId || copyOf}
	<CampaignEditor {copyOf} />
{/if}
