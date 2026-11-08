<script lang="ts">
	import { page } from '$app/stores';
	import { CircleHelp } from '@lucide/svelte';
	import { guideForScreen } from '$lib/help/screens';

	interface Props {
		// A guide slug; left out, the link goes to the current screen's guide
		guide?: string;
		label?: string;
	}

	let { guide, label = 'Help' }: Props = $props();

	let slug = $derived(guide ?? guideForScreen($page.route.id));
	let href = $derived(slug ? `/help/${slug}` : '/help');
</script>

<a {href} class="help-link" title="Help for this screen" aria-label="Help for this screen">
	<CircleHelp class="h-4 w-4" />
	{#if label}
		<span>{label}</span>
	{/if}
</a>
