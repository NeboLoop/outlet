<script lang="ts">
	import { guides, sections, searchGuides, snippet } from '$lib/help/guides';

	let query = $state('');
	let results = $derived(searchGuides(query));
	let searching = $derived(query.trim() !== '');
</script>

<svelte:head>
	<title>Help | Outlet</title>
</svelte:head>

<div class="help-page">
	<h1 class="text-2xl font-semibold text-text">Help</h1>
	<p class="mt-1 text-sm text-text-muted">
		Step-by-step guides for setting up and running Outlet.
	</p>

	<input
		type="search"
		class="help-search mt-6"
		placeholder="Search the guides"
		aria-label="Search the guides"
		bind:value={query}
	/>

	{#if searching}
		<p class="help-section-title">
			{results.length === 1 ? '1 guide' : `${results.length} guides`}
		</p>
		{#if results.length === 0}
			<p class="text-sm text-text-muted">
				No guide mentions that. Try fewer or simpler words, like "bounce" or "DNS".
			</p>
		{/if}
		<div class="space-y-3">
			{#each results as guide (guide.slug)}
				<a href="/help/{guide.slug}" class="help-card">
					<p class="help-card-title">{guide.title}</p>
					<p class="help-card-summary">{snippet(guide, query)}</p>
				</a>
			{/each}
		</div>
	{:else}
		{#each sections as section}
			<p class="help-section-title">{section}</p>
			<div class="grid gap-3 sm:grid-cols-2">
				{#each guides.filter((guide) => guide.section === section) as guide (guide.slug)}
					<a href="/help/{guide.slug}" class="help-card">
						<p class="help-card-title">{guide.title}</p>
						<p class="help-card-summary">{guide.summary}</p>
					</a>
				{/each}
			</div>
		{/each}
	{/if}
</div>
