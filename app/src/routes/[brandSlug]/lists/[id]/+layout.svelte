<script lang="ts">
	import { page } from '$app/stores';
	import { getList, type ListInfo } from '$lib/api';
	import { LoadingSpinner, Alert, Button, Badge, Drawer } from '$lib/components/ui';
	import { ArrowLeft, Users, Workflow, ListPlus, Code, Settings, Menu } from '@lucide/svelte';
	import { setListContext, type ListContext } from './listContext';

	let { children } = $props();

	// Get list ID and org slug from URL
	let listId = $derived($page.params.id!);
	let basePath = $derived(`/${$page.params.brandSlug}`);
	let currentPath = $derived($page.url.pathname);

	// State
	let loading = $state(true);
	let list = $state<ListInfo | null>(null);
	let error = $state('');
	let mobileMenuOpen = $state(false);

	// Create context object with reactive getters
	const listContext: ListContext = {
		get list() { return list; },
		get listId() { return listId; },
		get basePath() { return basePath; },
		reload: loadList
	};

	// Set context synchronously during initialization
	setListContext(listContext);

	// Determine active tab from current path
	let activeTab = $derived(() => {
		if (currentPath.endsWith('/autoresponders') || currentPath.includes('/autoresponders/')) return 'autoresponders';
		if (currentPath.endsWith('/custom-fields')) return 'custom-fields';
		if (currentPath.endsWith('/subscribe-form')) return 'subscribe-form';
		if (currentPath.endsWith('/settings')) return 'settings';
		return 'subscribers'; // default
	});

	$effect(() => {
		loadList();
	});

	async function loadList() {
		loading = true;
		error = '';
		try {
			list = await getList({}, listId);
		} catch (err) {
			console.error('Failed to load list:', err);
			error = 'Failed to load list';
		} finally {
			loading = false;
		}
	}

	const tabs = [
		{ id: 'subscribers', label: 'Subscribers', icon: Users, href: '' },
		{ id: 'autoresponders', label: 'Autoresponders', icon: Workflow, href: '/autoresponders' },
		{ id: 'custom-fields', label: 'Custom Fields', icon: ListPlus, href: '/custom-fields' },
		{ id: 'subscribe-form', label: 'Subscribe Form', icon: Code, href: '/subscribe-form' },
		{ id: 'settings', label: 'Settings', icon: Settings, href: '/settings' }
	];
</script>

<svelte:head>
	<title>{list?.name || 'List'} | Outlet</title>
</svelte:head>

<div class="p-6">
	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if !list}
		<Alert type="error" title="List not found">
			<p>The requested list could not be found.</p>
			<Button type="primary" class="mt-3" onclick={() => window.location.href = `${basePath}/lists`}>
				Back to Lists
			</Button>
		</Alert>
	{:else}
		<!-- Header -->
		<div class="flex items-center gap-4 mb-6 max-w-5xl">
			<a href="{basePath}/lists" class="p-2 rounded-lg hover:bg-bg-secondary transition-colors text-text-muted hover:text-text">
				<ArrowLeft class="h-5 w-5" />
			</a>
			<div class="flex-1 min-w-0">
				<div class="flex items-center gap-3">
					<h1 class="text-xl sm:text-2xl font-semibold text-text truncate">{list.name}</h1>
					{#if list.double_optin}
						<Badge variant="info" size="sm" class="hidden sm:inline-flex">Double opt-in</Badge>
					{/if}
				</div>
				{#if list.description}
					<p class="mt-1 text-sm text-text-muted hidden sm:block">{list.description}</p>
				{/if}
			</div>
			<div class="text-right flex-shrink-0">
				<div class="text-xl sm:text-2xl font-semibold text-text">{list.subscriber_count || 0}</div>
				<div class="text-xs sm:text-sm text-text-muted">subscribers</div>
			</div>
			<!-- Mobile menu button -->
			<button
				type="button"
				class="lg:hidden p-2 rounded-lg hover:bg-bg-secondary transition-colors text-text-muted hover:text-text"
				onclick={() => mobileMenuOpen = true}
			>
				<Menu class="h-5 w-5" />
			</button>
		</div>

		<!-- Main Layout: Content + Right Sidebar -->
		<div class="flex gap-6">
			<!-- Main Content Area -->
			<div class="flex-1 min-w-0 lg:max-w-3xl overflow-hidden">
				{@render children?.()}
			</div>

			<!-- Right Sidebar Navigation (Desktop) -->
			<div class="w-56 flex-shrink-0 hidden lg:block">
				<div class="sticky top-6">
					<nav class="space-y-1">
						{#each tabs as tab}
							<a
								href="{basePath}/lists/{listId}{tab.href}"
								class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors {activeTab() === tab.id
									? 'bg-primary/10 text-primary'
									: 'text-text-muted hover:bg-bg-secondary hover:text-text'}"
							>
								<tab.icon class="h-4 w-4 flex-shrink-0" />
								{tab.label}
							</a>
						{/each}
					</nav>

					<!-- Quick Stats -->
					<div class="mt-6 pt-6 border-t border-border">
						<h3 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Quick Stats</h3>
						<div class="space-y-2 text-sm">
							<div class="flex justify-between">
								<span class="text-text-muted">Active</span>
								<span class="font-medium text-text">{list.subscriber_count || 0}</span>
							</div>
							<div class="flex justify-between">
								<span class="text-text-muted">Created</span>
								<span class="font-medium text-text">{list.created_at ? new Date(list.created_at).toLocaleDateString() : '-'}</span>
							</div>
						</div>
					</div>

					<!-- Tags Section (placeholder for future) -->
					<div class="mt-6 pt-6 border-t border-border">
						<h3 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Tags</h3>
						<p class="text-xs text-text-muted italic">No tags yet</p>
						<!-- Future: Add tag management here -->
					</div>
				</div>
			</div>
		</div>

		<!-- Mobile Navigation Drawer -->
		<Drawer bind:open={mobileMenuOpen} position="right" title={list.name}>
			<nav class="space-y-1">
				{#each tabs as tab}
					<a
						href="{basePath}/lists/{listId}{tab.href}"
						class="flex items-center gap-3 px-3 py-3 rounded-lg text-sm font-medium transition-colors {activeTab() === tab.id
							? 'bg-primary/10 text-primary'
							: 'text-text-muted hover:bg-bg-secondary hover:text-text'}"
						onclick={() => mobileMenuOpen = false}
					>
						<tab.icon class="h-5 w-5 flex-shrink-0" />
						{tab.label}
					</a>
				{/each}
			</nav>

			<!-- Quick Stats -->
			<div class="mt-6 pt-6 border-t border-border">
				<h3 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Quick Stats</h3>
				<div class="space-y-2 text-sm">
					<div class="flex justify-between">
						<span class="text-text-muted">Active</span>
						<span class="font-medium text-text">{list.subscriber_count || 0}</span>
					</div>
					<div class="flex justify-between">
						<span class="text-text-muted">Created</span>
						<span class="font-medium text-text">{list.created_at ? new Date(list.created_at).toLocaleDateString() : '-'}</span>
					</div>
				</div>
			</div>

			<!-- Tags Section -->
			<div class="mt-6 pt-6 border-t border-border">
				<h3 class="text-xs font-semibold text-text-muted uppercase tracking-wider mb-3">Tags</h3>
				<p class="text-xs text-text-muted italic">No tags yet</p>
			</div>
		</Drawer>
	{/if}
</div>
