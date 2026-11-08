<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		listTransactionalEmails,
		createTransactionalEmail,
		deleteTransactionalEmail,
		type TransactionalEmailInfo
	} from '$lib/api';
	import {
		Button,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		SearchInput,
		DropdownMenu,
		SlideForm,
		Input,
		AlertDialog
	} from '$lib/components/ui';
	import { Plus, Zap, MoreVertical, Trash2, Edit } from '@lucide/svelte';

	let loading = $state(true);
	let templates = $state<TransactionalEmailInfo[]>([]);
	let error = $state('');
	let searchQuery = $state('');

	let showCreateForm = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newSlug = $state('');
	let newSubject = $state('');
	let slugEdited = $state(false);

	let basePath = $derived(`/${$page.params.brandSlug}`);

	// The ad-hoc template records API sends made without a template; it is not
	// something to edit
	let visible = $derived(templates.filter((t) => t.slug !== '_adhoc'));
	let filtered = $derived(
		visible.filter((t) => {
			const q = searchQuery.toLowerCase();
			return t.name.toLowerCase().includes(q) || t.slug.includes(q) || t.subject.toLowerCase().includes(q);
		})
	);

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const response = await listTransactionalEmails();
			templates = response.templates || [];
		} catch (err: any) {
			error = err.message || 'Failed to load templates';
		} finally {
			loading = false;
		}
	}

	function openCreateForm() {
		newName = '';
		newSlug = '';
		newSubject = '';
		slugEdited = false;
		showCreateForm = true;
	}

	function slugify(name: string): string {
		return name
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	async function submitCreate() {
		if (!newName || !newSlug || !newSubject) return;
		creating = true;
		error = '';
		try {
			const created = await createTransactionalEmail({
				name: newName,
				slug: newSlug,
				subject: newSubject,
				html_body: '<p>Hi {{name}},</p>\n<p>Write your email here.</p>',
				is_active: true
			});
			goto(`${basePath}/transactional/${created.id}`);
		} catch (err: any) {
			error = err.message || 'Failed to create template';
		} finally {
			creating = false;
		}
	}

	let showDeleteConfirm = $state(false);
	let deleteItem = $state<TransactionalEmailInfo | null>(null);

	async function executeDelete() {
		if (!deleteItem) return;
		try {
			await deleteTransactionalEmail({}, deleteItem.id);
			showDeleteConfirm = false;
			await loadData();
		} catch (err: any) {
			error = err.message || 'Failed to delete template';
		}
	}
</script>

<svelte:head>
	<title>Transactional | Outlet</title>
</svelte:head>

<div class="p-4 sm:p-6 max-w-5xl mx-auto">
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<h1 class="text-2xl font-semibold text-text">Transactional</h1>
			<p class="mt-1 text-sm text-text-muted">
				Emails your apps send through the API or SMTP: receipts, password resets, notifications
			</p>
		</div>
		<Button type="primary" onclick={openCreateForm} disabled={showCreateForm}>
			<Plus class="mr-2 h-4 w-4" />
			New Template
		</Button>
	</div>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	<SlideForm bind:show={showCreateForm} title="New transactional template">
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
			<div>
				<label for="tx-name" class="form-label">Name</label>
				<Input
					id="tx-name"
					bind:value={newName}
					oninput={() => {
						if (!slugEdited) newSlug = slugify(newName);
					}}
					placeholder="Password reset"
				/>
			</div>
			<div>
				<label for="tx-slug" class="form-label">Slug</label>
				<Input id="tx-slug" bind:value={newSlug} oninput={() => (slugEdited = true)} placeholder="password-reset" />
				<p class="mt-1 text-xs text-text-muted">Your app sends it with <code>template_slug</code></p>
			</div>
			<div class="sm:col-span-2">
				<label for="tx-subject" class="form-label">Subject</label>
				<Input id="tx-subject" bind:value={newSubject} placeholder={'Reset your password, {{name}}'} />
			</div>
		</div>

		{#snippet footer()}
			<Button type="secondary" onclick={() => (showCreateForm = false)} disabled={creating}>Cancel</Button>
			<Button type="primary" onclick={submitCreate} disabled={!newName || !newSlug || !newSubject || creating}>
				{creating ? 'Creating...' : 'Create and edit'}
			</Button>
		{/snippet}
	</SlideForm>

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if visible.length === 0}
		<EmptyState
			icon={Zap}
			title="No transactional templates yet"
			description="Create a template, then send it from your app with its slug and your brand's API key."
		>
			<Button type="primary" onclick={openCreateForm}>
				<Plus class="mr-2 h-4 w-4" />
				Create Template
			</Button>
		</EmptyState>
	{:else}
		{#if visible.length > 3}
			<div class="mb-4">
				<SearchInput bind:value={searchQuery} placeholder="Search by name, slug or subject..." />
			</div>
		{/if}

		<div class="data-table overflow-x-auto">
			<table class="w-full">
				<thead>
					<tr>
						<th class="text-left">Template</th>
						<th class="text-left hidden sm:table-cell">Slug</th>
						<th class="text-center">Status</th>
						<th class="text-right w-10"></th>
					</tr>
				</thead>
				<tbody>
					{#each filtered as t (t.id)}
						<tr>
							<td>
								<a href="{basePath}/transactional/{t.id}" class="block group min-w-0">
									<span class="font-medium text-text group-hover:text-primary transition-colors block truncate">{t.name}</span>
									<span class="text-sm text-text-muted block truncate">{t.subject}</span>
								</a>
							</td>
							<td class="hidden sm:table-cell"><code class="text-sm">{t.slug}</code></td>
							<td class="text-center">
								{#if t.is_active}
									<Badge variant="success" size="sm">Active</Badge>
								{:else}
									<Badge variant="default" size="sm">Inactive</Badge>
								{/if}
							</td>
							<td class="text-right">
								<DropdownMenu
									trigger={{
										icon: MoreVertical,
										class: 'p-1.5 rounded-lg hover:bg-bg-secondary transition-colors text-text-muted hover:text-text'
									}}
									items={[
										{ label: 'Edit', icon: Edit, onclick: () => goto(`${basePath}/transactional/${t.id}`) },
										{ divider: true },
										{
											label: 'Delete',
											icon: Trash2,
											variant: 'danger',
											onclick: () => {
												deleteItem = t;
												showDeleteConfirm = true;
											}
										}
									]}
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if searchQuery && filtered.length === 0}
			<div class="text-center py-8 text-text-muted">No templates match "{searchQuery}"</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete template"
	description={`Delete "${deleteItem?.name || ''}"? Apps that send "${deleteItem?.slug || ''}" will get an error until you create it again.`}
	actionLabel="Delete"
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => (showDeleteConfirm = false)}
/>
