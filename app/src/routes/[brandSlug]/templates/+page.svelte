<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		listEmailDesigns,
		createEmailDesign,
		deleteEmailDesign,
		type EmailDesignInfo
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
		Select,
		AlertDialog
	} from '$lib/components/ui';
	import { Plus, FileText, MoreVertical, Trash2, Edit, Copy } from '@lucide/svelte';

	let loading = $state(true);
	let templates = $state<EmailDesignInfo[]>([]);
	let error = $state('');
	let searchQuery = $state('');

	// Create form
	let showCreateForm = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newSlug = $state('');
	let newDescription = $state('');
	let newCategory = $state('general');

	// Build base path with brandSlug
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// Filtered templates based on search
	let filteredTemplates = $derived(
		templates.filter(t =>
			t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
			t.slug.toLowerCase().includes(searchQuery.toLowerCase()) ||
			t.category.toLowerCase().includes(searchQuery.toLowerCase())
		)
	);

	// Category options
	const categories = [
		{ value: 'general', label: 'General' },
		{ value: 'newsletter', label: 'Newsletter' },
		{ value: 'promotional', label: 'Promotional' },
		{ value: 'transactional', label: 'Transactional' }
	];

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';

		try {
			const response = await listEmailDesigns({});
			templates = response.designs || [];
		} catch (err) {
			console.error('Failed to load templates:', err);
			error = 'Failed to load templates';
		} finally {
			loading = false;
		}
	}

	function openCreateForm() {
		newName = '';
		newSlug = '';
		newDescription = '';
		newCategory = 'general';
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
			await createEmailDesign({
				name: newName,
				slug: newSlug,
				description: newDescription,
				category: newCategory,
				html_body: '<p>Start editing your template here...</p>',
				is_active: true
			});
			closeCreateForm();
			await loadData();
		} catch (err: any) {
			console.error('Failed to create template:', err);
			error = err.message || 'Failed to create template';
		} finally {
			creating = false;
		}
	}

	let showDeleteConfirm = $state(false);
	let deleteTemplateItem = $state<EmailDesignInfo | null>(null);
	let deleting = $state(false);

	function confirmDelete(template: EmailDesignInfo) {
		deleteTemplateItem = template;
		showDeleteConfirm = true;
	}

	async function executeDelete() {
		if (!deleteTemplateItem) return;
		deleting = true;
		try {
			await deleteEmailDesign({}, deleteTemplateItem.id);
			await loadData();
		} catch (err: any) {
			console.error('Failed to delete template:', err);
			error = err.message || 'Failed to delete template';
		} finally {
			deleting = false;
		}
	}

	function getCategoryVariant(category: string): 'default' | 'info' | 'success' | 'warning' {
		switch (category) {
			case 'newsletter': return 'info';
			case 'promotional': return 'warning';
			case 'transactional': return 'success';
			default: return 'default';
		}
	}

	function getCategoryLabel(category: string): string {
		const found = categories.find(c => c.value === category);
		return found?.label || category;
	}
</script>

<svelte:head>
	<title>Templates | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	<!-- Header -->
	<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
		<div>
			<h1 class="text-2xl font-semibold text-text">Templates</h1>
			<p class="mt-1 text-sm text-text-muted">Reusable email designs for campaigns and sequences</p>
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

	<!-- Inline Create Form -->
	<SlideForm bind:show={showCreateForm} title="Create New Template">
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
			<div>
				<label for="template-name" class="form-label">Name</label>
				<Input
					id="template-name"
					type="text"
					bind:value={newName}
					oninput={handleNameInput}
					placeholder="Welcome Email"
				/>
			</div>
			<div>
				<label for="template-slug" class="form-label">Slug</label>
				<Input
					id="template-slug"
					type="text"
					bind:value={newSlug}
					placeholder="welcome-email"
				/>
			</div>
			<div>
				<label for="template-category" class="form-label">Category</label>
				<Select
					id="template-category"
					bind:value={newCategory}
					options={categories}
				/>
			</div>
			<div>
				<label for="template-description" class="form-label">Description (optional)</label>
				<Input
					id="template-description"
					type="text"
					bind:value={newDescription}
					placeholder="Main welcome email for new subscribers"
				/>
			</div>
		</div>

		{#snippet footer()}
			<Button type="secondary" onclick={closeCreateForm} disabled={creating}>
				Cancel
			</Button>
			<Button type="primary" onclick={submitCreate} disabled={!newName || !newSlug || creating}>
				{creating ? 'Creating...' : 'Create Template'}
			</Button>
		{/snippet}
	</SlideForm>

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if templates.length === 0}
		<EmptyState
			icon={FileText}
			title="No templates yet"
			description="Create reusable email templates for your campaigns and sequences."
		>
			<Button type="primary" onclick={openCreateForm}>
				<Plus class="mr-2 h-4 w-4" />
				Create Template
			</Button>
		</EmptyState>
	{:else}
		<!-- Search -->
		{#if templates.length > 3}
			<div class="mb-4">
				<SearchInput
					bind:value={searchQuery}
					placeholder="Search templates..."
				/>
			</div>
		{/if}

		<!-- Templates Table -->
		<div class="data-table">
			<table class="w-full">
				<thead>
					<tr>
						<th class="text-left">Template</th>
						<th class="text-left">Category</th>
						<th class="text-center">Status</th>
						<th class="text-right w-10"></th>
					</tr>
				</thead>
				<tbody>
					{#each filteredTemplates as template}
						<tr>
							<td>
								<a href="{basePath}/templates/{template.id}" class="block group">
									<div class="flex items-center gap-3">
										<div class="data-table-icon">
											<FileText class="h-4 w-4" />
										</div>
										<div class="min-w-0 flex-1">
											<span class="font-medium text-text group-hover:text-primary transition-colors block truncate">
												{template.name}
											</span>
											{#if template.description}
												<span class="text-sm text-text-muted block truncate">{template.description}</span>
											{:else}
												<span class="text-sm text-text-muted">{template.slug}</span>
											{/if}
										</div>
									</div>
								</a>
							</td>
							<td>
								<Badge variant={getCategoryVariant(template.category)} size="sm">
									{getCategoryLabel(template.category)}
								</Badge>
							</td>
							<td class="text-center">
								{#if template.is_active}
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
										{
											label: 'Edit',
											icon: Edit,
											onclick: () => goto(`${basePath}/templates/${template.id}`)
										},
										{
											label: 'Duplicate',
											icon: Copy,
											onclick: () => goto(`${basePath}/templates/new?duplicate=${template.id}`)
										},
										{ divider: true },
										{
											label: 'Delete',
											icon: Trash2,
											variant: 'danger',
											onclick: () => confirmDelete(template)
										}
									]}
								/>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>

		{#if searchQuery && filteredTemplates.length === 0}
			<div class="text-center py-8 text-text-muted">
				No templates match "{searchQuery}"
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Template"
	description={`Delete "${deleteTemplateItem?.name || ''}"? This cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => showDeleteConfirm = false}
/>
