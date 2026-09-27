<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getCurrentUser, logout } from '$lib/auth';
	import AuthGuard from '$lib/components/admin/AuthGuard.svelte';
	import { listOrganizations, createOrganization, type OrgInfo } from '$lib/api';
	import { Modal, Input, Button, Alert } from '$lib/components/ui';
	import { ChevronDown, Check, Plus, Settings } from '@lucide/svelte';

	const { children } = $props();

	let user = $state(getCurrentUser());
	let showUserMenu = $state(false);
	let allOrgs = $state<OrgInfo[]>([]);
	let showBrandSwitcher = $state(false);

	// Create brand modal state
	let showCreateBrand = $state(false);
	let newBrandName = $state('');
	let newBrandSlug = $state('');
	let creatingBrand = $state(false);
	let createBrandError = $state('');

	$effect(() => {
		loadOrgs();
	});

	async function loadOrgs() {
		try {
			const result = await listOrganizations();
			allOrgs = result.organizations || [];
		} catch (err) {
			console.error('Failed to load organizations:', err);
		}
	}

	function handleLogout() {
		logout();
	}

	function handleClickOutside(event: MouseEvent) {
		const target = event.target as HTMLElement;
		if (!target.closest('.user-menu-container')) {
			showUserMenu = false;
		}
		if (!target.closest('.brand-switcher')) {
			showBrandSwitcher = false;
		}
	}

	function navigateToOrg(slug: string) {
		showBrandSwitcher = false;
		goto(`/${slug}`);
	}

	function generateSlug(name: string): string {
		return name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
	}

	function handleBrandNameInput() {
		newBrandSlug = generateSlug(newBrandName);
	}

	function openCreateBrand() {
		showBrandSwitcher = false;
		newBrandName = '';
		newBrandSlug = '';
		createBrandError = '';
		showCreateBrand = true;
	}

	async function handleCreateBrand() {
		if (!newBrandName.trim() || !newBrandSlug.trim()) return;
		creatingBrand = true;
		createBrandError = '';
		try {
			const newOrg = await createOrganization({
				name: newBrandName.trim(),
				slug: newBrandSlug.trim()
			});
			showCreateBrand = false;
			goto(`/${newOrg.slug}/welcome`);
		} catch (err: any) {
			createBrandError = err.message || 'Failed to create brand';
		} finally {
			creatingBrand = false;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<svelte:head>
	<title>Outlet</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<AuthGuard>
<div class="min-h-screen bg-base-50">
	<!-- Minimal navbar for brand selection / global settings -->
	<nav class="border-b border-base-200 bg-white">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="flex h-16 justify-between">
				<div class="flex items-center">
					<a href="/" class="text-xl font-bold text-primary">Outlet</a>
				</div>

				<div class="flex items-center space-x-4">
					<!-- Brand Switcher -->
					<div class="brand-switcher relative">
						<button
							onclick={(e) => { e.stopPropagation(); showUserMenu = false; showBrandSwitcher = !showBrandSwitcher; }}
							class="flex items-center gap-2 px-3 py-1.5 rounded-lg transition-colors hover:bg-base-100 group"
						>
							<span class="font-medium text-sm">Brands</span>
							<ChevronDown class="h-4 w-4 text-text-muted group-hover:text-text transition-colors" />
						</button>
						{#if showBrandSwitcher}
							<div class="absolute right-0 top-full mt-2 w-64 rounded-lg shadow-lg py-1 z-50 bg-white border border-base-200">
								<div class="px-3 py-2 border-b border-border">
									<p class="text-xs font-medium text-text-muted uppercase tracking-wide">Brands</p>
								</div>
								<div class="max-h-64 overflow-y-auto">
									{#each allOrgs as brand}
										<button
											onclick={() => navigateToOrg(brand.slug)}
											class="w-full px-3 py-2 flex items-center gap-3 text-left transition-colors hover:bg-base-100"
										>
											<span class="flex items-center justify-center h-8 w-8 rounded-md text-xs font-semibold text-white bg-primary flex-shrink-0">
												{brand.name[0].toUpperCase()}
											</span>
											<span class="flex-1 text-sm font-medium text-text truncate">{brand.name}</span>
										</button>
									{/each}
								</div>
								<div class="border-t border-border mt-1 pt-1">
									<button
										onclick={openCreateBrand}
										class="w-full px-3 py-2 flex items-center gap-3 text-left transition-colors hover:bg-base-100 text-text-muted hover:text-text"
									>
										<span class="flex items-center justify-center h-8 w-8 rounded-md border-2 border-dashed border-base-300 flex-shrink-0">
											<Plus class="h-4 w-4" />
										</span>
										<span class="text-sm">Add brand</span>
									</button>
								</div>
							</div>
						{/if}
					</div>

					<!-- User Menu -->
					<div class="user-menu-container relative ml-3">
						<div>
							<button
								onclick={(e) => { e.stopPropagation(); showBrandSwitcher = false; showUserMenu = !showUserMenu; }}
								class="flex items-center rounded-full text-sm transition-colors focus:outline-none"
								aria-expanded={showUserMenu}
								aria-haspopup="true"
							>
								<span class="sr-only">Open user menu</span>
								<div class="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-white">
									<span class="text-sm font-medium text-white">
										{(user?.name || user?.email || 'U')[0].toUpperCase()}
									</span>
								</div>
							</button>
						</div>

						{#if showUserMenu}
							<div class="absolute right-0 z-50 mt-2 w-56 origin-top-right rounded-lg py-1 bg-white border border-base-200 shadow-lg" role="menu" aria-orientation="vertical">
								<div class="px-4 py-3 border-b border-border">
									<p class="text-sm font-medium text-text truncate">{user?.name || user?.email}</p>
									<p class="text-xs text-text-muted capitalize">{user?.role}</p>
								</div>
								<a
									href="/settings"
									class="flex items-center gap-2 px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900"
									role="menuitem"
									onclick={() => showUserMenu = false}
								>
									<Settings class="h-4 w-4" />
									Settings
								</a>
								<div class="border-t border-border my-1"></div>
								<button
									onclick={handleLogout}
									class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900 w-full text-left"
									role="menuitem"
								>
									Sign out
								</button>
							</div>
						{/if}
					</div>
				</div>
			</div>
		</div>
	</nav>

	<main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
		{@render children()}
	</main>
</div>
</AuthGuard>

<!-- Create Brand Modal -->
<Modal bind:show={showCreateBrand} title="Create Brand">
	<div class="space-y-4">
		{#if createBrandError}
			<Alert type="error">{createBrandError}</Alert>
		{/if}
		<div>
			<label for="brand-name" class="block text-sm font-medium text-text mb-1">Brand Name</label>
			<Input
				id="brand-name"
				bind:value={newBrandName}
				oninput={handleBrandNameInput}
				placeholder="My Company"
			/>
		</div>
		<div>
			<label for="brand-slug" class="block text-sm font-medium text-text mb-1">URL Slug</label>
			<Input
				id="brand-slug"
				bind:value={newBrandSlug}
				placeholder="my-company"
			/>
			<p class="text-xs text-text-muted mt-1">This will be used in the URL: /{newBrandSlug || 'my-company'}</p>
		</div>
	</div>
	{#snippet footer()}
		<Button type="secondary" onclick={() => showCreateBrand = false} disabled={creatingBrand}>
			Cancel
		</Button>
		<Button type="primary" onclick={handleCreateBrand} disabled={!newBrandName.trim() || !newBrandSlug.trim() || creatingBrand}>
			{creatingBrand ? 'Creating...' : 'Create Brand'}
		</Button>
	{/snippet}
</Modal>
