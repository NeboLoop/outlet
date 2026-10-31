<script lang="ts">
	import { page } from '$app/stores';

	const { children } = $props();

	// Get brandSlug from URL for building paths
	let brandSlug = $derived($page.params.brandSlug);

	let tabs = $derived([
		{ name: 'General', href: `/${brandSlug}/settings` },
		{ name: 'Email', href: `/${brandSlug}/settings/email` },
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

		<div class="border-b border-border">
			<nav class="-mb-px flex space-x-8">
				{#each tabs as tab}
					<a
						href={tab.href}
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
