<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { ArrowLeft } from '@lucide/svelte';
	import CampaignForm from '../../CampaignForm.svelte';

	let base = $derived(`/${$page.params.brandSlug}/outbound`);
</script>

<a href="{base}/campaigns" class="inline-flex items-center text-sm text-text-muted hover:text-text mb-4">
	<ArrowLeft class="mr-1 h-4 w-4" />
	Campaigns
</a>

<h2 class="text-xl font-semibold text-text mb-4">New campaign</h2>

<CampaignForm oncreated={(id) => goto(`${base}/campaigns/${id}`)} />
