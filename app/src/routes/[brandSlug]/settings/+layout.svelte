<script lang="ts">
	import { page } from '$app/stores';
	import { scrollActiveTab } from '$lib/actions/scrollActiveTab';

	const { children } = $props();

	// Get brandSlug from URL for building paths
	let brandSlug = $derived($page.params.brandSlug);

	let tabs = $derived([
		{ name: 'General', href: `/${brandSlug}/settings` },
		{ name: 'Email', href: `/${brandSlug}/settings/email` },
		{ name: 'Sending', href: `/${brandSlug}/settings/sending` },
		{ name: 'SMTP', href: `/${brandSlug}/settings/smtp` },
		{ name: 'Webhooks', href: `/${brandSlug}/settings/webhooks` },
		{ name: 'People', href: `/${brandSlug}/settings/people` },
		{ name: 'Privacy', href: `/${brandSlug}/settings/privacy` }
	]);

	function isActive(href: string): boolean {
		return $page.url.pathname === href;
	}
</script>

<div class="py-6">
	<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-text">Brand Settings</h1>
			<p class="mt-1 text-sm text-text-muted">Manage your brand configuration</p>
		</div>

		<nav class="tab-nav" use:scrollActiveTab={$page.url.pathname}>
			{#each tabs as tab}
				<a
					href={tab.href}
					aria-current={isActive(tab.href) ? 'page' : undefined}
					class="tab-nav-item"
					class:active={isActive(tab.href)}
				>
					{tab.name}
				</a>
			{/each}
		</nav>

		<div class="mt-6">
			{@render children()}
		</div>
	</div>
</div>
