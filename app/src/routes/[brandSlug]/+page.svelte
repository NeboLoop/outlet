<script lang="ts">
	import { page } from '$app/stores';
	import { browser } from '$app/environment';
	import { goto } from '$app/navigation';
	import { Card, Badge, LoadingSpinner, EmptyState, Alert } from '$lib/components/ui';
	import { ChartCard, AreaChart } from '$lib/components/charts';
	import { getEmailDashboardStats, listLists, listCampaigns, getCurrentUser, type EmailDashboardStatsResponse, type ListInfo, type CampaignInfo } from '$lib/api';
	import { TrendingUp, TrendingDown, Users, Mail, Send, MousePointerClick, AlertTriangle, UserMinus, ArrowRight, Plus, RefreshCw, MessageSquare, Flame } from '@lucide/svelte';
	import { outbound, refreshOutbound } from '$lib/stores/outbound.svelte';
	import SetupCard from './outbound/SetupCard.svelte';
	import { dashboardOutbound, percent, waitingOnYou } from './outbound/shared';

	interface Props {
		data: {
			brandSlug?: string;
		};
	}

	const { data }: Props = $props();
	let basePath = $derived(data.brandSlug ? `/${data.brandSlug}` : '');
	let userName = $state('');

	let loading = $state(true);
	let emailStats = $state<EmailDashboardStatsResponse | null>(null);
	let statsError = $state('');
	let lists = $state<ListInfo[]>([]);
	let recentCampaigns = $state<CampaignInfo[]>([]);
	let orgName = $state('');

	// Outbound's numbers come from the overview the brand layout loads on
	// each navigation; the row shows only while outbound is on.
	let outboundBase = $derived(`${basePath}/outbound`);
	let outboundView = $derived(dashboardOutbound(outbound.overview));
	let waiting = $derived(outbound.overview ? waitingOnYou(outbound.overview, outboundBase) : []);

	$effect(() => {
		loadDashboardData();
	});

	$effect(() => {
		getCurrentUser()
			.then((me) => (userName = me.name || me.email.split('@')[0]))
			.catch(() => {});
	});

	async function loadDashboardData() {
		loading = true;
		statsError = '';

		try {
			const orgId = browser ? localStorage.getItem('currentOrgId') : null;
			if (!orgId) {
				loading = false;
				return;
			}

			// Load email stats, lists, and recent campaigns in parallel
			const [statsResult, listsResult, campaignsResult] = await Promise.all([
				getEmailDashboardStats({}).catch((err: Error) => {
					statsError = err.message || 'Failed to load email stats';
					return null;
				}),
				// Newsletter lists only: outreach lists are Outbound's.
				listLists({ kind: 'subscribers' }).catch(() => ({ lists: [] })),
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

	function refresh() {
		loadDashboardData();
		const orgId = browser ? localStorage.getItem('currentOrgId') : null;
		if (orgId) refreshOutbound(orgId);
	}

	function formatNumber(num: number | null | undefined): string {
		if (num == null) return '0';
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return num.toString();
	}

	// Rates come as fractions: 0.25 is 25%
	function formatPercent(rate: number | null | undefined): string {
		if (rate == null) return '0%';
		return (rate * 100).toFixed(1) + '%';
	}


	// Calculate total subscribers across all lists
	let totalSubscribers = $derived(lists.reduce((sum, list) => sum + (list.subscriber_count || 0), 0));

	// Check if user has any data yet
	let hasData = $derived(lists.length > 0 || (emailStats && emailStats.total_sent > 0));
</script>

{#snippet outboundRow()}
	{#if outbound.overview && outboundView !== 'off'}
		{@const o = outbound.overview}
		<section aria-labelledby="dash-outbound-title">
			<div class="flex items-center justify-between gap-3 mb-3">
				<h2 id="dash-outbound-title" class="text-lg font-semibold text-text">Outbound</h2>
				<a href={outboundBase} class="text-sm font-medium text-primary hover:underline flex items-center gap-1">
					Open outbound
					<ArrowRight class="h-4 w-4" />
				</a>
			</div>
			{#if outboundView === 'setup'}
				<SetupCard setup={o.setup || []} base={outboundBase} />
			{:else}
				<div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
					<a href="{outboundBase}/prospects" class="block">
						<Card class="hover:shadow-md transition-shadow cursor-pointer">
							<div class="flex items-start justify-between">
								<div>
									<p class="stat-label">Prospects</p>
									<p class="stat-value mt-1">{formatNumber(o.prospects_total)}</p>
									<p class="text-xs text-text-muted mt-1">On outreach lists</p>
								</div>
								<div class="p-2 bg-info/10 rounded-lg">
									<Users class="h-5 w-5 text-info" />
								</div>
							</div>
						</Card>
					</a>

					<a href="{outboundBase}/campaigns" class="block">
						<Card class="hover:shadow-md transition-shadow cursor-pointer">
							<div class="flex items-start justify-between">
								<div>
									<p class="stat-label">Emails sent</p>
									<p class="stat-value mt-1">{formatNumber(o.messages?.sent)}</p>
									<p class="text-xs text-text-muted mt-1">From your mailboxes</p>
								</div>
								<div class="p-2 bg-success/10 rounded-lg">
									<Send class="h-5 w-5 text-success" />
								</div>
							</div>
						</Card>
					</a>

					<a href={outboundBase} class="block">
						<Card class="hover:shadow-md transition-shadow cursor-pointer">
							<div class="flex items-start justify-between">
								<div>
									<p class="stat-label">Reply rate</p>
									<p class="stat-value mt-1">{percent(o.reply_rate)}</p>
									<p class="text-xs text-text-muted mt-1">Across campaigns</p>
								</div>
								<div class="p-2 bg-primary/10 rounded-lg">
									<MessageSquare class="h-5 w-5 text-primary" />
								</div>
							</div>
						</Card>
					</a>

					<a href="{outboundBase}/replies?hot=1" class="block">
						<Card class="hover:shadow-md transition-shadow cursor-pointer">
							<div class="flex items-start justify-between">
								<div>
									<p class="stat-label">Hot leads</p>
									<p class="stat-value mt-1">{formatNumber(o.hot_leads_waiting)}</p>
									<p class="text-xs text-text-muted mt-1">Interested, waiting on you</p>
								</div>
								<div class="p-2 bg-warning/10 rounded-lg">
									<Flame class="h-5 w-5 text-warning" />
								</div>
							</div>
						</Card>
					</a>
				</div>
				{#if waiting.length > 0}
					<p class="dash-waiting">
						<span class="font-medium text-text">Waiting on you</span>
						{#each waiting as w (w.href)}
							<a href={w.href} class="dash-waiting-link">{w.label}</a>
						{/each}
					</p>
				{/if}
			{/if}
		</section>
	{/if}
{/snippet}

<svelte:head>
	<title>Dashboard - {orgName || 'Admin'}</title>
</svelte:head>

<div class="p-6 space-y-6 max-w-6xl mx-auto">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-semibold text-text">Welcome back, {userName || 'there'}</h1>
			<p class="mt-1 text-sm text-text-muted">{orgName}</p>
		</div>
		<button
			onclick={refresh}
			disabled={loading}
			class="p-2 rounded-lg hover:bg-bg-secondary transition-colors"
			title="Refresh"
		>
			<RefreshCw class="h-5 w-5 text-text-muted {loading ? 'animate-spin' : ''}" />
		</button>
	</div>

	{#if statsError}
		<Alert type="error" title="Email stats">
			<p>{statsError}</p>
		</Alert>
	{/if}

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
					class="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-content font-medium hover:bg-primary/90 transition-colors"
				>
					<Plus class="h-4 w-4" />
					Create a List
				</a>
			</div>
		</Card>
		{@render outboundRow()}
	{:else}
		<!-- Email Metrics -->
		<div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
			<!-- Total Subscribers -->
			<a href="{basePath}/lists" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="stat-label">Subscribers</p>
							<p class="stat-value mt-1">{formatNumber(totalSubscribers)}</p>
							<p class="text-xs text-text-muted mt-1">{lists.length} list{lists.length !== 1 ? 's' : ''}</p>
						</div>
						<div class="p-2 bg-info/10 rounded-lg">
							<Users class="h-5 w-5 text-info" />
						</div>
					</div>
				</Card>
			</a>

			<!-- Emails Sent -->
			<a href="{basePath}/campaigns" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="stat-label">Emails Sent</p>
							<p class="stat-value mt-1">{formatNumber(emailStats?.total_sent)}</p>
							<p class="text-xs text-text-muted mt-1">All time</p>
						</div>
						<div class="p-2 bg-success/10 rounded-lg">
							<Send class="h-5 w-5 text-success" />
						</div>
					</div>
				</Card>
			</a>

			<!-- Open Rate -->
			<a href="{basePath}/campaigns" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="stat-label">Open Rate</p>
							<p class="stat-value mt-1">{formatPercent(emailStats?.open_rate)}</p>
							<p class="text-xs text-text-muted mt-1">{formatNumber(emailStats?.total_opened)} opened</p>
						</div>
						<div class="p-2 bg-primary/10 rounded-lg">
							<Mail class="h-5 w-5 text-primary" />
						</div>
					</div>
				</Card>
			</a>

			<!-- Click Rate -->
			<a href="{basePath}/campaigns" class="block">
				<Card class="hover:shadow-md transition-shadow cursor-pointer">
					<div class="flex items-start justify-between">
						<div>
							<p class="stat-label">Click Rate</p>
							<p class="stat-value mt-1">{formatPercent(emailStats?.click_rate)}</p>
							<p class="text-xs text-text-muted mt-1">{formatNumber(emailStats?.total_clicked)} clicked</p>
						</div>
						<div class="p-2 bg-info/10 rounded-lg">
							<MousePointerClick class="h-5 w-5 text-info" />
						</div>
					</div>
				</Card>
			</a>
		</div>

		{@render outboundRow()}

		<!-- Secondary Stats Row -->
		{#if emailStats && (emailStats.total_bounced > 0 || emailStats.total_complaints > 0 || emailStats.total_unsubscribed > 0)}
			<div class="grid grid-cols-3 gap-4">
				<Card class="flex items-center gap-3 p-4">
					<div class="p-2 bg-warning/10 rounded-lg">
						<AlertTriangle class="h-4 w-4 text-warning" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">{formatNumber(emailStats.total_bounced)}</p>
						<p class="text-xs text-text-muted">Bounced</p>
					</div>
				</Card>

				<Card class="flex items-center gap-3 p-4">
					<div class="p-2 bg-error/10 rounded-lg">
						<AlertTriangle class="h-4 w-4 text-error" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">{formatNumber(emailStats.total_complaints)}</p>
						<p class="text-xs text-text-muted">Complaints</p>
					</div>
				</Card>

				<Card class="flex items-center gap-3 p-4">
					<div class="p-2 bg-base-200 rounded-lg">
						<UserMinus class="h-4 w-4 text-text-muted" />
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
					<div class="p-2 bg-success/10 rounded-lg">
						<Send class="h-4 w-4 text-success" />
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
					<div class="p-2 bg-info/10 rounded-lg">
						<Users class="h-4 w-4 text-info" />
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
					<div class="p-2 bg-warning/10 rounded-lg">
						<Mail class="h-4 w-4 text-warning" />
					</div>
					<div>
						<p class="text-sm font-medium text-text">Designs</p>
						<p class="text-xs text-text-muted">Reusable email layouts</p>
					</div>
				</a>
			</div>
		</Card>
	{/if}
</div>
