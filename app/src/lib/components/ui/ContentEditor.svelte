<script lang="ts">
	import { goto } from '$app/navigation';
	import { Button, Input, Select, MarkdownEditor, ContentChatPanel, Textarea, LoadingSpinner, Alert, Modal, SaveButton } from '$lib/components/ui';
	import { Save, Trash2, ChevronDown, ChevronUp, ArrowLeft, Sparkles, Plus, FolderPlus, Layout } from '@lucide/svelte';

	type ContentType = 'post' | 'page';

	interface Category {
		id: string;
		name: string;
		slug: string;
	}

	interface Template {
		slug: string;
		name: string;
	}

	interface Props {
		type: ContentType;
		isNew?: boolean;
		title: string;
		backUrl: string;
		loading?: boolean;
		saving?: boolean;
		saved?: boolean;
		deleting?: boolean;
		error?: string;
		// Post-specific
		categories?: Category[];
		// Page-specific
		templates?: Template[];
		// Form data binding
		formData: {
			title: string;
			slug: string;
			content: string;
			status: string;
			scheduled_at?: string;
			meta_title?: string;
			meta_description?: string;
			featured_image?: string;
			// Post-specific
			excerpt?: string;
			category_id?: string;
			// Page-specific
			template?: string;
		};
		onSave: () => void;
		onDelete?: () => void;
		onCancel: () => void;
		onCreateCategory?: (name: string, slug: string) => Promise<void>;
		onCreateTemplate?: (name: string, slug: string) => Promise<void>;
	}

	let {
		type,
		isNew = false,
		title,
		backUrl,
		loading = false,
		saving = false,
		saved = false,
		deleting = false,
		error = '',
		categories = [],
		templates = [],
		formData = $bindable(),
		onSave,
		onDelete,
		onCancel,
		onCreateCategory,
		onCreateTemplate
	}: Props = $props();

	// UI state
	let showSeoFields = $state(false);
	let showAiAssistant = $state(true);
	let showDeleteConfirm = $state(false);

	// Category modal state (posts)
	let showCategoryModal = $state(false);
	let creatingCategory = $state(false);
	let newCategoryName = $state('');
	let newCategorySlug = $state('');
	let previousCategoryId = $state('');

	// Template modal state (pages)
	let showTemplateModal = $state(false);
	let creatingTemplate = $state(false);
	let newTemplateName = $state('');
	let newTemplateSlug = $state('');
	let previousTemplate = $state('default');

	// Status options
	const STATUS_OPTIONS = $derived(type === 'post'
		? [
			{ value: 'draft', label: 'Draft' },
			{ value: 'published', label: 'Published' },
			{ value: 'scheduled', label: 'Scheduled' }
		]
		: [
			{ value: 'draft', label: 'Draft' },
			{ value: 'published', label: 'Published' }
		]);

	// Category options with "create new"
	let categoryOptions = $derived([
		{ value: '', label: '-- Select Category --' },
		...categories.map(c => ({ value: c.id, label: c.name })),
		{ value: '__new__', label: '+ Create new category' }
	]);

	// Template options with "create new"
	let templateOptions = $derived([
		...templates.map(t => ({ value: t.slug, label: t.name })),
		{ value: '__new__', label: '+ Create new template' }
	]);

	function generateSlug(text: string): string {
		return text
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-|-$/g, '');
	}

	function handleTitleInput(e: Event) {
		const titleValue = (e.target as HTMLInputElement).value;
		// Auto-generate slug from title only for new items
		if (isNew && (!formData.slug || formData.slug === generateSlug(formData.title))) {
			formData.slug = generateSlug(titleValue);
		}
	}

	function insertContent(content: string) {
		if (formData.content) {
			formData.content += '\n\n' + content;
		} else {
			formData.content = content;
		}
	}

	// Watch for "create new category" selection
	$effect(() => {
		if (type === 'post' && formData.category_id === '__new__') {
			formData.category_id = previousCategoryId;
			openCategoryModal();
		} else if (type === 'post' && formData.category_id !== undefined) {
			previousCategoryId = formData.category_id;
		}
	});

	// Watch for "create new template" selection
	$effect(() => {
		if (type === 'page' && formData.template === '__new__') {
			formData.template = previousTemplate;
			openTemplateModal();
		} else if (type === 'page' && formData.template) {
			previousTemplate = formData.template;
		}
	});

	function openCategoryModal() {
		newCategoryName = '';
		newCategorySlug = '';
		showCategoryModal = true;
	}

	function openTemplateModal() {
		newTemplateName = '';
		newTemplateSlug = '';
		showTemplateModal = true;
	}

	function handleCategoryNameInput(e: Event) {
		const name = (e.target as HTMLInputElement).value;
		newCategorySlug = generateSlug(name);
	}

	function handleTemplateNameInput(e: Event) {
		const name = (e.target as HTMLInputElement).value;
		newTemplateSlug = generateSlug(name);
	}

	async function createCategory() {
		if (!newCategoryName || !newCategorySlug || !onCreateCategory) return;
		creatingCategory = true;
		try {
			await onCreateCategory(newCategoryName, newCategorySlug);
			showCategoryModal = false;
		} finally {
			creatingCategory = false;
		}
	}

	async function createTemplate() {
		if (!newTemplateName || !newTemplateSlug || !onCreateTemplate) return;
		creatingTemplate = true;
		try {
			await onCreateTemplate(newTemplateName, newTemplateSlug);
			showTemplateModal = false;
		} finally {
			creatingTemplate = false;
		}
	}

	function handleDelete() {
		showDeleteConfirm = false;
		if (onDelete) onDelete();
	}
</script>

<svelte:head>
	<title>{title} - Outlet</title>
</svelte:head>

{#if loading}
	<div class="fixed inset-0 z-50 bg-gray-100 flex items-center justify-center">
		<LoadingSpinner size="large" />
	</div>
{:else}
	<!-- Full-screen editor -->
	<div class="fixed inset-0 z-50 bg-gray-100">
		<form onsubmit={(e) => { e.preventDefault(); onSave(); }} class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between flex-shrink-0">
				<div class="flex items-center gap-4">
					<a href={backUrl} class="text-gray-500 hover:text-gray-700">
						<ArrowLeft size={20} />
					</a>
					<h3 class="text-lg font-semibold text-gray-900">
						{title}
					</h3>
				</div>
				<div class="flex items-center gap-4">
					{#if !isNew && onDelete}
						<button
							type="button"
							onclick={() => showDeleteConfirm = true}
							disabled={saving}
							class="inline-flex items-center rounded-md bg-white px-3 py-2 text-sm font-semibold text-red-600 shadow-sm ring-1 ring-inset ring-red-300 hover:bg-red-50 disabled:opacity-50"
						>
							<Trash2 size={16} class="mr-2" />
							Delete
						</button>
					{/if}
					<Button type="secondary" onclick={onCancel} disabled={saving}>
						Cancel
					</Button>
					<SaveButton
						{saving}
						{saved}
						label={isNew ? 'Save' : 'Save Changes'}
						onclick={onSave}
					/>
				</div>
			</div>

			<!-- Content -->
			<div class="flex-1 overflow-hidden flex">
				<!-- Left Sidebar - Settings & Chat -->
				<div class="w-96 bg-white border-r border-gray-200 flex flex-col flex-shrink-0">
					<!-- Scrollable settings area -->
					<div class="overflow-y-auto p-6 space-y-5 flex-shrink-0">
						{#if error}
							<Alert type="error" title="Error">
								<p>{error}</p>
							</Alert>
						{/if}

						<!-- Title -->
						<div>
							<label for="content-title" class="block text-sm font-medium text-gray-700">Title</label>
							<Input
								id="content-title"
								type="text"
								bind:value={formData.title}
								oninput={handleTitleInput}
								placeholder={type === 'post' ? 'My Awesome Post' : 'About Us'}
							/>
						</div>

						<!-- Slug -->
						<div>
							<label for="content-slug" class="block text-sm font-medium text-gray-700">Slug</label>
							<Input
								id="content-slug"
								type="text"
								bind:value={formData.slug}
								placeholder={type === 'post' ? 'my-awesome-post' : 'about-us'}
							/>
							<p class="mt-1 text-xs text-gray-500">
								URL: {type === 'post' ? '/blog/' : '/'}{formData.slug || (type === 'post' ? 'my-awesome-post' : 'about-us')}
							</p>
						</div>

						<!-- Category (posts only) -->
						{#if type === 'post'}
							<div>
								<div class="flex items-center justify-between mb-1">
									<label for="content-category" class="block text-sm font-medium text-gray-700">Category</label>
									{#if categories.length > 0}
										<button
											type="button"
											onclick={openCategoryModal}
											class="text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
										>
											<Plus size={12} />
											New
										</button>
									{/if}
								</div>
								{#if categories.length === 0}
									<button
										type="button"
										onclick={openCategoryModal}
										class="w-full flex items-center justify-center gap-2 px-4 py-3 border-2 border-dashed border-gray-300 rounded-xl text-sm text-gray-500 hover:border-blue-400 hover:text-blue-600 hover:bg-blue-50/50 transition-colors"
									>
										<FolderPlus size={16} />
										Create your first category
									</button>
								{:else}
									<Select
										id="content-category"
										bind:value={formData.category_id}
										options={categoryOptions}
									/>
								{/if}
							</div>
						{/if}

						<!-- Template (pages only) -->
						{#if type === 'page'}
							<div>
								<div class="flex items-center justify-between mb-1">
									<label for="content-template" class="block text-sm font-medium text-gray-700">Template</label>
									{#if templates.length > 0}
										<button
											type="button"
											onclick={openTemplateModal}
											class="text-xs text-blue-600 hover:text-blue-700 flex items-center gap-1"
										>
											<Plus size={12} />
											New
										</button>
									{/if}
								</div>
								{#if templates.length === 0}
									<button
										type="button"
										onclick={openTemplateModal}
										class="w-full flex items-center justify-center gap-2 px-4 py-3 border-2 border-dashed border-gray-300 rounded-xl text-sm text-gray-500 hover:border-blue-400 hover:text-blue-600 hover:bg-blue-50/50 transition-colors"
									>
										<Layout size={16} />
										Create your first template
									</button>
								{:else}
									<Select
										id="content-template"
										bind:value={formData.template}
										options={templateOptions}
									/>
									{#if formData.template}
										<p class="mt-1 text-xs text-gray-500">Template slug sent via SDK: <code class="bg-gray-100 px-1 rounded">{formData.template}</code></p>
									{/if}
								{/if}
							</div>
						{/if}

						<!-- Status -->
						<div>
							<label for="content-status" class="block text-sm font-medium text-gray-700">Status</label>
							<Select
								id="content-status"
								bind:value={formData.status}
								options={STATUS_OPTIONS}
							/>
							{#if formData.status === 'scheduled'}
								<div class="mt-2">
									<label for="scheduled-at" class="block text-xs font-medium text-gray-600 mb-1">Scheduled Date & Time</label>
									<Input
										id="scheduled-at"
										type="datetime-local"
										bind:value={formData.scheduled_at}
									/>
								</div>
							{/if}
						</div>

						<!-- Excerpt (posts only) -->
						{#if type === 'post'}
							<div>
								<label for="content-excerpt" class="block text-sm font-medium text-gray-700">Excerpt</label>
								<Textarea
									id="content-excerpt"
									bind:value={formData.excerpt}
									placeholder="A brief summary of your post..."
									rows={3}
								/>
							</div>
						{/if}

						<!-- SEO Section -->
						<div class="border-t border-gray-200 pt-4">
							<button
								type="button"
								onclick={() => showSeoFields = !showSeoFields}
								class="flex items-center justify-between w-full text-left"
							>
								<span class="text-sm font-medium text-gray-700">SEO Settings</span>
								{#if showSeoFields}
									<ChevronUp size={16} class="text-gray-400" />
								{:else}
									<ChevronDown size={16} class="text-gray-400" />
								{/if}
							</button>

							{#if showSeoFields}
								<div class="mt-4 space-y-4">
									<div>
										<label for="meta-title" class="block text-sm font-medium text-gray-700">Meta Title</label>
										<Input
											id="meta-title"
											type="text"
											bind:value={formData.meta_title}
											placeholder="Custom title for search engines"
										/>
										<p class="mt-1 text-xs text-gray-500">
											{formData.meta_title?.length || 0}/60 characters
										</p>
									</div>

									<div>
										<label for="meta-desc" class="block text-sm font-medium text-gray-700">Meta Description</label>
										<Textarea
											id="meta-desc"
											bind:value={formData.meta_description}
											placeholder="Description for search engine results"
											rows={3}
										/>
										<p class="mt-1 text-xs text-gray-500">
											{formData.meta_description?.length || 0}/160 characters
										</p>
									</div>

									<div>
										<label for="featured-image" class="block text-sm font-medium text-gray-700">Featured Image URL</label>
										<Input
											id="featured-image"
											type="text"
											bind:value={formData.featured_image}
											placeholder="https://..."
										/>
									</div>
								</div>
							{/if}
						</div>
					</div>

					<!-- Spacer when AI is collapsed - pushes toggle to bottom -->
					{#if !showAiAssistant}
						<div class="flex-1"></div>
					{/if}

					<!-- AI Chat Panel - Collapsible (at bottom when collapsed) -->
					<div class="border-t border-gray-200 {showAiAssistant ? 'flex-1 min-h-0 flex flex-col' : ''}">
						<button
							type="button"
							onclick={() => showAiAssistant = !showAiAssistant}
							class="w-full flex items-center justify-between px-4 py-3 bg-gray-50 hover:bg-gray-100 transition-colors flex-shrink-0"
						>
							<div class="flex items-center gap-2">
								<Sparkles size={16} class="text-blue-600" />
								<span class="font-medium text-gray-900 text-sm">AI Assistant</span>
							</div>
							{#if showAiAssistant}
								<ChevronDown size={16} class="text-gray-400" />
							{:else}
								<ChevronUp size={16} class="text-gray-400" />
							{/if}
						</button>
						{#if showAiAssistant}
							<div class="flex-1 min-h-0">
								<ContentChatPanel
									contentContext={formData.content}
									onInsertContent={insertContent}
									hideHeader={true}
									class="h-full"
								/>
							</div>
						{/if}
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col overflow-hidden p-6">
					<MarkdownEditor
						bind:value={formData.content}
						placeholder={type === 'post' ? 'Start writing your post...' : 'Start writing your page content...'}
						class="h-full"
					/>
				</div>
			</div>
		</form>
	</div>
{/if}

<!-- Delete Confirmation Modal -->
{#if !isNew && onDelete}
	<Modal bind:show={showDeleteConfirm} title="Delete {type === 'post' ? 'Post' : 'Page'}">
		<div class="flex items-start gap-4">
			<div class="flex-shrink-0 w-10 h-10 rounded-full bg-red-100 flex items-center justify-center">
				<Trash2 size={20} class="text-red-600" />
			</div>
			<div>
				<p class="text-gray-900">
					Are you sure you want to delete "<span class="font-medium">{formData.title}</span>"?
				</p>
				<p class="mt-2 text-sm text-gray-500">
					This action cannot be undone. The {type} will be permanently removed.
				</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex justify-end gap-3">
				<Button type="secondary" onclick={() => showDeleteConfirm = false} disabled={deleting}>
					Cancel
				</Button>
				<Button type="danger" onclick={handleDelete} disabled={deleting}>
					{deleting ? 'Deleting...' : `Delete ${type === 'post' ? 'Post' : 'Page'}`}
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- Create Category Modal (posts) -->
{#if type === 'post'}
	<Modal bind:show={showCategoryModal} title="Create Category">
		<div class="space-y-4">
			<div>
				<label for="category-name" class="block text-sm font-medium text-gray-700">Name</label>
				<Input
					id="category-name"
					type="text"
					bind:value={newCategoryName}
					oninput={handleCategoryNameInput}
					placeholder="Technology"
				/>
			</div>
			<div>
				<label for="category-slug" class="block text-sm font-medium text-gray-700">Slug</label>
				<Input
					id="category-slug"
					type="text"
					bind:value={newCategorySlug}
					placeholder="technology"
				/>
				<p class="mt-1 text-xs text-gray-500">URL: /blog/category/{newCategorySlug || 'technology'}</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex justify-end gap-3">
				<Button type="secondary" onclick={() => showCategoryModal = false} disabled={creatingCategory}>
					Cancel
				</Button>
				<Button type="primary" onclick={createCategory} disabled={!newCategoryName || !newCategorySlug || creatingCategory}>
					{creatingCategory ? 'Creating...' : 'Create Category'}
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}

<!-- Create Template Modal (pages) -->
{#if type === 'page'}
	<Modal bind:show={showTemplateModal} title="Create Template">
		<div class="space-y-4">
			<div>
				<label for="template-name" class="block text-sm font-medium text-gray-700">Name</label>
				<Input
					id="template-name"
					type="text"
					bind:value={newTemplateName}
					oninput={handleTemplateNameInput}
					placeholder="Landing Page"
				/>
			</div>
			<div>
				<label for="template-slug" class="block text-sm font-medium text-gray-700">Slug</label>
				<Input
					id="template-slug"
					type="text"
					bind:value={newTemplateSlug}
					placeholder="landing-page"
				/>
				<p class="mt-1 text-xs text-gray-500">This slug will be sent via SDK so your system knows which template to use.</p>
			</div>
		</div>

		{#snippet footer()}
			<div class="flex justify-end gap-3">
				<Button type="secondary" onclick={() => showTemplateModal = false} disabled={creatingTemplate}>
					Cancel
				</Button>
				<Button type="primary" onclick={createTemplate} disabled={!newTemplateName || !newTemplateSlug || creatingTemplate}>
					{creatingTemplate ? 'Creating...' : 'Create Template'}
				</Button>
			</div>
		{/snippet}
	</Modal>
{/if}
