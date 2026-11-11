<script lang="ts">
	import { page } from '$app/stores';
	import { scrollActiveTab } from '$lib/actions/scrollActiveTab';

	const { children } = $props();

	let base = $derived(`/${$page.params.brandSlug}/outbound`);

	let tabs = $derived([
		{ name: 'Overview', href: base, exact: true },
		{ name: 'Campaigns', href: `${base}/campaigns`, exact: false },
		{ name: 'Drafts', href: `${base}/drafts`, exact: false },
		{ name: 'Replies', href: `${base}/replies`, exact: false },
		{ name: 'Prospects', href: `${base}/prospects`, exact: false },
		{ name: 'Targeting', href: `${base}/targeting`, exact: false },
		{ name: 'Materials', href: `${base}/materials`, exact: false },
		{ name: 'Do not contact', href: `${base}/do-not-contact`, exact: false },
		{ name: 'Setup', href: `${base}/setup`, exact: false }
	]);

	function isActive(href: string, exact: boolean): boolean {
		const path = $page.url.pathname.replace(/\/$/, '');
		return exact ? path === href : path === href || path.startsWith(`${href}/`);
	}
</script>

<svelte:head>
	<title>Outbound | Outlet</title>
</svelte:head>

<div class="p-4 sm:p-6 max-w-5xl mx-auto">
	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-text">Outbound</h1>
		<p class="mt-1 text-sm text-text-muted">
			One-to-one outreach sent from your own mailboxes
		</p>
	</div>

	<nav class="tab-nav mb-6" use:scrollActiveTab={$page.url.pathname}>
		{#each tabs as tab}
			<a
				href={tab.href}
				class="tab-nav-item whitespace-nowrap"
				class:active={isActive(tab.href, tab.exact)}
				aria-current={isActive(tab.href, tab.exact) ? 'page' : undefined}
			>
				{tab.name}
			</a>
		{/each}
	</nav>

	{@render children()}
</div>
