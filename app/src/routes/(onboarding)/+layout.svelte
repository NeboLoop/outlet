<script lang="ts">
	import { page } from '$app/stores';
	import HelpLink from '$lib/components/ui/HelpLink.svelte';

	let { children } = $props();

	// The setup wizard runs before anyone can sign in, and Help needs a sign-in
	let showHelp = $derived(!$page.url.pathname.startsWith('/setup'));
</script>

<div class="min-h-screen bg-base-50 flex flex-col items-center justify-start pt-16 pb-12 px-4 sm:px-6 lg:px-8">
	<div class="w-full max-w-5xl">
		{#if showHelp}
			<div class="flex justify-end">
				<HelpLink />
			</div>
		{/if}
		{@render children()}
	</div>
</div>
