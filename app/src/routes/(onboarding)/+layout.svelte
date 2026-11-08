<script lang="ts">
	import { page } from '$app/stores';
	import { BrandMark } from '$lib/components/ui';
	import HelpLink from '$lib/components/ui/HelpLink.svelte';

	let { children } = $props();

	// The setup wizard runs before anyone can sign in, and Help needs a sign-in
	let showHelp = $derived(!$page.url.pathname.startsWith('/setup'));
</script>

<div class="auth-shell auth-shell-wide">
	<main class="w-full max-w-5xl">
		<div class="auth-topbar">
			<BrandMark />
			{#if showHelp}
				<HelpLink />
			{/if}
		</div>
		{@render children()}
	</main>
</div>
