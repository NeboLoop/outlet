<script lang="ts">
	import { page } from '$app/stores';
	import {
		listListSubscribers,
		removeListSubscriber,
		listSequences,
		enrollInSequence,
		getSubscriberDetail,
		type ListSubscriberInfo,
		type SequenceInfo,
		type SubscriberDetailResponse
	} from '$lib/api';
	import {
		Button,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		SearchInput,
		Select,
		Modal,
		AlertDialog
	} from '$lib/components/ui';
	import {
		Users,
		Trash2,
		Mail,
		Eye,
		MousePointer,
		Shield,
		UserPlus,
		X
	} from '@lucide/svelte';
	import { getListContext } from '../listContext';

	const ctx = getListContext();
	let listId = $derived($page.params.id!);
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// State
	let subscribers = $state<ListSubscriberInfo[]>([]);
	let subscribersLoading = $state(true);
	let subscriberSearch = $state('');
	let error = $state('');

	// Subscriber detail modal
	let showSubscriberDetail = $state(false);
	let subscriberDetail = $state<SubscriberDetailResponse | null>(null);
	let subscriberDetailLoading = $state(false);
	let selectedSubscriber = $state<ListSubscriberInfo | null>(null);

	// Enroll in sequence modal
	let showEnrollModal = $state(false);
	let enrollingSubscriber = $state<ListSubscriberInfo | null>(null);
	let selectedSequenceSlug = $state('');
	let enrolling = $state(false);
	let enrollSuccess = $state(false);
	let sequences = $state<SequenceInfo[]>([]);
	let sequencesLoading = $state(false);
	let sequencesLoaded = $state(false);

	// Filtered subscribers
	let filteredSubscribers = $derived(
		subscribers.filter(s =>
			s.email.toLowerCase().includes(subscriberSearch.toLowerCase()) ||
			(s.name || '').toLowerCase().includes(subscriberSearch.toLowerCase())
		)
	);

	$effect(() => {
		loadSubscribers();
	});

	async function loadSubscribers() {
		subscribersLoading = true;
		try {
			const response = await listListSubscribers({}, listId);
			subscribers = response.subscribers || [];
		} catch (err) {
			console.error('Failed to load subscribers:', err);
		} finally {
			subscribersLoading = false;
		}
	}

	async function loadSequences() {
		sequencesLoading = true;
		try {
			const response = await listSequences();
			sequences = response.sequences || [];
		} catch (err) {
			console.error('Failed to load sequences:', err);
		} finally {
			sequencesLoading = false;
			sequencesLoaded = true;
		}
	}

	let showRemoveConfirm = $state(false);
	let removeSubscriber = $state<ListSubscriberInfo | null>(null);
	let removing = $state(false);

	function confirmRemoveSubscriber(subscriber: ListSubscriberInfo) {
		removeSubscriber = subscriber;
		showRemoveConfirm = true;
	}

	async function executeRemoveSubscriber() {
		if (!removeSubscriber) return;
		removing = true;
		try {
			await removeListSubscriber({}, listId, removeSubscriber.id);
			await loadSubscribers();
		} catch (err: any) {
			error = err.message || 'Failed to remove subscriber';
		} finally {
			removing = false;
		}
	}

	async function openSubscriberDetail(subscriber: ListSubscriberInfo) {
		selectedSubscriber = subscriber;
		showSubscriberDetail = true;
		subscriberDetail = null;
		subscriberDetailLoading = true;
		try {
			subscriberDetail = await getSubscriberDetail({}, listId, subscriber.id);
		} catch (err) {
			console.error('Failed to load subscriber detail:', err);
			error = 'Failed to load subscriber details';
		} finally {
			subscriberDetailLoading = false;
		}
	}

	function openEnrollModal(subscriber: ListSubscriberInfo) {
		enrollingSubscriber = subscriber;
		selectedSequenceSlug = '';
		enrollSuccess = false;
		showEnrollModal = true;
		if (!sequencesLoaded) {
			loadSequences();
		}
	}

	async function handleEnrollInSequence() {
		if (!enrollingSubscriber || !selectedSequenceSlug) return;
		enrolling = true;
		error = '';
		try {
			await enrollInSequence({
				sequence_slug: selectedSequenceSlug,
				email: enrollingSubscriber.email
			});
			enrollSuccess = true;
			setTimeout(() => {
				showEnrollModal = false;
				enrollingSubscriber = null;
				enrollSuccess = false;
			}, 1500);
		} catch (err: any) {
			error = err.message || 'Failed to enroll in sequence';
		} finally {
			enrolling = false;
		}
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

{#if subscribersLoading}
	<div class="flex justify-center py-12">
		<LoadingSpinner />
	</div>
{:else if subscribers.length === 0}
	<EmptyState
		icon={Users}
		title="No subscribers yet"
		description="Subscribers will appear here when they join this list."
	/>
{:else}
	{#if subscribers.length > 5}
		<div class="mb-4">
			<SearchInput bind:value={subscriberSearch} placeholder="Search subscribers..." />
		</div>
	{/if}

	<div class="data-table">
		<table class="w-full text-sm">
			<thead>
				<tr>
					<th class="text-left">Email</th>
					<th class="text-left">Name</th>
					<th class="text-left">Status</th>
					<th class="text-left">Subscribed</th>
				</tr>
			</thead>
			<tbody>
				{#each filteredSubscribers as subscriber}
					<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_noninteractive_element_interactions -->
					<tr
						class="cursor-pointer hover:bg-bg-secondary/50 transition-colors"
						onclick={() => openSubscriberDetail(subscriber)}
						role="button"
						tabindex="0"
					>
						<td class="font-medium text-text">{subscriber.email}</td>
						<td class="text-text-muted">{subscriber.name || '-'}</td>
						<td>
							{#if subscriber.status === 'confirmed' || subscriber.status === 'active'}
								<Badge variant="success" size="sm">Confirmed</Badge>
							{:else if subscriber.status === 'pending'}
								<Badge variant="warning" size="sm">Pending</Badge>
							{:else if subscriber.status === 'unsubscribed'}
								<Badge variant="default" size="sm">Unsubscribed</Badge>
							{:else}
								<Badge variant="default" size="sm">{subscriber.status}</Badge>
							{/if}
						</td>
						<td class="text-text-muted">
							{#if subscriber.subscribed_at}
								{new Date(subscriber.subscribed_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}
							{:else}
								-
							{/if}
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<!-- Enroll in Sequence Modal -->
<Modal
	bind:show={showEnrollModal}
	title="Enroll in Sequence"
	size="sm"
>
	{#if enrollingSubscriber}
		<div class="space-y-4">
			<p class="text-sm text-base-content/70">
				Select a sequence to enroll <span class="font-medium text-base-content">{enrollingSubscriber.email}</span> into:
			</p>

			{#if sequencesLoading}
				<div class="text-center py-4">
					<LoadingSpinner size="small" />
					<p class="text-sm text-base-content/60 mt-2">Loading sequences...</p>
				</div>
			{:else if sequences.length === 0}
				<Alert type="info" title="No sequences available">
					<p>You haven't created any sequences yet. Create a sequence first to enroll subscribers.</p>
					<a href="{basePath}/sequences" class="btn btn-sm btn-primary mt-3">
						Create Sequence
					</a>
				</Alert>
			{:else}
				<Select
					bind:value={selectedSequenceSlug}
					options={[
						{ value: '', label: 'Select a sequence...' },
						...sequences.map(s => ({ value: s.slug, label: `${s.name} (${s.trigger_event})` }))
					]}
				/>
			{/if}

			{#if enrollSuccess}
				<Alert type="success" title="Enrolled">
					<p>{enrollingSubscriber.email} has been enrolled in the sequence.</p>
				</Alert>
			{/if}

			<div class="flex justify-end gap-3 pt-2">
				<Button type="secondary" onclick={() => showEnrollModal = false} disabled={enrolling}>
					Cancel
				</Button>
				{#if sequences.length > 0}
					<Button
						type="primary"
						onclick={handleEnrollInSequence}
						disabled={!selectedSequenceSlug || enrolling || enrollSuccess}
					>
						{#if enrolling}
							Enrolling...
						{:else if enrollSuccess}
							Enrolled!
						{:else}
							Enroll
						{/if}
					</Button>
				{/if}
			</div>
		</div>
	{/if}
</Modal>

<!-- Subscriber Detail Full-Screen Overlay -->
{#if showSubscriberDetail}
	<div class="fixed inset-0 z-50 bg-base-100 flex flex-col">
		<!-- Header -->
		<div class="flex items-center justify-between px-6 py-4 border-b border-base-300 bg-base-200">
			<div class="flex items-center gap-4">
				<button
					type="button"
					class="btn btn-ghost btn-sm btn-circle"
					onclick={() => showSubscriberDetail = false}
				>
					<X class="h-5 w-5" />
				</button>
				<div>
					<h2 class="text-xl font-semibold">Subscriber Details</h2>
					{#if subscriberDetail}
						<p class="text-sm text-base-content/60">{subscriberDetail.email}</p>
					{/if}
				</div>
			</div>
			{#if subscriberDetail}
				<div class="flex items-center gap-2">
					<Button
						type="secondary"
						size="sm"
						onclick={() => {
							if (selectedSubscriber) {
								openEnrollModal(selectedSubscriber);
							}
						}}
					>
						<UserPlus class="h-4 w-4 mr-1.5" />
						Enroll in Sequence
					</Button>
					<Button
						type="danger"
						size="sm"
						onclick={() => {
							if (selectedSubscriber) {
								showSubscriberDetail = false;
								confirmRemoveSubscriber(selectedSubscriber);
							}
						}}
					>
						<Trash2 class="h-4 w-4 mr-1.5" />
						Remove from List
					</Button>
				</div>
			{/if}
		</div>

		<!-- Content -->
		<div class="flex-1 overflow-y-auto">
			{#if subscriberDetailLoading}
				<div class="flex justify-center items-center h-full">
					<LoadingSpinner size="large" />
				</div>
			{:else if subscriberDetail}
				<div class="max-w-6xl mx-auto p-6">
					<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
						<!-- Left Column: Profile & Stats -->
						<div class="space-y-6">
							<!-- Profile Card -->
							<div class="bg-base-200 rounded-xl p-6">
								<div class="flex items-start justify-between mb-4">
									<div class="flex items-center gap-4">
										<div class="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
											<span class="text-2xl font-bold text-primary">
												{(subscriberDetail.name || subscriberDetail.email).charAt(0).toUpperCase()}
											</span>
										</div>
										<div>
											<h3 class="text-lg font-semibold">{subscriberDetail.name || 'No name'}</h3>
											<p class="text-sm text-base-content/60">{subscriberDetail.email}</p>
										</div>
									</div>
								</div>
								<div class="flex items-center gap-2">
									{#if subscriberDetail.status === 'active'}
										<Badge variant="success">Active</Badge>
									{:else if subscriberDetail.status === 'pending'}
										<Badge variant="warning">Pending</Badge>
									{:else if subscriberDetail.status === 'unsubscribed'}
										<Badge variant="default">Unsubscribed</Badge>
									{:else}
										<Badge variant="default">{subscriberDetail.status}</Badge>
									{/if}
									{#if subscriberDetail.gdpr_consent}
										<Badge variant="info" size="sm">
											<Shield class="h-3 w-3 mr-1" />
											GDPR
										</Badge>
									{/if}
								</div>
							</div>

							<!-- Stats Cards -->
							<div class="grid grid-cols-3 gap-3">
								<div class="bg-base-200 rounded-lg p-4 text-center">
									<Mail class="h-5 w-5 mx-auto mb-2 text-base-content/60" />
									<div class="text-2xl font-bold">{subscriberDetail.emails_sent}</div>
									<div class="text-xs text-base-content/60 uppercase">Sent</div>
								</div>
								<div class="bg-base-200 rounded-lg p-4 text-center">
									<Eye class="h-5 w-5 mx-auto mb-2 text-base-content/60" />
									<div class="text-2xl font-bold">{subscriberDetail.emails_opened}</div>
									<div class="text-xs text-base-content/60 uppercase">Opened</div>
								</div>
								<div class="bg-base-200 rounded-lg p-4 text-center">
									<MousePointer class="h-5 w-5 mx-auto mb-2 text-base-content/60" />
									<div class="text-2xl font-bold">{subscriberDetail.emails_clicked}</div>
									<div class="text-xs text-base-content/60 uppercase">Clicked</div>
								</div>
							</div>

							<!-- Details -->
							<div class="bg-base-200 rounded-xl p-6">
								<h4 class="font-semibold mb-4">Details</h4>
								<dl class="space-y-3 text-sm">
									<div class="flex justify-between">
										<dt class="text-base-content/60">Subscribed</dt>
										<dd class="font-medium">
											{subscriberDetail.subscribed_at ? new Date(subscriberDetail.subscribed_at).toLocaleDateString() : '-'}
										</dd>
									</div>
									<div class="flex justify-between">
										<dt class="text-base-content/60">Verified</dt>
										<dd class="font-medium">
											{#if subscriberDetail.email_verified}
												<span class="text-success">Yes</span>
											{:else}
												<span class="text-warning">Not verified</span>
											{/if}
										</dd>
									</div>
									<div class="flex justify-between">
										<dt class="text-base-content/60">Source</dt>
										<dd class="font-medium">{subscriberDetail.source || '-'}</dd>
									</div>
									<div class="flex justify-between">
										<dt class="text-base-content/60">Last Activity</dt>
										<dd class="font-medium">
											{subscriberDetail.last_open_at
												? new Date(subscriberDetail.last_open_at).toLocaleDateString()
												: subscriberDetail.last_email_at
													? new Date(subscriberDetail.last_email_at).toLocaleDateString()
													: '-'}
										</dd>
									</div>
								</dl>
							</div>

							<!-- Custom Fields -->
							{#if subscriberDetail.custom_fields && Object.keys(subscriberDetail.custom_fields).length > 0}
								<div class="bg-base-200 rounded-xl p-6">
									<h4 class="font-semibold mb-4">Custom Fields</h4>
									<dl class="space-y-3 text-sm">
										{#each Object.entries(subscriberDetail.custom_fields) as [key, value]}
											<div class="flex justify-between">
												<dt class="text-base-content/60">{key}</dt>
												<dd class="font-medium">{value || '-'}</dd>
											</div>
										{/each}
									</dl>
								</div>
							{/if}
						</div>

						<!-- Right Column: Activity -->
						<div class="lg:col-span-2 space-y-6">
							<!-- Campaign Activity -->
							<div class="bg-base-200 rounded-xl p-6">
								<h4 class="font-semibold mb-4">Campaign Activity</h4>
								{#if subscriberDetail.campaign_activity && subscriberDetail.campaign_activity.length > 0}
									<div class="overflow-x-auto">
										<table class="table table-sm">
											<thead>
												<tr>
													<th>Campaign</th>
													<th class="text-center">Sent</th>
													<th class="text-center">Opens</th>
													<th class="text-center">Clicks</th>
												</tr>
											</thead>
											<tbody>
												{#each subscriberDetail.campaign_activity as activity}
													<tr>
														<td class="font-medium">{activity.campaign_name || activity.campaign_subject || 'Unknown'}</td>
														<td class="text-center text-base-content/60">
															{activity.sent_at ? new Date(activity.sent_at).toLocaleDateString() : '-'}
														</td>
														<td class="text-center">
															{#if activity.opened_at}
																<Badge variant="success" size="sm">{activity.open_count}</Badge>
															{:else}
																<span class="text-base-content/40">-</span>
															{/if}
														</td>
														<td class="text-center">
															{#if activity.clicked_at}
																<Badge variant="info" size="sm">{activity.click_count}</Badge>
															{:else}
																<span class="text-base-content/40">-</span>
															{/if}
														</td>
													</tr>
												{/each}
											</tbody>
										</table>
									</div>
								{:else}
									<p class="text-base-content/60 text-sm italic">No campaign activity yet</p>
								{/if}
							</div>

							<!-- Sequence Enrollments -->
							<div class="bg-base-200 rounded-xl p-6">
								<h4 class="font-semibold mb-4">Sequence Enrollments</h4>
								{#if subscriberDetail.sequence_enrollments && subscriberDetail.sequence_enrollments.length > 0}
									<div class="space-y-3">
										{#each subscriberDetail.sequence_enrollments as enrollment}
											<div class="flex items-center justify-between p-4 bg-base-100 rounded-lg">
												<div>
													<p class="font-medium">{enrollment.sequence_name || 'Unknown Sequence'}</p>
													<p class="text-xs text-base-content/60">
														Step {enrollment.current_position} • Started {enrollment.started_at ? new Date(enrollment.started_at).toLocaleDateString() : '-'}
													</p>
												</div>
												<div>
													{#if enrollment.completed_at}
														<Badge variant="success" size="sm">Completed</Badge>
													{:else if enrollment.paused_at}
														<Badge variant="warning" size="sm">Paused</Badge>
													{:else if enrollment.unsubscribed_at}
														<Badge variant="default" size="sm">Unsubscribed</Badge>
													{:else if enrollment.is_active}
														<Badge variant="info" size="sm">Active</Badge>
													{:else}
														<Badge variant="default" size="sm">Inactive</Badge>
													{/if}
												</div>
											</div>
										{/each}
									</div>
								{:else}
									<p class="text-base-content/60 text-sm italic">Not enrolled in any sequences</p>
								{/if}
							</div>
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}

<AlertDialog
	bind:open={showRemoveConfirm}
	title="Remove Subscriber"
	description={`Remove ${removeSubscriber?.email || ''} from this list?`}
	actionLabel={removing ? 'Removing...' : 'Remove'}
	actionType="danger"
	onAction={executeRemoveSubscriber}
	onCancel={() => showRemoveConfirm = false}
/>
