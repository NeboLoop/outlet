<script lang="ts">
	import { page } from '$app/stores';
	import { scrollActiveTab } from '$lib/actions/scrollActiveTab';

	const { children } = $props();

	const tabs = [
		{ name: 'MCP', href: '/settings/mcp' },
		{ name: 'API', href: '/settings/api' },
		{ name: 'Integrations', href: '/settings/email' },
		{ name: 'Users', href: '/settings/users' },
		{ name: 'Brands', href: '/settings/brands' },
		{ name: 'Backup', href: '/settings/backup' },
		{ name: 'Updates', href: '/settings/updates' }
	];

	function isActive(href: string): boolean {
		return $page.url.pathname === href;
	}
</script>

<div class="py-6">
	<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
		<div class="mb-6">
			<h1 class="text-2xl font-semibold text-text">Settings</h1>
			<p class="mt-1 text-sm text-text-muted">Manage your account settings</p>
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
