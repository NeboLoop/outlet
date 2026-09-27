<script lang="ts">
	import { page } from '$app/stores';
	import {
		listCustomFields,
		createCustomField,
		updateCustomField,
		deleteCustomField,
		type CustomFieldInfo
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
		Select,
		Modal,
		AlertDialog
	} from '$lib/components/ui';
	import {
		ListPlus,
		Plus,
		Edit,
		Trash2
	} from '@lucide/svelte';
	import { getListContext } from '../listContext';

	const ctx = getListContext();
	let listId = $derived($page.params.id!);

	// State
	let customFields = $state<CustomFieldInfo[]>([]);
	let customFieldsLoading = $state(true);
	let showCreateField = $state(false);
	let creatingField = $state(false);
	let newFieldName = $state('');
	let newFieldKey = $state('');
	let newFieldType = $state('text');
	let newFieldRequired = $state(false);
	let newFieldPlaceholder = $state('');
	let newFieldOptions = $state('');
	let editingField = $state<CustomFieldInfo | null>(null);
	let deletingFieldId = $state<string | null>(null);
	let error = $state('');

	$effect(() => {
		loadCustomFields();
	});

	async function loadCustomFields() {
		customFieldsLoading = true;
		try {
			const response = await listCustomFields({}, listId);
			customFields = response.fields || [];
		} catch (err) {
			console.error('Failed to load custom fields:', err);
		} finally {
			customFieldsLoading = false;
		}
	}

	function generateFieldKey(name: string): string {
		return name.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_|_$/g, '');
	}

	function handleFieldNameChange() {
		if (!editingField) {
			newFieldKey = generateFieldKey(newFieldName);
		}
	}

	function resetFieldForm() {
		newFieldName = '';
		newFieldKey = '';
		newFieldType = 'text';
		newFieldRequired = false;
		newFieldPlaceholder = '';
		newFieldOptions = '';
		editingField = null;
	}

	function openEditField(field: CustomFieldInfo) {
		editingField = field;
		newFieldName = field.name;
		newFieldKey = field.field_key;
		newFieldType = field.field_type;
		newFieldRequired = field.required;
		newFieldPlaceholder = field.placeholder || '';
		newFieldOptions = field.options?.join(', ') || '';
		showCreateField = true;
	}

	async function handleSaveField() {
		if (!newFieldName || !newFieldKey) return;
		creatingField = true;
		error = '';
		try {
			const options = newFieldType === 'dropdown' && newFieldOptions
				? newFieldOptions.split(',').map(o => o.trim()).filter(o => o)
				: undefined;

			if (editingField) {
				await updateCustomField({}, {
					name: newFieldName,
					field_key: newFieldKey,
					field_type: newFieldType,
					required: newFieldRequired,
					placeholder: newFieldPlaceholder,
					options
				}, listId, editingField.id);
			} else {
				await createCustomField({}, {
					name: newFieldName,
					field_key: newFieldKey,
					field_type: newFieldType,
					required: newFieldRequired,
					placeholder: newFieldPlaceholder,
					options
				}, listId);
			}
			showCreateField = false;
			resetFieldForm();
			await loadCustomFields();
		} catch (err: any) {
			error = err.message || 'Failed to save custom field';
		} finally {
			creatingField = false;
		}
	}

	let showDeleteFieldConfirm = $state(false);
	let deleteFieldId = $state('');

	function confirmDeleteField(fieldId: string) {
		deleteFieldId = fieldId;
		showDeleteFieldConfirm = true;
	}

	async function executeDeleteField() {
		deletingFieldId = deleteFieldId;
		try {
			await deleteCustomField({}, listId, deleteFieldId);
			await loadCustomFields();
		} catch (err: any) {
			error = err.message || 'Failed to delete custom field';
		} finally {
			deletingFieldId = null;
		}
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

<div class="space-y-4">
	<div class="flex justify-between items-center">
		<div>
			<h3 class="font-medium text-text">Custom Fields</h3>
			<p class="text-sm text-text-muted">Add custom fields to collect additional information from subscribers</p>
		</div>
		<Button type="primary" onclick={() => { resetFieldForm(); showCreateField = true; }}>
			<Plus class="mr-2 h-4 w-4" />
			Add Field
		</Button>
	</div>

	{#if customFieldsLoading}
		<div class="flex justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if customFields.length === 0}
		<EmptyState
			icon={ListPlus}
			title="No custom fields"
			message="Add custom fields to collect more information from subscribers, like name, company, or phone number."
		>
			<Button type="primary" onclick={() => { resetFieldForm(); showCreateField = true; }}>
				<Plus class="mr-2 h-4 w-4" />
				Add Your First Field
			</Button>
		</EmptyState>
	{:else}
		<Card>
			<div class="overflow-x-auto">
				<table class="table w-full">
					<thead>
						<tr>
							<th>Name</th>
							<th>Merge Tag</th>
							<th>Type</th>
							<th>Required</th>
							<th class="text-right">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each customFields as field}
							<tr>
								<td class="font-medium">{field.name}</td>
								<td>
									<code class="text-xs bg-bg-secondary px-1.5 py-0.5 rounded">{`{{${field.field_key}}}`}</code>
								</td>
								<td>
									<Badge variant="default" size="sm">{field.field_type}</Badge>
								</td>
								<td>
									{#if field.required}
										<Badge variant="warning" size="sm">Required</Badge>
									{:else}
										<span class="text-text-muted">Optional</span>
									{/if}
								</td>
								<td class="text-right">
									<div class="flex justify-end gap-2">
										<Button type="ghost" size="sm" onclick={() => openEditField(field)}>
											<Edit class="h-4 w-4" />
										</Button>
										<Button
											type="ghost"
											size="sm"
											onclick={() => confirmDeleteField(field.id)}
											disabled={deletingFieldId === field.id}
										>
											{#if deletingFieldId === field.id}
												<LoadingSpinner size="small" />
											{:else}
												<Trash2 class="h-4 w-4 text-red-500" />
											{/if}
										</Button>
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</Card>

		<Card>
			<h4 class="font-medium text-text mb-2">Using Merge Tags</h4>
			<p class="text-sm text-text-muted mb-3">
				Use merge tags in your email templates to personalize content for each subscriber.
			</p>
			<div class="bg-bg-secondary p-3 rounded-lg">
				<code class="text-sm font-mono text-text">
					Hi {`{{name}}`}, thanks for signing up!
				</code>
			</div>
		</Card>
	{/if}
</div>

<!-- Custom Field Modal -->
<Modal
	bind:show={showCreateField}
	title={editingField ? 'Edit Custom Field' : 'Add Custom Field'}
	size="md"
>
	<div class="space-y-4">
		<div>
			<label for="field-name" class="form-label">Field Name</label>
			<Input
				id="field-name"
				type="text"
				bind:value={newFieldName}
				oninput={handleFieldNameChange}
				placeholder="e.g., First Name"
			/>
			<p class="mt-1 text-xs text-text-muted">The label shown on the subscribe form</p>
		</div>

		<div>
			<label for="field-key" class="form-label">Merge Tag Key</label>
			<Input
				id="field-key"
				type="text"
				bind:value={newFieldKey}
				placeholder="e.g., first_name"
			/>
			<p class="mt-1 text-xs text-text-muted">Used in emails as {`{{${newFieldKey || 'field_key'}}}`}</p>
		</div>

		<div>
			<label for="field-type" class="form-label">Field Type</label>
			<Select
				bind:value={newFieldType}
				options={[
					{ value: 'text', label: 'Text' },
					{ value: 'number', label: 'Number' },
					{ value: 'date', label: 'Date' },
					{ value: 'dropdown', label: 'Dropdown' }
				]}
			/>
		</div>

		{#if newFieldType === 'dropdown'}
			<div>
				<label for="field-options" class="form-label">Options</label>
				<Input
					id="field-options"
					type="text"
					bind:value={newFieldOptions}
					placeholder="Option 1, Option 2, Option 3"
				/>
				<p class="mt-1 text-xs text-text-muted">Comma-separated list of options</p>
			</div>
		{/if}

		<div>
			<label for="field-placeholder" class="form-label">Placeholder (optional)</label>
			<Input
				id="field-placeholder"
				type="text"
				bind:value={newFieldPlaceholder}
				placeholder="Enter your first name..."
			/>
		</div>

		<div>
			<Checkbox bind:checked={newFieldRequired} label="Required field" />
			<p class="text-xs text-text-muted mt-1 ml-6">
				Subscribers must fill in this field to subscribe
			</p>
		</div>

		{#if error}
			<Alert type="error" title="Error">
				<p>{error}</p>
			</Alert>
		{/if}
	</div>

	{#snippet footer()}
		<Button type="secondary" onclick={() => { showCreateField = false; resetFieldForm(); }} disabled={creatingField}>
			Cancel
		</Button>
		<Button
			type="primary"
			onclick={handleSaveField}
			disabled={!newFieldName || !newFieldKey || creatingField}
		>
			{#if creatingField}
				Saving...
			{:else if editingField}
				Save Changes
			{:else}
				Add Field
			{/if}
		</Button>
	{/snippet}
</Modal>

<AlertDialog
	bind:open={showDeleteFieldConfirm}
	title="Delete Custom Field"
	description="Are you sure you want to delete this custom field? All data for this field will be lost."
	actionLabel={deletingFieldId ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDeleteField}
	onCancel={() => showDeleteFieldConfirm = false}
/>
