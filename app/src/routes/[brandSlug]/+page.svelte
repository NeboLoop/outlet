<script lang="ts">
	import { page } from '$app/stores';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { Card, Badge, LoadingSpinner, EmptyState } from '$lib/components/ui';
	import { ChartCard, AreaChart } from '$lib/components/charts';
	import { getEmailDashboardStats, listLists, listCampaigns, type EmailDashboardStatsResponse, type ListInfo, type CampaignInfo } from '$lib/api';
	import { getCurrentUser } from '$lib/auth';
	import { TrendingUp, TrendingDown, Users, Mail, Send, MousePointerClick, AlertTriangle, UserMinus, ArrowRight, Plus, RefreshCw } from '@lucide/svelte';

	interface Props {
		data: {
			brandSlug?: string;
		};
	}

	const { data }: Props = $props();
	let basePath = $derived(data.brandSlug ? `/${data.brandSlug}` : '');
	let user = $state(getCurrentUser());

	let loading = $state(true);
	let emailStats = $state<EmailDashboardStatsResponse | null>(null);
	let lists = $state<ListInfo[]>([]);
	let recentCampaigns = $state<CampaignInfo[]>([]);
	let orgName = $state('');

	$effect(() => {
		loadDashboardData();
	});

	async function loadDashboardData() {
		loading = true;

		try {
			const orgId = browser ? localStorage.getItem('currentOrgId') : null;
			if (!orgId) {
				loading = false;
				return;
			}

			// Load email stats, lists, and recent campaigns in parallel
			const [statsResult, listsResult, campaignsResult] = await Promise.all([
				getEmailDashboardStats({}).catch(() => null),
				listLists().catch(() => ({ lists: [] })),
				listCampaigns({ limit: 5 }).catch(() => ({ campaigns: [] }))
			]);

			emailStats = statsResult;
			lists = listsResult?.lists || [];
			recentCampaigns = (campaignsResult?.campaigns || []).slice(0, 5);

			// Get org name from localStorage
			const storedOrgName = browser ? localStorage.getItem('currentOrgName') : null;
			orgName = storedOrgName || 'Your Organization';

		} catch (err) {
			console.error('Failed to load dashboard data:', err);
		} finally {
			loading = false;
		}
	}

	function formatNumber(num: number | null | undefined): string {
		if (num == null) return '0';
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return num.toString();
	}

	function formatPercent(num: number | null | undefined): string {
		if (num == null) return '0%';
		return num.toFixed(1) + '%';
	}

	function formatDate(dateString: string): string {
		if (!dateString) return 'N/A';
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
	}

	// Calculate total subscribers across all lists
	let totalSubscribers = $derived(lists.reduce((sum, list) => sum + (list.subscriber_count || 0), 0));

	// Check if user has any data yet
	let hasData = $derived(lists.length > 0 || (emailStats && emailStats.total_sent > 0));
</script>

<svelte:head>
	<title>Dashboard - {orgName || 'Admin'}</title>
</svelte:head>

<div class="p-6 space-y-6 max-w-6xl mx-auto">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-semibold text-text">Welcome back, {user?.name || user?.email?.split('@')[0] || 'there'}</h1>
			<p class="mt-1 text-sm text-text-muted">{orgName}</p>
		</div>
		<button
			onclick={loadDashboardData}
			disabled={loading}
			class="p-2 rounded-lg hover:bg-bg-secondary transition-colors"
			title="Refresh"
		>
			<RefreshCw class="h-5 w-5 text-text-muted {loading ? 'animate-spin' : ''}" />
		</button>
	</div>

	{#if loading}
		<div class="flex items-center justify-center h-64">
			<LoadingSpinner size="large" />
		</div>
	{:else if !hasData}
		<!-- Empty state for new users -->
		<Card class="text-center py-12">
			<Mail class="h-12 w-12 text-text-muted mx-auto mb-4" />
			<h2 class="text-xl font-semibold text-text mb-2">Get started with email</h2>
			<p class="text-text-muted mb-6 max-w-md mx-auto">
				Create your first email list to start collecting subscribers and sending campaigns.
			</p>
			<div class="flex justify-center gap-3">
				<a
					href="{basePath}/lists"
					class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-white font-medium hover:bg-primary/90 transition-colors"
				>
					<Plus class="h-4 w-4" />
					Create a List
				</a>
			</div>
		</Card>
	{:else}
		<!-- Email Metrics -->
		<div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
			<!-- Total Subscribers -->
			<a href="{basePath}/lists" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="text-xs font-medium text-text-muted uppercase tracking-wide">Subscribers</p>
							<p class="text-2xl font-bold text-text mt-1">{formatNumber(totalSubscribers)}</p>
							<p class="text-xs text-text-muted mt-1">{lists.length} list{lists.length !== 1 ? 's' : ''}</p>
						</div>
						<div class="p-2 bg-blue-100 rounded-lg">
							<Users class="h-5 w-5 text-blue-600" />
						</div>
					</div>
				</Card>
			</a>

			<!-- Emails Sent -->
			<a href="{basePath}/campaigns" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="text-xs font-medium text-text-muted uppercase tracking-wide">Emails Sent</p>
							<p class="text-2xl font-bold text-text mt-1">{formatNumber(emailStats?.total_sent)}</p>
							<p class="text-xs text-text-muted mt-1">All time</p>
						</div>
						<div class="p-2 bg-green-100 rounded-lg">
							<Send class="h-5 w-5 text-green-600" />
						</div>
					</div>
				</Card>
			</a>

			<!-- Open Rate -->
			<a href="{basePath}/campaigns" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="text-xs font-medium text-text-muted uppercase tracking-wide">Open Rate</p>
							<p class="text-2xl font-bold text-text mt-1">{formatPercent(emailStats?.open_rate)}</p>
							<p class="text-xs text-text-muted mt-1">{formatNumber(emailStats?.total_opened)} opened</p>
						</div>
						<div class="p-2 bg-purple-100 rounded-lg">
							<Mail class="h-5 w-5 text-purple-600" />
						</div>
					</div>
				</Card>
			</a>

			<!-- Click Rate -->
			<a href="{basePath}/campaigns" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="text-xs font-medium text-text-muted uppercase tracking-wide">Click Rate</p>
							<p class="text-2xl font-bold text-text mt-1">{formatPercent(emailStats?.click_rate)}</p>
							<p class="text-xs text-text-muted mt-1">{formatNumber(emailStats?.total_clicked)} clicked</p>
						</div>
						<div class="p-2 bg-cyan-100 rounded-lg">
							<MousePointerClick class="h-5 w-5 text-cyan-600" />
						</div>
					</div>
				</Card>
			</a>
		</div>

		<!-- Secondary Stats Row -->
		{#if emailStats && (emailStats.total_bounced > 0 || emailStats.total_complaints > 0 || emailStats.total_unsubscribed > 0)}
			<div class="grid grid-cols-3 gap-4">
				<Card class="flex items-center gap-3 p-4">
					<div class="p-2 bg-amber-100 rounded-lg">
						<AlertTriangle class="h-4 w-4 text-amber-600" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">{formatNumber(emailStats.total_bounced)}</p>
						<p class="text-xs text-text-muted">Bounced</p>
					</div>
				</Card>

				<Card class="flex items-center gap-3 p-4">
					<div class="p-2 bg-red-100 rounded-lg">
						<AlertTriangle class="h-4 w-4 text-red-600" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">{formatNumber(emailStats.total_complaints)}</p>
						<p class="text-xs text-text-muted">Complaints</p>
					</div>
				</Card>

				<Card class="flex items-center gap-3 p-4">
					<div class="p-2 bg-gray-100 rounded-lg">
						<UserMinus class="h-4 w-4 text-gray-600" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">{formatNumber(emailStats.total_unsubscribed)}</p>
						<p class="text-xs text-text-muted">Unsubscribed</p>
					</div>
				</Card>
			</div>
		{/if}

		<!-- Lists and Campaigns Row -->
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Email Lists -->
			<Card>
				<div class="flex items-center justify-between mb-4">
					<h2 class="text-lg font-semibold text-text">Email Lists</h2>
					<a href="{basePath}/lists" class="text-sm font-medium text-primary hover:underline flex items-center gap-1">
						View all
						<ArrowRight class="h-4 w-4" />
					</a>
				</div>
				{#if lists.length > 0}
					<div class="space-y-3">
						{#each lists.slice(0, 5) as list}
							<a href="{basePath}/lists/{list.id}" class="flex items-center justify-between py-2 hover:bg-bg-secondary -mx-2 px-2 rounded-lg transition-colors">
								<div class="min-w-0">
									<p class="text-sm font-medium text-text truncate">{list.name}</p>
									{#if list.description}
										<p class="text-xs text-text-muted truncate">{list.description}</p>
									{/if}
								</div>
								<div class="flex items-center gap-2 ml-4">
									<Badge variant="default" size="sm">
										{formatNumber(list.subscriber_count)} subscribers
									</Badge>
								</div>
							</a>
						{/each}
					</div>
				{:else}
					<div class="text-center py-6">
						<Users class="h-8 w-8 text-text-muted mx-auto mb-2" />
						<p class="text-sm text-text-muted">No lists yet</p>
						<a href="{basePath}/lists" class="text-sm text-primary hover:underline mt-1 inline-block">
							Create your first list
						</a>
					</div>
				{/if}
			</Card>

			<!-- Recent Campaigns -->
			<Card>
				<div class="flex items-center justify-between mb-4">
					<h2 class="text-lg font-semibold text-text">Recent Campaigns</h2>
					<a href="{basePath}/campaigns" class="text-sm font-medium text-primary hover:underline flex items-center gap-1">
						View all
						<ArrowRight class="h-4 w-4" />
					</a>
				</div>
				{#if recentCampaigns.length > 0}
					<div class="space-y-3">
						{#each recentCampaigns as campaign}
							<a href="{basePath}/campaigns/{campaign.id}" class="flex items-center justify-between py-2 hover:bg-bg-secondary -mx-2 px-2 rounded-lg transition-colors">
								<div class="min-w-0">
									<p class="text-sm font-medium text-text truncate">{campaign.name}</p>
									<p class="text-xs text-text-muted truncate">{campaign.subject}</p>
								</div>
								<div class="flex items-center gap-2 ml-4">
									<Badge
										variant={campaign.status === 'sent' ? 'success' : campaign.status === 'draft' ? 'default' : 'info'}
										size="sm"
									>
										{campaign.status}
									</Badge>
								</div>
							</a>
						{/each}
					</div>
				{:else}
					<div class="text-center py-6">
						<Send class="h-8 w-8 text-text-muted mx-auto mb-2" />
						<p class="text-sm text-text-muted">No campaigns yet</p>
						<a href="{basePath}/campaigns" class="text-sm text-primary hover:underline mt-1 inline-block">
							Create your first campaign
						</a>
					</div>
				{/if}
			</Card>
		</div>

		<!-- Quick Actions -->
		<Card>
			<h2 class="text-lg font-semibold text-text mb-4">Quick Actions</h2>
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
				<a
					href="{basePath}/campaigns/new"
					class="flex items-center gap-3 p-3 rounded-lg border border-border hover:bg-bg-secondary transition-colors"
				>
					<div class="p-2 bg-green-100 rounded-lg">
						<Send class="h-4 w-4 text-green-600" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">New Campaign</p>
						<p class="text-xs text-text-muted">Send emails</p>
					</div>
				</a>
				<a
					href="{basePath}/lists"
					class="flex items-center gap-3 p-3 rounded-lg border border-border hover:bg-bg-secondary transition-colors"
				>
					<div class="p-2 bg-blue-100 rounded-lg">
						<Users class="h-4 w-4 text-blue-600" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">Manage Lists</p>
						<p class="text-xs text-text-muted">Subscribers</p>
					</div>
				</a>
				<a
					href="{basePath}/templates"
					class="flex items-center gap-3 p-3 rounded-lg border border-border hover:bg-bg-secondary transition-colors"
				>
					<div class="p-2 bg-amber-100 rounded-lg">
						<Mail class="h-4 w-4 text-amber-600" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">Templates</p>
						<p class="text-xs text-text-muted">Email designs</p>
					</div>
				</a>
			</div>
		</Card>
	{/if}
</div>
