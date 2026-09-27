<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import * as api from '$lib/api';
	import type { OrgInfo, SESQuotaResponse, OrgStats } from '$lib/api';
	import { Button, Card, Input, Badge, Modal, Alert, LoadingSpinner, Drawer } from '$lib/components/ui';
	import { Plus, Building2, ChevronRight, Cloud, Cpu, Copy, Check, Info, ArrowUpAZ, CalendarDays, Users, Mail, List } from '@lucide/svelte';

	let loading = $state(true);
	let organizations = $state<OrgInfo[]>([]);
	let orgStats = $state<Record<string, OrgStats>>({});
	let sortBy = $state<'name' | 'date'>('name');
	let sortedOrganizations = $derived(
		[...organizations].sort((a, b) => {
			if (sortBy === 'name') {
				return a.name.localeCompare(b.name);
			} else {
				return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
			}
		})
	);
	let error = $state('');

	// SES quota state
	let sesQuota = $state<SESQuotaResponse | null>(null);
	let sesLoading = $state(true);
	let sesError = $state('');

	// MCP state
	let mcpEndpoint = $state('');
	let copiedEndpoint = $state(false);

	// Mobile drawer state
	let showInfoDrawer = $state(false);

	// Check if platform is set up on mount
	onMount(async () => {
		try {
			const status = await api.getPublicSetupStatus();
			// Only require admin account - AWS/email can be configured later
			if (!status.has_admin) {
				goto('/setup');
				return;
			}
		} catch (err) {
			// If we can't check status, continue - the API might require auth
			console.error('Failed to check setup status:', err);
		}

		// Load SES quota and set MCP endpoint
		loadSESQuota();
		mcpEndpoint = `${window.location.origin}/mcp`;
	});

	function copyEndpoint() {
		navigator.clipboard.writeText(mcpEndpoint);
		copiedEndpoint = true;
		setTimeout(() => (copiedEndpoint = false), 2000);
	}

	async function loadSESQuota() {
		sesLoading = true;
		sesError = '';
		try {
			sesQuota = await api.getPlatformSESQuota();
		} catch (err: any) {
			console.error('Failed to load SES quota:', err);
			sesError = err.message || 'AWS not configured';
		} finally {
			sesLoading = false;
		}
	}

	function formatNumber(num: number): string {
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return Math.round(num).toString();
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}

	async function selectOrg(org: OrgInfo) {
		localStorage.setItem('currentOrgId', org.id);
		localStorage.setItem('currentOrgName', org.name);
		localStorage.setItem('currentOrgSlug', org.slug);

		// Check if org has email configured, unless user has skipped setup
		const skipped = localStorage.getItem(`orgSetupSkipped_${org.slug}`);
		if (!skipped && !org.from_email) {
			goto(`/${org.slug}/welcome`);
			return;
		}

		goto(`/${org.slug}`);
	}

	// Create modal
	let showCreateModal = $state(false);
	let creating = $state(false);
	let newOrgName = $state('');
	let newOrgSlug = $state('');
	let newFromName = $state('');
	let newFromEmail = $state('');
	let newReplyTo = $state('');

	$effect(() => {
		loadOrganizations();
	});

	async function loadOrganizations() {
		loading = true;
		error = '';

		try {
			const [orgsResponse, statsResponse] = await Promise.all([
				api.listOrganizations(),
				api.getAllOrganizationsStats()
			]);
			organizations = orgsResponse.organizations || [];
			orgStats = statsResponse.stats || {};
		} catch (err) {
			console.error('Failed to load organizations:', err);
			error = 'Failed to load organizations';
		} finally {
			loading = false;
		}
	}

	function openCreateModal() {
		newOrgName = '';
		newOrgSlug = '';
		newFromName = '';
		newFromEmail = '';
		newReplyTo = '';
		showCreateModal = true;
	}

	function closeCreateModal() {
		showCreateModal = false;
	}

	function generateSlug(name: string): string {
		return name
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	function handleNameInput() {
		newOrgSlug = generateSlug(newOrgName);
	}

	async function submitCreate() {
		if (!newOrgName || !newOrgSlug || !newFromEmail) return;

		creating = true;
		try {
			await api.createOrganization({
				name: newOrgName,
				slug: newOrgSlug,
				from_name: newFromName,
				from_email: newFromEmail,
				reply_to: newReplyTo || newFromEmail
			});
			closeCreateModal();
			await loadOrganizations();
		} catch (err: any) {
			console.error('Failed to create organization:', err);
			error = err.message || 'Failed to create organization';
		} finally {
			creating = false;
		}
	}
</script>

<svelte:head>
	<title>Brands - Outlet</title>
</svelte:head>

<div class="py-6">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
		<div class="flex gap-8">
			<!-- Main Content -->
			<div class="flex-1 min-w-0">
				<div class="mb-6 flex justify-between items-center">
					<h1 class="text-2xl font-semibold text-text">Brands</h1>
					<div class="flex items-center gap-2">
						<Button type="secondary" onclick={() => showInfoDrawer = true} class="md:hidden">
							<Info class="h-4 w-4" />
						</Button>
						<Button type="primary" onclick={openCreateModal}>
							<Plus class="mr-2 h-4 w-4" />
							New Brand
						</Button>
					</div>
				</div>

				{#if error}
					<div class="mb-6">
						<Alert type="error" title="Error">
							<p>{error}</p>
						</Alert>
					</div>
				{/if}

				{#if loading}
					<div class="flex justify-center py-12">
						<LoadingSpinner size="large" />
					</div>
				{:else if organizations.length === 0}
					<Card class="p-6">
						<h2 class="text-lg font-semibold text-text mb-4">Get started</h2>
						<div class="space-y-4 text-sm text-text-secondary">
							<p>
								Create a brand to organize your email operations. Each brand has its own lists,
								templates, and sending configuration.
							</p>
							<p>
								You might use one brand per project, per client, or per department depending on
								how you want to separate your email workflows.
							</p>
						</div>
						<div class="mt-6">
							<Button type="primary" onclick={openCreateModal}>
								<Plus class="mr-2 h-4 w-4" />
								Create brand
							</Button>
						</div>
					</Card>
				{:else}
					<!-- Sort toggle -->
					<div class="mb-4 flex items-center gap-2">
						<span class="text-xs text-text-muted">Sort by:</span>
						<div class="flex rounded-lg border border-border overflow-hidden">
							<button
								onclick={() => sortBy = 'name'}
								class="flex items-center gap-1.5 px-3 py-1.5 text-xs transition-colors {sortBy === 'name' ? 'bg-primary text-white' : 'bg-bg hover:bg-bg-secondary text-text-muted'}"
							>
								<ArrowUpAZ class="h-3.5 w-3.5" />
								Name
							</button>
							<button
								onclick={() => sortBy = 'date'}
								class="flex items-center gap-1.5 px-3 py-1.5 text-xs transition-colors {sortBy === 'date' ? 'bg-primary text-white' : 'bg-bg hover:bg-bg-secondary text-text-muted'}"
							>
								<CalendarDays class="h-3.5 w-3.5" />
								Newest
							</button>
						</div>
					</div>

					<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
						{#each sortedOrganizations as org}
							{@const stats = orgStats[org.id]}
							<button
								onclick={() => selectOrg(org)}
								class="text-left bg-bg border border-border rounded-lg p-4 hover:border-primary/50 hover:bg-bg-secondary transition-all group"
							>
								<div class="flex items-center gap-3 mb-3">
									<div class="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
										<span class="text-primary font-semibold">{org.name[0].toUpperCase()}</span>
									</div>
									<div class="min-w-0 flex-1">
										<span class="font-medium text-text group-hover:text-primary transition-colors block truncate">{org.name}</span>
										{#if org.from_email}
											<p class="text-xs text-text-muted mt-0.5 truncate">{org.from_email}</p>
										{:else}
											<p class="text-xs text-warning mt-0.5">Email not configured</p>
										{/if}
									</div>
									<ChevronRight class="w-5 h-5 text-text-muted group-hover:text-primary transition-colors flex-shrink-0" />
								</div>
								{#if stats}
									<div class="flex items-center gap-4 text-xs text-text-muted border-t border-border pt-3">
										<span class="flex items-center gap-1" title="Contacts">
											<Users class="w-3.5 h-3.5" />
											{formatNumber(stats.total_contacts)}
										</span>
										<span class="flex items-center gap-1" title="Emails sent (30d)">
											<Mail class="w-3.5 h-3.5" />
											{formatNumber(stats.emails_sent_30d)}
										</span>
										<span class="flex items-center gap-1" title="Lists">
											<List class="w-3.5 h-3.5" />
											{stats.list_count}
										</span>
									</div>
								{/if}
							</button>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Right Sidebar -->
			<aside class="hidden md:block w-56 flex-shrink-0">
				<div class="sticky top-24 space-y-6">
					<!-- Brands count -->
					<div>
						<h3 class="text-sm font-medium text-text mb-3">Overview</h3>
						<div class="space-y-2 text-sm">
							<div class="flex items-center justify-between">
								<span class="text-text-muted">Brands</span>
								<span class="font-medium text-text">{organizations.length}</span>
							</div>
						</div>
					</div>

					<!-- SES Quota -->
					<div>
						<div class="flex items-center gap-2 mb-3">
							<Cloud class="h-4 w-4 text-text-muted" />
							<h3 class="text-sm font-medium text-text">AWS SES</h3>
						</div>
						{#if sesLoading}
							<div class="text-sm text-text-muted">Loading...</div>
						{:else if sesError}
							<div class="text-xs text-text-muted">{sesError}</div>
						{:else if sesQuota}
							<div class="space-y-2 text-sm">
								<div class="flex items-center justify-between">
									<span class="text-text-muted">Region</span>
									<span class="font-medium text-text">{sesQuota.region}</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-text-muted">24hr Limit</span>
									<span class="font-medium text-text">{formatNumber(sesQuota.max_24_hour_send)}</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-text-muted">Sent Today</span>
									<span class="font-medium text-text">{formatNumber(sesQuota.sent_last_24_hours)}</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-text-muted">Remaining</span>
									<span class="font-medium text-text">{formatNumber(sesQuota.remaining_quota)}</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-text-muted">Rate</span>
									<span class="font-medium text-text">{sesQuota.max_send_rate}/sec</span>
								</div>
								{#if sesQuota.timezone}
									<div class="flex items-center justify-between">
										<span class="text-text-muted">Timezone</span>
										<span class="font-medium text-text">{sesQuota.timezone}</span>
									</div>
								{/if}
							</div>
						{/if}
					</div>

					<!-- MCP Connection -->
					<div>
						<div class="flex items-center gap-2 mb-3">
							<Cpu class="h-4 w-4 text-text-muted" />
							<h3 class="text-sm font-medium text-text">MCP Server</h3>
						</div>
						<div class="space-y-2 text-sm">
							<div class="flex items-center justify-between">
								<span class="text-text-muted">Name</span>
								<span class="font-medium text-text">Outlet</span>
							</div>
							<div>
								<span class="text-text-muted block mb-1">URL</span>
								<div class="flex items-center gap-1">
									<code class="text-xs bg-bg-secondary px-1.5 py-0.5 rounded font-mono text-text truncate flex-1">{mcpEndpoint}</code>
									<button
										onclick={copyEndpoint}
										class="p-1 hover:bg-bg-secondary rounded transition-colors"
										title="Copy URL"
									>
										{#if copiedEndpoint}
											<Check class="h-3.5 w-3.5 text-success" />
										{:else}
											<Copy class="h-3.5 w-3.5 text-text-muted" />
										{/if}
									</button>
								</div>
							</div>
						</div>
					</div>
				</div>
			</aside>
		</div>
	</div>
</div>

<!-- Create Modal -->
<Modal bind:show={showCreateModal} title="New Brand">
	<div class="space-y-4">
		<div>
			<label for="org-name" class="form-label">Name</label>
			<Input
				id="org-name"
				type="text"
				bind:value={newOrgName}
				oninput={handleNameInput}
				placeholder="Acme Corp"
			/>
		</div>

		<div class="border-t border-border pt-4">
			<h4 class="text-sm font-medium text-text mb-3">Email Settings</h4>
			<div class="space-y-3">
				<div>
					<label for="from-name" class="form-label">From Name</label>
					<Input
						id="from-name"
						type="text"
						bind:value={newFromName}
						placeholder="Acme Corp"
					/>
				</div>
				<div>
					<label for="from-email" class="form-label">From Email</label>
					<Input
						id="from-email"
						type="email"
						bind:value={newFromEmail}
						placeholder="hello@acme.com"
					/>
				</div>
				<div>
					<label for="reply-to" class="form-label">Reply To (optional)</label>
					<Input
						id="reply-to"
						type="email"
						bind:value={newReplyTo}
						placeholder="support@acme.com"
					/>
					<p class="mt-1 text-xs text-text-muted">Defaults to From Email if empty</p>
				</div>
			</div>
		</div>
	</div>

	{#snippet footer()}
		<div class="flex justify-end gap-3">
			<Button type="secondary" onclick={closeCreateModal} disabled={creating}>
				Cancel
			</Button>
			<Button type="primary" onclick={submitCreate} disabled={!newOrgName || !newOrgSlug || !newFromEmail || creating}>
				{creating ? 'Creating...' : 'Create'}
			</Button>
		</div>
	{/snippet}
</Modal>

<!-- Mobile Info Drawer -->
<Drawer bind:open={showInfoDrawer} position="right" title="Platform Info">
	<div class="space-y-6">
		<!-- Brands count -->
		<div>
			<h3 class="text-sm font-medium text-text mb-3">Overview</h3>
			<div class="space-y-2 text-sm">
				<div class="flex items-center justify-between">
					<span class="text-text-muted">Brands</span>
					<span class="font-medium text-text">{organizations.length}</span>
				</div>
			</div>
		</div>

		<!-- SES Quota -->
		<div>
			<div class="flex items-center gap-2 mb-3">
				<Cloud class="h-4 w-4 text-text-muted" />
				<h3 class="text-sm font-medium text-text">AWS SES</h3>
			</div>
			{#if sesLoading}
				<div class="text-sm text-text-muted">Loading...</div>
			{:else if sesError}
				<div class="text-xs text-text-muted">{sesError}</div>
			{:else if sesQuota}
				<div class="space-y-2 text-sm">
					<div class="flex items-center justify-between">
						<span class="text-text-muted">Region</span>
						<span class="font-medium text-text">{sesQuota.region}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-text-muted">24hr Limit</span>
						<span class="font-medium text-text">{formatNumber(sesQuota.max_24_hour_send)}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-text-muted">Sent Today</span>
						<span class="font-medium text-text">{formatNumber(sesQuota.sent_last_24_hours)}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-text-muted">Remaining</span>
						<span class="font-medium text-text">{formatNumber(sesQuota.remaining_quota)}</span>
					</div>
					<div class="flex items-center justify-between">
						<span class="text-text-muted">Rate</span>
						<span class="font-medium text-text">{sesQuota.max_send_rate}/sec</span>
					</div>
					{#if sesQuota.timezone}
						<div class="flex items-center justify-between">
							<span class="text-text-muted">Timezone</span>
							<span class="font-medium text-text">{sesQuota.timezone}</span>
						</div>
					{/if}
				</div>
			{/if}
		</div>

		<!-- MCP Connection -->
		<div>
			<div class="flex items-center gap-2 mb-3">
				<Cpu class="h-4 w-4 text-text-muted" />
				<h3 class="text-sm font-medium text-text">MCP Server</h3>
			</div>
			<div class="space-y-2 text-sm">
				<div class="flex items-center justify-between">
					<span class="text-text-muted">Name</span>
					<span class="font-medium text-text">Outlet</span>
				</div>
				<div>
					<span class="text-text-muted block mb-1">URL</span>
					<div class="flex items-center gap-1">
						<code class="text-xs bg-bg-secondary px-1.5 py-0.5 rounded font-mono text-text truncate flex-1">{mcpEndpoint}</code>
						<button
							onclick={copyEndpoint}
							class="p-1 hover:bg-bg-secondary rounded transition-colors"
							title="Copy URL"
						>
							{#if copiedEndpoint}
								<Check class="h-3.5 w-3.5 text-success" />
							{:else}
								<Copy class="h-3.5 w-3.5 text-text-muted" />
							{/if}
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</Drawer>
