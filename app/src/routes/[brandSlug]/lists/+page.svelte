<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		listLists,
		createList,
		deleteList,
		type ListInfo
	} from '$lib/api';
	import {
		Button,
		Card,
		Input,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		Checkbox,
		Modal,
		SearchInput,
		DropdownMenu,
		AlertDialog
	} from '$lib/components/ui';
	import { Plus, Mail, Users, MoreVertical, Trash2, Edit, ExternalLink } from '@lucide/svelte';

	let loading = $state(true);
	let lists = $state<ListInfo[]>([]);
	let error = $state('');
	let searchQuery = $state('');

	// Create form state
	let showCreateForm = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newSlug = $state('');
	let newDescription = $state('');
	let newDoubleOptin = $state(true);

	// Build base path with brandSlug
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// Filtered lists based on search
	let filteredLists = $derived(
		lists.filter(list =>
			list.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			list.slug.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';

		try {
			const response = await listLists();
			lists = response.lists || [];
		} catch (err) {
			console.error('Failed to load lists:', err);
			error = 'Failed to load email lists';
		} finally {
			loading = false;
		}
	}

	function openCreateForm() {
		newName = '';
		newSlug = '';
		newDescription = '';
		newDoubleOptin = true;
		showCreateForm = true;
	}

	function closeCreateForm() {
		showCreateForm = false;
	}

	function generateSlug(name: string): string {
		return name
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	function handleNameInput() {
		newSlug = generateSlug(newName);
	}

	async function submitCreate() {
		if (!newName || !newSlug) return;

		creating = true;
		error = '';
		try {
			await createList({
				name: newName,
				slug: newSlug,
				description: newDescription,
				double_optin: newDoubleOptin
			});
			closeCreateForm();
			await loadData();
		} catch (err: any) {
			console.error('Failed to create list:', err);
			error = err.message || 'Failed to create list';
		} finally {
			creating = false;
		}
	}

	let showDeleteConfirm = $state(false);
	let deleteListItem = $state<ListInfo | null>(null);
	let deleting = $state(false);

	function confirmDelete(list: ListInfo) {
		deleteListItem = list;
		showDeleteConfirm = true;
	}

	async function executeDelete() {
		if (!deleteListItem) return;
		deleting = true;
		try {
			await deleteList({}, deleteListItem.id);
			await loadData();
		} catch (err: any) {
			console.error('Failed to delete list:', err);
			error = err.message || 'Failed to delete list';
		} finally {
			deleting = false;
		}
	}
</script>

<svelte:head>
	<title>Lists | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<h1 class="text-2xl font-semibold text-text">Lists</h1>
			<p class="mt-1 text-sm text-text-muted">Manage your email subscriber lists</p>
		</div>
		<Button type="primary" onclick={openCreateForm}>
			<Plus class="mr-2 h-4 w-4" />
			New List
		</Button>
	</div>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	<!-- Create List Modal -->
	<Modal bind:show={showCreateForm} title="Create New List" size="md">
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
			<div>
				<label for="list-name" class="form-label">Name</label>
				<Input
					id="list-name"
					type="text"
					bind:value={newName}
					oninput={handleNameInput}
					placeholder="Newsletter"
				/>
			</div>
			<div>
				<label for="list-slug" class="form-label">Slug</label>
				<Input
					id="list-slug"
					type="text"
					bind:value={newSlug}
					placeholder="newsletter"
				/>
				<p class="mt-1 text-xs text-text-muted">Used in URLs and API</p>
			</div>
			<div class="sm:col-span-2">
				<label for="list-description" class="form-label">Description (optional)</label>
				<Input
					id="list-description"
					type="text"
					bind:value={newDescription}
					placeholder="Main newsletter for product updates"
				/>
			</div>
			<div class="sm:col-span-2">
				<Checkbox
					bind:checked={newDoubleOptin}
					label="Require double opt-in"
				/>
				<p class="text-xs text-text-muted mt-1 ml-6">
					Subscribers must confirm their email before being added
				</p>
			</div>
		</div>

		{#snippet footer()}
			<Button type="secondary" onclick={closeCreateForm} disabled={creating}>
				Cancel
			</Button>
			<Button type="primary" onclick={submitCreate} disabled={!newName || !newSlug || creating}>
				{creating ? 'Creating...' : 'Create List'}
			</Button>
		{/snippet}
	</Modal>

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if lists.length === 0}
		<EmptyState
			icon={Mail}
			title="No lists yet"
			description="Create your first email list to start collecting subscribers."
		>
			<Button type="primary" onclick={openCreateForm}>
				<Plus class="mr-2 h-4 w-4" />
				Create List
			</Button>
		</EmptyState>
	{:else}
		<!-- Search -->
		{#if lists.length > 3}
			<div class="mb-4">
				<SearchInput
					bind:value={searchQuery}
					placeholder="Search lists..."
				/>
			</div>
		{/if}

		<!-- Lists Table -->
		<div class="data-table">
			<table class="w-full">
				<thead>
					<tr>
						<th class="text-left">List</th>
						<th class="text-right">Subscribers</th>
						<th class="text-right w-10"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredLists as list}
						<tr>
							<td>
								<a href="{basePath}/lists/{list.id}" class="block group">
									<div class="flex items-center gap-3">
										<div class="data-table-icon">
											<Users class="h-4 w-4" />
										</div>
										<div class="min-w-0 flex-1">
											<div class="flex items-center gap-2 flex-wrap">
												<span class="font-medium text-text group-hover:text-primary transition-colors">
													{list.name}
												</span>
												{#if list.double_optin}
													<Badge variant="info" size="sm">Double opt-in</Badge>
												{/if}
											</div>
											{#if list.description}
												<p class="text-sm text-text-muted truncate">{list.description}</p>
											{/if}
										</div>
									</div>
								</a>
							</td>
							<td class="text-right">
								<span class="text-lg font-semibold text-text">{list.subscriber_count || 0}</span>
							</td>
							<td class="text-right">
								<DropdownMenu
									trigger={{
										icon: MoreVertical,
										class: 'p-1.5 rounded-lg hover:bg-bg-secondary transition-colors text-text-muted hover:text-text'
									}}
									items={[
										{
											label: 'Edit',
											icon: Edit,
											onclick: () => goto(`${basePath}/lists/${list.id}`)
										},
										{
											label: 'View subscribers',
											icon: ExternalLink,
											onclick: () => goto(`${basePath}/lists/${list.id}/subscribers`)
										},
										{ divider: true },
										{
											label: 'Delete',
											icon: Trash2,
											variant: 'danger',
											onclick: () => confirmDelete(list)
										}
									]}
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if searchQuery && filteredLists.length === 0}
			<div class="text-center py-8 text-text-muted">
				No lists match "{searchQuery}"
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete List"
	description={`Delete "${deleteListItem?.name || ''}"? This cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => showDeleteConfirm = false}
/>
