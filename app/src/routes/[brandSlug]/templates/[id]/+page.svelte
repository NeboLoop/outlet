<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getEmailDesign,
		updateEmailDesign,
		deleteEmailDesign,
		type EmailDesignInfo
	} from '$lib/api';
	import {
		Button,
		Input,
		Alert,
		LoadingSpinner,
		Badge,
		Select,
		Toggle,
		AlertDialog,
		EmailEditor,
		PersonalizationTags,
		Textarea
	} from '$lib/components/ui';
	import {
		ArrowLeft,
		Save,
		Trash2,
		Eye,
		X
	} from '@lucide/svelte';

	// Get template ID from URL
	let templateId = $derived($page.params.id);
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// Main state
	let loading = $state(true);
	let template = $state<EmailDesignInfo | null>(null);
	let error = $state('');

	// Edit state
	let editName = $state('');
	let editSlug = $state('');
	let editDescription = $state('');
	let editCategory = $state('general');
	let editHtmlBody = $state('');
	let editPlainText = $state('');
	let editIsActive = $state(true);
	let saving = $state(false);
	let saved = $state(false);

	// Preview state
	let showPreview = $state(false);

	// Delete state
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);

	// Variable insertion
	let insertVariable = $state<((variable: string) => void) | null>(null);

	// Template variables
	const templateVariables = [
		{ name: 'content', label: 'Main content area' },
		{ name: 'email', label: 'Subscriber email' },
		{ name: 'name', label: 'Subscriber name' },
		{ name: 'name,fallback=Friend', label: 'Name with fallback' },
		{ name: 'unsubscribe_url', label: 'Unsubscribe link' },
		{ name: 'web_version_url', label: 'Web version link' }
	];

	// Category options
	const categories = [
		{ value: 'general', label: 'General' },
		{ value: 'newsletter', label: 'Newsletter' },
		{ value: 'promotional', label: 'Promotional' },
		{ value: 'transactional', label: 'Transactional' }
	];

	// Check if there are unsaved changes
	let hasChanges = $derived(
		template !== null && (
			editName !== template.name ||
			editDescription !== (template.description || '') ||
			editCategory !== template.category ||
			editHtmlBody !== template.html_body ||
			editPlainText !== (template.plain_text || '') ||
			editIsActive !== template.is_active
		)
	);

	$effect(() => {
		loadTemplate();
	});

	async function loadTemplate() {
		if (!templateId) return;
		loading = true;
		error = '';
		try {
			template = await getEmailDesign({}, templateId);
			editName = template.name;
			editSlug = template.slug;
			editDescription = template.description || '';
			editCategory = template.category;
			editHtmlBody = template.html_body;
			editPlainText = template.plain_text || '';
			editIsActive = template.is_active;
		} catch (err) {
			console.error('Failed to load template:', err);
			error = 'Failed to load template';
		} finally {
			loading = false;
		}
	}

	async function saveTemplate() {
		if (!template || !templateId || !editName.trim()) return;
		saving = true;
		saved = false;
		error = '';
		try {
			const updated = await updateEmailDesign({}, {
				name: editName.trim(),
				description: editDescription.trim() || undefined,
				category: editCategory,
				html_body: editHtmlBody,
				plain_text: editPlainText || undefined,
				is_active: editIsActive
			}, templateId);
			template = updated;
			saved = true;
			setTimeout(() => { saved = false; }, 2000);
		} catch (err: any) {
			console.error('Failed to save template:', err);
			error = err.message || 'Failed to save template';
		} finally {
			saving = false;
		}
	}

	async function saveAndClose() {
		await saveTemplate();
		if (!error) {
			goto(`${basePath}/templates`);
		}
	}

	async function executeDelete() {
		if (!templateId) return;
		deleting = true;
		try {
			await deleteEmailDesign({}, templateId);
			goto(`${basePath}/templates`);
		} catch (err: any) {
			error = err.message || 'Failed to delete template';
		} finally {
			deleting = false;
		}
	}

	function handleCancel() {
		goto(`${basePath}/templates`);
	}

	function regeneratePlainText() {
		const div = document.createElement('div');
		div.innerHTML = editHtmlBody;
		editPlainText = div.textContent || div.innerText || '';
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
	<title>{template?.name || 'Template'} | Outlet</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="lg" />
	</div>
{:else if !template}
	<div class="p-6">
		<Alert type="error" title="Template not found">
			<p>The requested template could not be found.</p>
			<Button type="primary" class="mt-3" onclick={() => goto(`${basePath}/templates`)}>
				Back to Templates
			</Button>
		</Alert>
	</div>
{:else}
	<!-- Full Screen Layout -->
	<div class="fixed inset-0 z-50 bg-base-200">
		<div class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-base-100 border-b border-base-300 px-6 py-4 flex items-center justify-between flex-shrink-0">
				<div class="flex items-center gap-4">
					<button
						type="button"
						onclick={handleCancel}
						class="p-2 rounded-lg hover:bg-base-200 transition-colors text-base-content/60 hover:text-base-content"
					>
						<ArrowLeft class="h-5 w-5" />
					</button>
					<div>
						<div class="flex items-center gap-2">
							<input
								type="text"
								bind:value={editName}
								placeholder="Template Name"
								class="text-lg font-semibold text-base-content bg-transparent border-none focus:outline-none focus:ring-0 placeholder:text-base-content/40 w-64"
							/>
							<Badge variant={getCategoryVariant(editCategory)} size="sm">
								{getCategoryLabel(editCategory)}
							</Badge>
							{#if editIsActive}
								<Badge variant="success" size="sm">Active</Badge>
							{:else}
								<Badge variant="default" size="sm">Inactive</Badge>
							{/if}
						</div>
						<p class="text-xs text-base-content/50">Email Template</p>
					</div>
				</div>
				<div class="flex items-center gap-3">
					<Button type="secondary" onclick={handleCancel}>
						<X class="mr-2 h-4 w-4" />
						Cancel
					</Button>
					<Button type="secondary" onclick={() => showPreview = true}>
						<Eye class="mr-2 h-4 w-4" />
						Preview
					</Button>
					<Button
						type="primary"
						onclick={saveAndClose}
						disabled={saving || !hasChanges}
					>
						<Save class="mr-2 h-4 w-4" />
						{#if saving}
							Saving...
						{:else if saved}
							Saved!
						{:else}
							Save & Close
						{/if}
					</Button>
				</div>
			</div>

			{#if error}
				<div class="px-6 pt-4">
					<Alert type="error" title="Error" onclose={() => (error = '')}>
						<p>{error}</p>
					</Alert>
				</div>
			{/if}

			<!-- Content -->
			<div class="flex-1 overflow-hidden flex">
				<!-- Left Sidebar - Settings -->
				<div class="w-80 bg-base-100 border-r border-base-300 p-6 overflow-y-auto flex-shrink-0">
					<div class="space-y-6">
						<!-- Slug (readonly) -->
						<div>
							<label class="form-label">Slug</label>
							<p class="text-sm text-base-content/70 bg-base-200 px-3 py-2 rounded-md font-mono">{editSlug}</p>
							<p class="mt-1 text-xs text-base-content/50 italic">Cannot be changed after creation</p>
						</div>

						<!-- Description -->
						<div>
							<label for="template-description" class="form-label">Description</label>
							<Input
								id="template-description"
								type="text"
								bind:value={editDescription}
								placeholder="Optional description"
							/>
						</div>

						<!-- Category -->
						<div>
							<label for="template-category" class="form-label">Category</label>
							<Select
								id="template-category"
								bind:value={editCategory}
								options={categories}
							/>
						</div>

						<!-- Active Toggle -->
						<div class="flex items-center justify-between py-2">
							<span class="text-sm font-medium text-base-content">Active</span>
							<Toggle bind:checked={editIsActive} />
						</div>

						<!-- Personalization Tags -->
						<PersonalizationTags
							variables={templateVariables}
							{insertVariable}
						/>

						<!-- Danger Zone -->
						<div class="border-t border-base-300 pt-6">
							<h4 class="text-sm font-medium text-error mb-3">Danger Zone</h4>
							<p class="text-xs text-base-content/60 mb-3">
								Permanently delete this template. This action cannot be undone.
							</p>
							<Button type="danger" size="sm" onclick={() => showDeleteConfirm = true}>
								<Trash2 class="mr-2 h-4 w-4" />
								Delete Template
							</Button>
						</div>
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col overflow-hidden p-6 gap-4">
					<div class="flex-1 min-h-0">
						<EmailEditor
							bind:value={editHtmlBody}
							placeholder="Start designing your email template..."
							showVariableInserts={false}
							onInsertVariable={(fn) => insertVariable = fn}
							class="h-full"
						/>
					</div>
					<div class="flex-shrink-0">
						<div class="flex items-center justify-between mb-1">
							<label for="plain-text" class="form-label mb-0">Plain Text Version</label>
							<button
								type="button"
								class="text-xs text-primary hover:text-primary/80"
								onclick={regeneratePlainText}
							>
								Regenerate from HTML
							</button>
						</div>
						<Textarea
							id="plain-text"
							bind:value={editPlainText}
							placeholder="Auto-generated from HTML if left empty..."
							rows={4}
						/>
						<p class="mt-1 text-xs text-base-content/50 italic">
							Plain text version for email clients that don't support HTML.
						</p>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<!-- Preview Modal -->
{#if showPreview && template}
	<div class="fixed inset-0 bg-black/50 flex items-center justify-center z-[60] p-4">
		<div class="bg-base-100 rounded-xl shadow-xl max-w-3xl w-full max-h-[90vh] flex flex-col">
			<div class="flex items-center justify-between p-4 border-b border-base-300">
				<h3 class="text-lg font-medium text-base-content">Preview: {editName}</h3>
				<Button type="secondary" onclick={() => showPreview = false}>Close</Button>
			</div>
			<div class="flex-1 overflow-auto p-4 bg-white">
				{@html editHtmlBody}
			</div>
		</div>
	</div>
{/if}

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Template"
	description={`Are you sure you want to delete "${template?.name}"? This action cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => showDeleteConfirm = false}
/>
