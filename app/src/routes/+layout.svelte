<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getPublicSetupStatus } from '$lib/api';
	import '$src/app.css';

	let { children } = $props();

	// Nothing renders, and so nothing calls the API, until we know the
	// platform is set up: a fresh install goes straight to /setup.
	let ready = $state(false);

	onMount(async () => {
		// Skip setup check if already on setup page
		if ($page.url.pathname.startsWith('/setup')) {
			ready = true;
			return;
		}

		try {
			const status = await getPublicSetupStatus();
			if (status.setup_required) {
				await goto('/setup');
			}
		} catch (err) {
			// If we can't tell, show the app; its own requests report the problem
			console.error('Failed to check setup status:', err);
		}
		ready = true;
	});
</script>

<svelte:head>
	<meta name="viewport" content="width=device-width, initial-scale=1" />
</svelte:head>

{#if ready}
	{@render children()}
{/if}
