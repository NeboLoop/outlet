<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import Sidebar from '$lib/components/admin/Sidebar.svelte';
	import AuthGuard from '$lib/components/admin/AuthGuard.svelte';
	import { getOrganizationBySlug, listOrganizations, createOrganization, type OrgInfo } from '$lib/api';
	import { Modal, Input, Button, Alert } from '$lib/components/ui';
	import { getCurrentUser, logout } from '$lib/auth';
	import { Search, X, Menu, ChevronDown, Building2, Check, Plus } from '@lucide/svelte';

	interface Props {
		data: {
			brandSlug: string;
		};
		children: import('svelte').Snippet;
	}

	const { data, children }: Props = $props();

	let org = $state<OrgInfo | null>(null);
	let allOrgs = $state<OrgInfo[]>([]);
	let loading = $state(true);
	let error = $state('');
	let user = $state(getCurrentUser());

	// Mobile sidebar state
	let mobileMenuOpen = $state(false);

	// User menu state
	let showUserMenu = $state(false);

	// Brand switcher state
	let showBrandSwitcher = $state(false);

	// Create brand modal state
	let showCreateBrand = $state(false);
	let newBrandName = $state('');
	let newBrandSlug = $state('');
	let creatingBrand = $state(false);
	let createBrandError = $state('');

	$effect(() => {
		loadOrg();
	});

	// Close mobile menu on navigation
	$effect(() => {
		$page.url.pathname;
		mobileMenuOpen = false;
	});

	async function loadOrg() {
		loading = true;
		error = '';
		try {
			const [orgResult, orgsResult] = await Promise.all([
				getOrganizationBySlug({}, data.brandSlug),
				listOrganizations()
			]);
			org = orgResult;
			allOrgs = (orgsResult.organizations || []).sort((a, b) => a.name.localeCompare(b.name));
			localStorage.setItem('currentOrgId', org.id);
			localStorage.setItem('currentOrgName', org.name);
			localStorage.setItem('currentOrgSlug', org.slug);

			// Check if org needs setup (no from_email and user hasn't skipped)
			const setupSkipped = localStorage.getItem(`orgSetupSkipped_${org.slug}`);
			if (!org.from_email && !setupSkipped) {
				goto(`/${org.slug}/welcome`);
				return;
			}
		} catch (err: any) {
			console.error('Failed to load organization:', err);
			error = `Organization "${data.brandSlug}" not found`;
			goto('/');
		} finally {
			loading = false;
		}
	}

	function switchOrg() {
		localStorage.removeItem('currentOrgId');
		localStorage.removeItem('currentOrgName');
		localStorage.removeItem('currentOrgSlug');
		goto('/');
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

<svelte:head>
	<title>{org?.name || 'Admin'} | Outlet</title>
	<meta name="robots" content="noindex, nofollow" />
</svelte:head>

<svelte:window onclick={handleClickOutside} />

<AuthGuard>
	{#if loading}
		<div class="min-h-screen bg-base-50 flex items-center justify-center">
			<div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
		</div>
	{:else if error}
		<div class="min-h-screen bg-base-50 flex items-center justify-center">
			<div class="text-error">{error}</div>
		</div>
	{:else}
		<div class="admin-layout">
			<!-- Desktop Sidebar -->
			<div class="admin-sidebar hidden md:block">
				<Sidebar brandSlug={data.brandSlug} orgName={org?.name} />
			</div>

			<!-- Mobile Header -->
			<div class="fixed top-0 left-0 right-0 h-14 px-4 flex items-center justify-between z-40 border-b border-base-200 bg-white md:hidden">
				<button
					onclick={() => mobileMenuOpen = !mobileMenuOpen}
					class="p-2 rounded-lg transition-colors text-text-muted hover:bg-base-100"
				>
					{#if mobileMenuOpen}
						<X class="h-6 w-6" />
					{:else}
						<Menu class="h-6 w-6" />
					{/if}
				</button>
				<!-- Mobile Brand Switcher -->
				<div class="brand-switcher relative">
					<button
						onclick={(e) => { e.stopPropagation(); showUserMenu = false; showBrandSwitcher = !showBrandSwitcher; }}
						class="flex items-center gap-1 font-semibold text-text"
					>
						{org?.name || 'Outlet'}
						<ChevronDown class="h-4 w-4 text-text-muted" />
					</button>
					{#if showBrandSwitcher}
						<div class="absolute left-1/2 -translate-x-1/2 top-full mt-2 w-64 rounded-lg shadow-lg py-1 z-50 bg-white border border-base-200">
							<div class="px-3 py-2 border-b border-border">
								<p class="text-xs font-medium text-text-muted uppercase tracking-wide">Brands</p>
							</div>
							<div class="max-h-64 overflow-y-auto">
								{#each allOrgs as brand}
									<button
										onclick={() => navigateToOrg(brand.slug)}
										class="w-full px-3 py-2 flex items-center gap-2 text-left transition-colors hover:bg-base-100 {brand.id === org?.id ? 'bg-base-50' : ''}"
									>
										<span class="flex-1 text-sm font-medium text-text truncate">{brand.name}</span>
										{#if brand.id === org?.id}
											<Check class="h-4 w-4 text-primary flex-shrink-0" />
										{/if}
									</button>
								{/each}
							</div>
							<div class="border-t border-border mt-1 pt-1">
								<button
									onclick={openCreateBrand}
									class="w-full px-3 py-2 flex items-center gap-2 text-left transition-colors hover:bg-base-100 text-text-muted hover:text-text"
								>
									<Plus class="h-4 w-4" />
									<span class="text-sm">Add brand</span>
								</button>
							</div>
						</div>
					{/if}
				</div>
				<div class="user-menu-container relative">
					<button
						onclick={(e) => { e.stopPropagation(); showBrandSwitcher = false; showUserMenu = !showUserMenu; }}
						class="flex items-center gap-2 p-1 rounded-lg transition-colors hover:bg-base-100"
					>
						<div class="flex items-center justify-center h-8 w-8 rounded-full text-sm font-medium text-white bg-primary">
							{(user?.name || user?.email || 'U')[0].toUpperCase()}
						</div>
					</button>
					{#if showUserMenu}
						<div class="absolute right-0 top-full mt-2 w-56 rounded-lg shadow-lg py-1 z-50 bg-white border border-base-200">
							<div class="px-4 py-3 border-b border-border">
								<p class="text-sm font-medium text-text truncate">{user?.email}</p>
							</div>
							<a href="/{data.brandSlug}/settings" class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900" onclick={() => showUserMenu = false}>
								Settings
							</a>
							<div class="border-t border-border my-1"></div>
							<button onclick={switchOrg} class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900 w-full text-left">
								Switch Brand
							</button>
							<button onclick={handleLogout} class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900 w-full text-left">
								Sign out
							</button>
						</div>
					{/if}
				</div>
			</div>

			<!-- Mobile Sidebar Overlay -->
			{#if mobileMenuOpen}
				<div
					class="fixed inset-0 z-40 bg-black/50 md:hidden"
					onclick={() => mobileMenuOpen = false}
					onkeydown={(e) => e.key === 'Escape' && (mobileMenuOpen = false)}
					role="button"
					tabindex="-1"
				></div>
				<div class="fixed inset-y-0 left-0 z-50 w-[220px] md:hidden">
					<Sidebar brandSlug={data.brandSlug} orgName={org?.name} />
				</div>
			{/if}

			<!-- Main Content -->
			<div class="admin-main">
				<!-- Top Bar (Desktop) -->
				<header class="h-16 px-6 items-center justify-end border-b border-base-200 sticky top-0 z-20 bg-white hidden md:flex">
					<div class="flex items-center gap-4">
						<!-- Brand Switcher -->
						<div class="brand-switcher relative">
							<button
								onclick={(e) => { e.stopPropagation(); showUserMenu = false; showBrandSwitcher = !showBrandSwitcher; }}
								class="flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors hover:bg-base-100 group"
							>
								<span class="text-sm text-text-muted">Brand:</span>
								<span class="font-medium text-sm">{org?.name || 'Organization'}</span>
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
												class="w-full px-3 py-2 flex items-center gap-2 text-left transition-colors hover:bg-base-100 {brand.id === org?.id ? 'bg-base-50' : ''}"
											>
												<span class="flex-1 text-sm font-medium text-text truncate">{brand.name}</span>
												{#if brand.id === org?.id}
													<Check class="h-4 w-4 text-primary flex-shrink-0" />
												{/if}
											</button>
										{/each}
									</div>
									<div class="border-t border-border mt-1 pt-1">
										<button
											onclick={openCreateBrand}
											class="w-full px-3 py-2 flex items-center gap-2 text-left transition-colors hover:bg-base-100 text-text-muted hover:text-text"
										>
											<Plus class="h-4 w-4" />
											<span class="text-sm">Add brand</span>
										</button>
									</div>
								</div>
							{/if}
						</div>
						<!-- User Menu -->
						<div class="user-menu-container relative">
							<button
								onclick={(e) => { e.stopPropagation(); showBrandSwitcher = false; showUserMenu = !showUserMenu; }}
								class="flex items-center gap-2 p-1 rounded-lg transition-colors hover:bg-base-100"
							>
								<div class="flex items-center justify-center h-8 w-8 rounded-full text-sm font-medium text-white bg-primary">
									{(user?.name || user?.email || 'U')[0].toUpperCase()}
								</div>
								<span class="text-sm text-text-muted hidden lg:block">{user?.email}</span>
							</button>
							{#if showUserMenu}
								<div class="absolute right-0 top-full mt-2 w-56 rounded-lg shadow-lg py-1 z-50 bg-white border border-base-200">
									<div class="px-4 py-3 border-b border-border">
										<p class="text-sm font-medium text-text truncate">{user?.email}</p>
										<p class="text-xs text-text-muted capitalize">{user?.role || 'user'}</p>
									</div>
									<a href="/{data.brandSlug}/settings" class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900" onclick={() => showUserMenu = false}>
										Settings
									</a>
									<div class="border-t border-border my-1"></div>
									<button onclick={switchOrg} class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900 w-full text-left">
										Switch Brand
									</button>
									<button onclick={handleLogout} class="block px-4 py-2 text-sm transition-colors text-base-700 hover:bg-base-100 hover:text-base-900 w-full text-left">
										Sign out
									</button>
								</div>
							{/if}
						</div>
					</div>
				</header>

				<!-- Page Content -->
				<main class="flex-1">
					{@render children()}
				</main>
			</div>
		</div>
	{/if}
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
