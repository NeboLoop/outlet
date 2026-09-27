<script lang="ts">
	import { page } from '$app/stores';
	import {
		listSuppressedEmails,
		addSuppressedEmail,
		deleteSuppressedEmail,
		listBlockedDomains,
		createBlockedDomain,
		deleteBlockedDomain,
		type SuppressedEmailInfo,
		type BlockedDomainInfo
	} from '$lib/api';
	import {
		Button,
		Card,
		Input,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		SearchInput,
		AlertDialog,
		Tabs
	} from '$lib/components/ui';
	import {
		Ban,
		Mail,
		Globe,
		Plus,
		Trash2,
		Search
	} from '@lucide/svelte';

	let basePath = $derived(`/${$page.params.brandSlug}`);

	// Tab state
	let activeTab = $state('emails');

	// Suppressed emails state
	let emails = $state<SuppressedEmailInfo[]>([]);
	let emailsLoading = $state(true);
	let emailSearch = $state('');
	let newEmail = $state('');
	let newEmailReason = $state('');
	let addingEmail = $state(false);

	// Blocked domains state
	let domains = $state<BlockedDomainInfo[]>([]);
	let domainsLoading = $state(true);
	let domainSearch = $state('');
	let newDomain = $state('');
	let newDomainReason = $state('');
	let addingDomain = $state(false);

	// Error state
	let error = $state('');

	// Filtered lists
	let filteredEmails = $derived(
		emails.filter(e =>
			e.email.toLowerCase().includes(emailSearch.toLowerCase()) ||
			(e.reason || '').toLowerCase().includes(emailSearch.toLowerCase())
		)
	);

	let filteredDomains = $derived(
		domains.filter(d =>
			d.domain.toLowerCase().includes(domainSearch.toLowerCase()) ||
			(d.reason || '').toLowerCase().includes(domainSearch.toLowerCase())
		)
	);

	$effect(() => {
		if (activeTab === 'emails') {
			loadEmails();
		} else if (activeTab === 'domains') {
			loadDomains();
		}
	});

	async function loadEmails() {
		emailsLoading = true;
		try {
			const response = await listSuppressedEmails({});
			emails = response.emails || [];
		} catch (err) {
			console.error('Failed to load suppressed emails:', err);
		} finally {
			emailsLoading = false;
		}
	}

	async function loadDomains() {
		domainsLoading = true;
		try {
			const response = await listBlockedDomains({});
			domains = response.domains || [];
		} catch (err) {
			console.error('Failed to load blocked domains:', err);
		} finally {
			domainsLoading = false;
		}
	}

	async function handleAddEmail() {
		if (!newEmail.trim()) return;
		addingEmail = true;
		error = '';
		try {
			await addSuppressedEmail({
				email: newEmail.trim().toLowerCase(),
				reason: newEmailReason.trim() || 'Manual suppression'
			});
			newEmail = '';
			newEmailReason = '';
			await loadEmails();
		} catch (err: any) {
			error = err.message || 'Failed to add email to suppression list';
		} finally {
			addingEmail = false;
		}
	}

	let showDeleteEmailConfirm = $state(false);
	let deleteEmailItem = $state<SuppressedEmailInfo | null>(null);

	function confirmDeleteEmail(email: SuppressedEmailInfo) {
		deleteEmailItem = email;
		showDeleteEmailConfirm = true;
	}

	async function executeDeleteEmail() {
		if (!deleteEmailItem) return;
		try {
			await deleteSuppressedEmail({}, deleteEmailItem.id);
			await loadEmails();
		} catch (err: any) {
			error = err.message || 'Failed to remove email';
		}
	}

	async function handleAddDomain() {
		if (!newDomain.trim()) return;
		addingDomain = true;
		error = '';
		try {
			await createBlockedDomain({
				domain: newDomain.trim().toLowerCase(),
				reason: newDomainReason.trim() || 'Manual block'
			});
			newDomain = '';
			newDomainReason = '';
			await loadDomains();
		} catch (err: any) {
			error = err.message || 'Failed to block domain';
		} finally {
			addingDomain = false;
		}
	}

	let showDeleteDomainConfirm = $state(false);
	let deleteDomainItem = $state<BlockedDomainInfo | null>(null);

	function confirmDeleteDomain(domain: BlockedDomainInfo) {
		deleteDomainItem = domain;
		showDeleteDomainConfirm = true;
	}

	async function executeDeleteDomain() {
		if (!deleteDomainItem) return;
		try {
			await deleteBlockedDomain({}, deleteDomainItem.id);
			await loadDomains();
		} catch (err: any) {
			error = err.message || 'Failed to unblock domain';
		}
	}

	const tabs = [
		{ id: 'emails', label: 'Suppressed Emails', icon: Mail },
		{ id: 'domains', label: 'Blocked Domains', icon: Globe }
	];
</script>

<svelte:head>
	<title>Blocklist | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<!-- Header -->
	<div class="mb-6">
		<h1 class="text-2xl font-semibold text-text">Blocklist</h1>
		<p class="mt-1 text-sm text-text-muted">
			Manage suppressed emails and blocked domains. Blocked addresses won't receive any emails.
		</p>
	</div>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	<!-- Tabs -->
	<div class="border-b border-border mb-6">
		<nav class="-mb-px flex space-x-8">
			{#each tabs as tab}
				<button
					onclick={() => activeTab = tab.id}
					class="whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium flex items-center gap-2 {activeTab === tab.id
						? 'border-primary text-primary'
						: 'border-transparent text-text-muted hover:border-border hover:text-text'}"
				>
					<svelte:component this={tab.icon} class="h-4 w-4" />
					{tab.label}
				</button>
			{/each}
		</nav>
	</div>

	<!-- Tab Content -->
	{#if activeTab === 'emails'}
		<!-- Suppressed Emails Tab -->
		<Card class="mb-6">
			<h2 class="text-lg font-medium text-text mb-4">Add Email to Suppression List</h2>
			<div class="flex gap-3">
				<Input
					type="email"
					bind:value={newEmail}
					placeholder="email@example.com"
					class="flex-1"
				/>
				<Input
					type="text"
					bind:value={newEmailReason}
					placeholder="Reason (optional)"
					class="flex-1"
				/>
				<Button type="primary" onclick={handleAddEmail} disabled={!newEmail.trim() || addingEmail}>
					{#if addingEmail}
						Adding...
					{:else}
						<Plus class="mr-2 h-4 w-4" />
						Add
					{/if}
				</Button>
			</div>
		</Card>

		{#if emailsLoading}
			<div class="flex justify-center py-12">
				<LoadingSpinner />
			</div>
		{:else if emails.length === 0}
			<EmptyState
				icon={Mail}
				title="No suppressed emails"
				message="Add email addresses that should never receive emails from this brand."
			/>
		{:else}
			{#if emails.length > 5}
				<div class="mb-4">
					<SearchInput bind:value={emailSearch} placeholder="Search suppressed emails..." />
				</div>
			{/if}

			<div class="data-table">
				<table class="w-full">
					<thead>
						<tr>
							<th class="text-left">Email</th>
							<th class="text-left">Reason</th>
							<th class="text-left">Source</th>
							<th class="text-left">Added</th>
							<th class="text-right w-10"></th>
						</tr>
					</thead>
					<tbody>
						{#each filteredEmails as email}
							<tr>
								<td class="font-medium text-text">{email.email}</td>
								<td class="text-text-muted">{email.reason || '-'}</td>
								<td>
									<Badge variant="default" size="sm">{email.source}</Badge>
								</td>
								<td class="text-text-muted text-sm">
									{new Date(email.created_at).toLocaleDateString()}
								</td>
								<td class="text-right">
									<Button
										type="ghost"
										size="sm"
										onclick={() => confirmDeleteEmail(email)}
										class="text-text-muted hover:text-error"
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}

	{:else if activeTab === 'domains'}
		<!-- Blocked Domains Tab -->
		<Card class="mb-6">
			<h2 class="text-lg font-medium text-text mb-4">Block a Domain</h2>
			<p class="text-sm text-text-muted mb-4">
				All emails from blocked domains will be rejected. Use this to block spam or competitor domains.
			</p>
			<div class="flex gap-3">
				<Input
					type="text"
					bind:value={newDomain}
					placeholder="example.com"
					class="flex-1"
				/>
				<Input
					type="text"
					bind:value={newDomainReason}
					placeholder="Reason (optional)"
					class="flex-1"
				/>
				<Button type="primary" onclick={handleAddDomain} disabled={!newDomain.trim() || addingDomain}>
					{#if addingDomain}
						Blocking...
					{:else}
						<Plus class="mr-2 h-4 w-4" />
						Block
					{/if}
				</Button>
			</div>
		</Card>

		{#if domainsLoading}
			<div class="flex justify-center py-12">
				<LoadingSpinner />
			</div>
		{:else if domains.length === 0}
			<EmptyState
				icon={Globe}
				title="No blocked domains"
				message="Block domains to prevent all emails from those addresses."
			/>
		{:else}
			{#if domains.length > 5}
				<div class="mb-4">
					<SearchInput bind:value={domainSearch} placeholder="Search blocked domains..." />
				</div>
			{/if}

			<div class="data-table">
				<table class="w-full">
					<thead>
						<tr>
							<th class="text-left">Domain</th>
							<th class="text-left">Reason</th>
							<th class="text-left">Block Attempts</th>
							<th class="text-left">Added</th>
							<th class="text-right w-10"></th>
						</tr>
					</thead>
					<tbody>
						{#each filteredDomains as domain}
							<tr>
								<td class="font-medium text-text">{domain.domain}</td>
								<td class="text-text-muted">{domain.reason || '-'}</td>
								<td>
									{#if domain.block_attempts > 0}
										<Badge variant="warning" size="sm">{domain.block_attempts}</Badge>
									{:else}
										<span class="text-text-muted">0</span>
									{/if}
								</td>
								<td class="text-text-muted text-sm">
									{new Date(domain.created_at).toLocaleDateString()}
								</td>
								<td class="text-right">
									<Button
										type="ghost"
										size="sm"
										onclick={() => confirmDeleteDomain(domain)}
										class="text-text-muted hover:text-error"
									>
										<Trash2 class="h-4 w-4" />
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={showDeleteEmailConfirm}
	title="Remove Suppressed Email"
	description={`Remove ${deleteEmailItem?.email || ''} from the suppression list?`}
	actionLabel="Remove"
	actionType="danger"
	onAction={executeDeleteEmail}
	onCancel={() => showDeleteEmailConfirm = false}
/>

<AlertDialog
	bind:open={showDeleteDomainConfirm}
	title="Unblock Domain"
	description={`Unblock domain ${deleteDomainItem?.domain || ''}?`}
	actionLabel="Unblock"
	actionType="danger"
	onAction={executeDeleteDomain}
	onCancel={() => showDeleteDomainConfirm = false}
/>
