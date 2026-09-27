<script lang="ts">
	import { page } from '$app/stores';
	import { browser } from '$app/environment';
	import {
		LayoutDashboard,
		Users,
		Send,
		FileText,
		Settings,
		ChevronLeft,
		ChevronRight,
		Mail,
		Ban,
		Trash2
	} from '@lucide/svelte';

	interface Props {
		brandSlug: string;
		orgName?: string;
	}

	const { brandSlug, orgName = 'Outlet' }: Props = $props();

	const STORAGE_KEY = 'sidebar-collapsed';

	// Load collapsed state from localStorage
	let collapsed = $state(false);

	$effect(() => {
		if (browser) {
			const saved = localStorage.getItem(STORAGE_KEY);
			collapsed = saved === 'true';
		}
	});

	function toggleCollapse() {
		collapsed = !collapsed;
		if (browser) {
			localStorage.setItem(STORAGE_KEY, String(collapsed));
		}
	}

	let basePath = $derived(`/${brandSlug}`);

	// Navigation items - flat, email-focused
	// Sequences/Autoresponders live inside Lists now
	const navItems = [
		{ id: 'dashboard', label: 'Dashboard', href: '', icon: LayoutDashboard, exact: true },
		{ id: 'campaigns', label: 'Campaigns', href: '/campaigns', icon: Send },
		{ id: 'lists', label: 'Lists', href: '/lists', icon: Users },
		{ id: 'templates', label: 'Templates', href: '/templates', icon: FileText },
		{ id: 'housekeeping', label: 'Housekeeping', href: '/housekeeping', icon: Trash2 },
		{ id: 'blocklist', label: 'Blocklist', href: '/blocklist', icon: Ban },
	];

	const bottomItems = [
		{ id: 'settings', label: 'Settings', href: '/settings', icon: Settings },
	];

	function isActive(href: string, exact = false): boolean {
		const fullPath = `${basePath}${href}`;
		if (exact) {
			return $page.url.pathname === fullPath || $page.url.pathname === fullPath + '/';
		}
		return $page.url.pathname.startsWith(fullPath) && fullPath !== basePath;
	}
</script>

<aside class="sidebar" class:collapsed>
	<!-- Logo / Brand -->
	<div class="sidebar-header">
		{#if !collapsed}
			<a href={basePath} class="sidebar-brand">
				<Mail class="h-6 w-6 text-primary" />
				<span class="sidebar-brand-text">{orgName}</span>
			</a>
		{:else}
			<a href={basePath} class="sidebar-brand-icon">
				<Mail class="h-6 w-6 text-primary" />
			</a>
		{/if}
	</div>

	<!-- Main Navigation -->
	<nav class="sidebar-nav">
		{#each navItems as item}
			{@const Icon = item.icon}
			<a
				href="{basePath}{item.href}"
				class="sidebar-link"
				class:active={isActive(item.href, item.exact)}
				title={collapsed ? item.label : undefined}
			>
				<Icon class="sidebar-icon" />
				{#if !collapsed}
					<span class="sidebar-label">{item.label}</span>
				{/if}
			</a>
		{/each}
	</nav>

	<!-- Spacer -->
	<div class="flex-1"></div>

	<!-- Bottom Navigation -->
	<nav class="sidebar-nav sidebar-nav-bottom">
		{#each bottomItems as item}
			{@const Icon = item.icon}
			<a
				href="{basePath}{item.href}"
				class="sidebar-link"
				class:active={isActive(item.href)}
				title={collapsed ? item.label : undefined}
			>
				<Icon class="sidebar-icon" />
				{#if !collapsed}
					<span class="sidebar-label">{item.label}</span>
				{/if}
			</a>
		{/each}

		<!-- Collapse Toggle -->
		<button
			onclick={toggleCollapse}
			class="sidebar-collapse-btn"
			title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
		>
			{#if collapsed}
				<ChevronRight class="h-4 w-4" />
			{:else}
				<ChevronLeft class="h-4 w-4" />
				<span class="sidebar-label">Collapse</span>
			{/if}
		</button>
	</nav>
</aside>
