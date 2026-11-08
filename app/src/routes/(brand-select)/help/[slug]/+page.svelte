<script lang="ts">
	import { ChevronLeft } from '@lucide/svelte';
	import Markdown from '$lib/components/ui/Markdown.svelte';
	import { guides } from '$lib/help/guides';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	let related = $derived(
		guides.filter((guide) => guide.section === data.guide.section && guide.slug !== data.guide.slug)
	);
</script>

<svelte:head>
	<title>{data.guide.title} | Help | Outlet</title>
</svelte:head>

<div class="help-page">
	<a href="/help" class="help-back">
		<ChevronLeft class="h-4 w-4" />
		All guides
	</a>

	<article class="help-article">
		<Markdown content={data.guide.markdown} />
	</article>

	{#if related.length > 0}
		<div class="help-related">
			<p class="help-section-title">More in {data.guide.section}</p>
			<div class="grid gap-3 sm:grid-cols-2">
				{#each related as guide (guide.slug)}
					<a href="/help/{guide.slug}" class="help-card">
						<p class="help-card-title">{guide.title}</p>
						<p class="help-card-summary">{guide.summary}</p>
					</a>
				{/each}
			</div>
		</div>
	{/if}
</div>
