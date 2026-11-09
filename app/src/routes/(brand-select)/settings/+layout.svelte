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

		<div class="border-b border-border">
			<nav class="tab-scroll -mb-px flex space-x-8" use:scrollActiveTab={$page.url.pathname}>
				{#each tabs as tab}
					<a
						href={tab.href}
						aria-current={isActive(tab.href) ? 'page' : undefined}
						class="whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium {isActive(tab.href)
							? 'border-primary text-primary'
							: 'border-transparent text-text-muted hover:border-border hover:text-text'}"
					>
						{tab.name}
					</a>
				{/each}
			</nav>
		</div>

		<div class="mt-6">
			{@render children()}
		</div>
	</div>
</div>
