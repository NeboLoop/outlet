<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getSequence,
		getList,
		listLists,
		listSequences,
		updateSequence,
		deleteSequence,
		createTemplate,
		updateTemplate,
		deleteTemplate,
		type SequenceInfo,
		type TemplateInfo,
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
		Select,
		AlertDialog,
		Textarea,
		EmailEditor,
		PersonalizationTags
	} from '$lib/components/ui';
	import {
		ArrowLeft,
		Plus,
		Mail,
		Trash2,
		Edit,
		Play,
		Pause,
		GripVertical,
		Clock,
		ArrowRight,
		X,
		Save
	} from '@lucide/svelte';

	// Route params
	let listId = $derived($page.params.id!);
	let seqId = $derived($page.params.seqId!);
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// State
	let loading = $state(true);
	let sequence = $state<SequenceInfo | null>(null);
	let templates = $state<TemplateInfo[]>([]);
	let list = $state<ListInfo | null>(null);
	let allLists = $state<ListInfo[]>([]);
	let allSequences = $state<SequenceInfo[]>([]);
	let error = $state('');

	// Edit sequence state
	let editName = $state('');
	let editTrigger = $state('signup');
	let editActive = $state(true);
	let editListId = $state('');
	let editOnCompletionSequenceId = $state<string>('');
	let saving = $state(false);
	let saved = $state(false);

	// Full-screen email editor state
	let showEmailEditor = $state(false);
	let editingEmail = $state<TemplateInfo | null>(null);
	let editorSubject = $state('');
	let editorDelay = $state(0);
	let editorHtmlBody = $state('');
	let editorPlainText = $state('');
	let editorSaving = $state(false);
	let editorSaved = $state(false);
	let insertVariable = $state<((variable: string) => void) | null>(null);

	// Delete states
	let showDeleteSequence = $state(false);
	let deletingSequence = $state(false);
	let deletingEmailId = $state<string | null>(null);

	// Personalization variables for EmailEditor
	const sequenceVariables = [
		{ name: 'name', label: 'Subscriber name' },
		{ name: 'email', label: 'Subscriber email' },
		{ name: 'name,fallback=Friend', label: 'Name with fallback' }
	];

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';
		try {
			const [seqResponse, listResponse, allSeqResponse, allListsResponse] = await Promise.all([
				getSequence({}, seqId),
				getList({}, listId),
				listSequences(),
				listLists()
			]);
			sequence = seqResponse.sequence;
			templates = (seqResponse.templates || []).sort((a, b) => a.position - b.position);
			list = listResponse;
			allLists = allListsResponse.lists || [];
			// Filter out the current sequence from completion options
			allSequences = (allSeqResponse.sequences || []).filter(s => s.id !== seqId);

			// Set edit state
			editName = sequence.name;
			editTrigger = sequence.trigger_event;
			editActive = sequence.is_active;
			editListId = sequence.list_id || listId;
			editOnCompletionSequenceId = sequence.on_completion_sequence_id || '';
		} catch (err) {
			console.error('Failed to load sequence:', err);
			error = 'Failed to load autoresponder';
		} finally {
			loading = false;
		}
	}

	async function saveSequence() {
		if (!sequence || !editName.trim()) return;
		saving = true;
		saved = false;
		error = '';
		try {
			const listChanged = editListId !== listId;
			const updatedSeq = await updateSequence({}, {
				name: editName.trim(),
				trigger_event: editTrigger,
				is_active: editActive,
				list_id: editListId,
				on_completion_sequence_id: editOnCompletionSequenceId || undefined
			}, seqId);
			sequence.name = editName.trim();
			sequence.trigger_event = editTrigger;
			sequence.is_active = editActive;
			sequence.list_id = editListId;
			sequence.on_completion_sequence_id = editOnCompletionSequenceId || '';
			sequence.on_completion_sequence_name = updatedSeq.on_completion_sequence_name || '';
			saved = true;

			// If list changed, redirect to the new list's sequence page
			if (listChanged) {
				goto(`${basePath}/lists/${editListId}/autoresponders/${seqId}`);
			} else {
				setTimeout(() => { saved = false; }, 2000);
			}
		} catch (err: any) {
			error = err.message || 'Failed to save';
		} finally {
			saving = false;
		}
	}

	async function toggleActive() {
		if (!sequence) return;
		try {
			await updateSequence({}, { is_active: !sequence.is_active }, seqId);
			sequence.is_active = !sequence.is_active;
			editActive = sequence.is_active;
		} catch (err: any) {
			error = err.message || 'Failed to toggle status';
		}
	}

	function openCreateEmail() {
		editingEmail = null;
		editorSubject = '';
		editorDelay = templates.length === 0 ? 0 : 24; // First email immediately, others 24h
		editorHtmlBody = '';
		editorPlainText = '';
		showEmailEditor = true;
	}

	function openEditEmail(template: TemplateInfo) {
		editingEmail = template;
		editorSubject = template.subject;
		editorDelay = template.delay_hours;
		editorHtmlBody = template.html_body;
		editorPlainText = template.plain_text || '';
		showEmailEditor = true;
	}

	async function saveEmailAndClose() {
		if (!editorSubject.trim()) {
			error = 'Subject is required';
			return;
		}

		editorSaving = true;
		editorSaved = false;
		error = '';

		try {
			if (editingEmail) {
				// Update existing email
				await updateTemplate({}, {
					subject: editorSubject.trim(),
					delay_hours: editorDelay,
					html_body: editorHtmlBody || '<p>Email content here</p>',
					plain_text: editorPlainText.trim() || undefined
				}, editingEmail.id);
			} else {
				// Create new email
				await createTemplate({
					sequence_id: seqId,
					subject: editorSubject.trim(),
					delay_hours: editorDelay,
					html_body: editorHtmlBody || '<p>Email content here</p>',
					plain_text: editorPlainText.trim() || undefined,
					is_active: true,
					position: templates.length + 1
				});
			}

			editorSaved = true;
			showEmailEditor = false;
			await loadData();
		} catch (err: any) {
			error = err.message || 'Failed to save email';
		} finally {
			editorSaving = false;
		}
	}

	async function handleDeleteEmail(templateId: string) {
		deletingEmailId = templateId;
		try {
			await deleteTemplate({}, templateId);
			await loadData();
		} catch (err: any) {
			error = err.message || 'Failed to delete email';
		} finally {
			deletingEmailId = null;
		}
	}

	async function executeDeleteSequence() {
		deletingSequence = true;
		try {
			await deleteSequence({}, seqId);
			goto(`${basePath}/lists/${listId}/autoresponders`);
		} catch (err: any) {
			error = err.message || 'Failed to delete autoresponder';
		} finally {
			deletingSequence = false;
		}
	}

	function regeneratePlainText() {
		const div = document.createElement('div');
		div.innerHTML = editorHtmlBody;
		editorPlainText = div.textContent || div.innerText || '';
	}

	function formatDelay(hours: number): string {
		if (hours === 0) return 'Immediately';
		if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} after`;
		const days = Math.floor(hours / 24);
		const remainingHours = hours % 24;
		if (remainingHours === 0) return `${days} day${days === 1 ? '' : 's'} after`;
		return `${days}d ${remainingHours}h after`;
	}

	function getTriggerLabel(trigger: string): string {
		switch (trigger) {
			case 'signup': return 'On signup';
			case 'manual': return 'Manual enrollment';
			default: return trigger;
		}
	}
</script>

<svelte:head>
	<title>{sequence?.name || 'Autoresponder'} | Outlet</title>
</svelte:head>

<div class="p-6 max-w-5xl mx-auto">
	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if !sequence}
		<Alert type="error" title="Not found">
			<p>The requested autoresponder could not be found.</p>
			<Button type="primary" class="mt-3" onclick={() => goto(`${basePath}/lists/${listId}`)}>
				Back to List
			</Button>
		</Alert>
	{:else}
		<!-- Header -->
		<div class="flex items-center gap-4 mb-6">
			<a href="{basePath}/lists/{listId}/autoresponders" class="p-2 rounded-lg hover:bg-bg-secondary transition-colors text-text-muted hover:text-text">
				<ArrowLeft class="h-5 w-5" />
			</a>
			<div class="flex-1">
				<div class="flex items-center gap-2 text-sm text-text-muted mb-1">
					<span>{list?.name}</span>
					<span>/</span>
					<span>Autoresponder</span>
				</div>
				<div class="flex items-center gap-3">
					<h1 class="text-2xl font-semibold text-text">{sequence.name}</h1>
					{#if sequence.is_active}
						<Badge variant="success" size="sm">Active</Badge>
					{:else}
						<Badge variant="default" size="sm">Paused</Badge>
					{/if}
				</div>
				{#if sequence.on_completion_sequence_name}
					<div class="flex items-center gap-1 text-sm text-text-muted mt-1">
						<span>On completion:</span>
						<ArrowRight class="h-3 w-3" />
						<span class="text-primary">{sequence.on_completion_sequence_name}</span>
					</div>
				{/if}
			</div>
			<Button
				type={sequence.is_active ? 'secondary' : 'primary'}
				onclick={toggleActive}
			>
				{#if sequence.is_active}
					<Pause class="mr-2 h-4 w-4" />
					Pause
				{:else}
					<Play class="mr-2 h-4 w-4" />
					Activate
				{/if}
			</Button>
		</div>

		{#if error}
			<Alert type="error" title="Error" class="mb-4">
				<p>{error}</p>
			</Alert>
		{/if}

		<!-- Sequence Settings -->
		<Card class="mb-6">
			<h2 class="text-lg font-medium text-text mb-4">Settings</h2>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div>
					<label for="seq-name" class="form-label">Name</label>
					<Input id="seq-name" type="text" bind:value={editName} />
				</div>
				<div>
					<label for="seq-trigger" class="form-label">Trigger</label>
					<Select
						id="seq-trigger"
						bind:value={editTrigger}
						options={[
							{ value: 'signup', label: 'On signup' },
							{ value: 'manual', label: 'Manual enrollment' }
						]}
					/>
				</div>
			</div>
			<div class="mt-4">
				<label for="seq-list" class="form-label">List</label>
				<p class="text-sm text-text-muted mb-2">Move this autoresponder to a different list.</p>
				<Select
					id="seq-list"
					bind:value={editListId}
					options={allLists.map(l => ({ value: String(l.id), label: l.name }))}
				/>
			</div>
			<div class="mt-4">
				<label for="seq-on-completion" class="form-label">On Completion</label>
				<p class="text-sm text-text-muted mb-2">Automatically enroll contacts in another sequence when they complete this one.</p>
				<Select
					id="seq-on-completion"
					bind:value={editOnCompletionSequenceId}
					options={[
						{ value: '', label: 'None (do nothing)' },
						...allSequences.map(s => ({ value: s.id, label: s.name }))
					]}
				/>
			</div>
			<div class="mt-4 flex items-center gap-4">
				<Button
					type="primary"
					onclick={saveSequence}
					disabled={saving || (editName === sequence.name && editTrigger === sequence.trigger_event && editListId === (sequence.list_id || listId) && editOnCompletionSequenceId === (sequence.on_completion_sequence_id || ''))}
				>
					{#if saving}
						Saving...
					{:else if saved}
						Saved!
					{:else}
						Save Changes
					{/if}
				</Button>
				<Button type="danger" onclick={() => showDeleteSequence = true}>
					<Trash2 class="mr-2 h-4 w-4" />
					Delete
				</Button>
			</div>
		</Card>

		<!-- Emails Section -->
		<div class="flex items-center justify-between mb-4">
			<h2 class="text-lg font-medium text-text">Emails</h2>
			<Button type="primary" onclick={openCreateEmail}>
				<Plus class="mr-2 h-4 w-4" />
				Add Email
			</Button>
		</div>

		<!-- Email List -->
		{#if templates.length === 0}
			<EmptyState
				icon={Mail}
				title="No emails yet"
				description="Add your first email to this autoresponder sequence."
			>
				<Button type="primary" onclick={openCreateEmail}>
					<Plus class="mr-2 h-4 w-4" />
					Add Email
				</Button>
			</EmptyState>
		{:else}
			<div class="space-y-3">
				{#each templates as template, i}
					<Card class="flex items-center gap-4 {!template.is_active ? 'opacity-60' : ''}">
						<div class="text-text-muted">
							<GripVertical class="h-5 w-5" />
						</div>
						<div class="flex items-center justify-center h-8 w-8 rounded-full bg-primary text-white text-sm font-medium">
							{i + 1}
						</div>
						<div class="flex-1 min-w-0">
							<div class="font-medium text-text truncate">{template.subject}</div>
							<div class="text-sm text-text-muted flex items-center gap-2">
								<Clock class="h-3 w-3" />
								{formatDelay(template.delay_hours)}
							</div>
						</div>
						<div class="flex items-center gap-2">
							<Button type="ghost" size="sm" onclick={() => openEditEmail(template)}>
								<Edit class="h-4 w-4" />
							</Button>
							<Button
								type="ghost"
								size="sm"
								onclick={() => handleDeleteEmail(template.id)}
								disabled={deletingEmailId === template.id}
								class="text-error hover:bg-error/10"
							>
								<Trash2 class="h-4 w-4" />
							</Button>
						</div>
					</Card>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<!-- Full Screen Email Editor Overlay -->
{#if showEmailEditor}
	<div class="fixed inset-0 z-50 bg-base-200">
		<div class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-base-100 border-b border-base-300 px-6 py-4 flex items-center justify-between flex-shrink-0">
				<h3 class="text-lg font-semibold text-base-content">
					{editingEmail ? 'Edit Sequence Email' : 'Add Sequence Email'}
				</h3>
				<div class="flex items-center gap-3">
					<Button type="secondary" onclick={() => showEmailEditor = false}>
						<X class="mr-2 h-4 w-4" />
						Cancel
					</Button>
					<Button
						type="primary"
						onclick={saveEmailAndClose}
						disabled={editorSaving || !editorSubject.trim()}
					>
						<Save class="mr-2 h-4 w-4" />
						{editorSaving ? 'Saving...' : editorSaved ? 'Saved!' : 'Save & Close'}
					</Button>
				</div>
			</div>

			<!-- Content -->
			<div class="flex-1 overflow-hidden flex">
				<!-- Left Sidebar - Settings -->
				<div class="w-80 bg-base-100 border-r border-base-300 p-6 overflow-y-auto flex-shrink-0">
					<div class="space-y-6">
						<div>
							<label for="email-subject" class="form-label">Subject Line</label>
							<Input
								id="email-subject"
								type="text"
								bind:value={editorSubject}
								placeholder="Welcome to our newsletter!"
							/>
						</div>

						<div>
							<label for="email-delay" class="form-label">Send Timing</label>
							<div class="flex items-center gap-2">
								<Input
									id="email-delay"
									type="number"
									bind:value={editorDelay}
									min={0}
									class="w-24"
								/>
								<span class="text-sm text-base-content/70">hours after {templates.length === 0 && !editingEmail ? 'signup' : 'previous email'}</span>
							</div>
							<p class="mt-1 text-xs text-base-content/50 italic">
								{editorDelay === 0 ? 'Sends immediately' : `Waits ${editorDelay} hours`}
							</p>
						</div>

						<div class="bg-base-200 rounded-lg p-4">
							<h4 class="text-sm font-medium text-base-content mb-2">About This Email</h4>
							<p class="text-xs text-base-content/70">
								This email is part of your autoresponder sequence. It will be sent automatically based on the timing you set above.
							</p>
						</div>

						<PersonalizationTags
							variables={sequenceVariables}
							{insertVariable}
						/>
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col overflow-hidden p-6 gap-4">
					<div class="flex-1 min-h-0">
						<EmailEditor
							bind:value={editorHtmlBody}
							placeholder="Write your sequence email content..."
							showVariableInserts={false}
							onInsertVariable={(fn) => insertVariable = fn}
							class="h-full"
						/>
					</div>
					<div class="flex-shrink-0">
						<div class="flex items-center justify-between mb-1">
							<label for="email-plaintext" class="form-label mb-0">Plain Text Version</label>
							<button
								type="button"
								class="text-xs text-primary hover:text-primary/80"
								onclick={regeneratePlainText}
							>
								Regenerate from HTML
							</button>
						</div>
						<Textarea
							id="email-plaintext"
							bind:value={editorPlainText}
							placeholder="Plain text version of this email"
							rows={4}
						/>
						<p class="mt-1 text-xs text-base-content/50 italic">
							Auto-generated from HTML if left empty. Edit to customize.
						</p>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<AlertDialog
	bind:open={showDeleteSequence}
	title="Delete Autoresponder"
	description={`Are you sure you want to delete "${sequence?.name}"? This will delete all emails in this sequence. This action cannot be undone.`}
	actionLabel={deletingSequence ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDeleteSequence}
	onCancel={() => showDeleteSequence = false}
/>
