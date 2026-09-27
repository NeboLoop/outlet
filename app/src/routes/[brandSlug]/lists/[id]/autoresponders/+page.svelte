<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		listSequences,
		createSequence,
		deleteSequence,
		updateSequence,
		type SequenceInfo
	} from '$lib/api';
	import {
		Button,
		Card,
		Input,
		Alert,
		LoadingSpinner,
		Badge,
		EmptyState,
		DropdownMenu,
		Select,
		Toggle,
		Modal,
		AlertDialog
	} from '$lib/components/ui';
	import {
		Workflow,
		Plus,
		MoreVertical,
		Trash2,
		Edit,
		Play,
		Pause
	} from '@lucide/svelte';
	import { getListContext } from '../listContext';

	const ctx = getListContext();
	let listId = $derived($page.params.id);
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// State
	let sequences = $state<SequenceInfo[]>([]);
	let sequencesLoading = $state(true);
	let showCreateSequence = $state(false);
	let creatingSequence = $state(false);
	let newSeqName = $state('');
	let newSeqSlug = $state('');
	let newSeqTrigger = $state('signup');
	let newSeqActive = $state(true);
	let error = $state('');

	// Sequences for this list only
	let sequencesForList = $derived(
		sequences.filter(s => s.list_id === listId)
	);

	$effect(() => {
		loadSequences();
	});

	async function loadSequences() {
		sequencesLoading = true;
		try {
			const response = await listSequences();
			sequences = response.sequences || [];
		} catch (err) {
			console.error('Failed to load sequences:', err);
		} finally {
			sequencesLoading = false;
		}
	}

	function generateSlug(name: string): string {
		return name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
	}

	function handleSeqNameInput() {
		newSeqSlug = generateSlug(newSeqName);
	}

	function openCreateSequence() {
		newSeqName = '';
		newSeqSlug = '';
		newSeqTrigger = 'signup';
		newSeqActive = true;
		showCreateSequence = true;
	}

	async function submitCreateSequence() {
		if (!newSeqName || !newSeqSlug) return;
		creatingSequence = true;
		error = '';
		try {
			await createSequence({
				name: newSeqName,
				slug: newSeqSlug,
				list_id: listId,
				trigger_event: newSeqTrigger,
				sequence_type: 'lifecycle'
			});
			showCreateSequence = false;
			await loadSequences();
		} catch (err: any) {
			error = err.message || 'Failed to create autoresponder';
		} finally {
			creatingSequence = false;
		}
	}

	let showDeleteConfirm = $state(false);
	let deleteSequenceItem = $state<SequenceInfo | null>(null);
	let deleting = $state(false);

	function confirmDeleteSequence(sequence: SequenceInfo) {
		deleteSequenceItem = sequence;
		showDeleteConfirm = true;
	}

	async function executeDeleteSequence() {
		if (!deleteSequenceItem) return;
		deleting = true;
		try {
			await deleteSequence({}, deleteSequenceItem.id);
			await loadSequences();
		} catch (err: any) {
			error = err.message || 'Failed to delete autoresponder';
		} finally {
			deleting = false;
		}
	}

	async function toggleSequenceActive(sequence: SequenceInfo) {
		try {
			await updateSequence({}, { is_active: !sequence.is_active }, sequence.id);
			await loadSequences();
		} catch (err: any) {
			error = err.message || 'Failed to update autoresponder';
		}
	}

	function getTriggerLabel(trigger: string): string {
		switch (trigger) {
			case 'signup': return 'On signup';
			case 'purchase': return 'On purchase';
			case 'tag_added': return 'Tag added';
			case 'manual': return 'Manual';
			default: return trigger;
		}
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

<div class="flex justify-between items-center mb-4">
	<p class="text-sm text-text-muted">
		Automated email sequences triggered when subscribers join this list.
	</p>
	<Button type="primary" onclick={openCreateSequence} disabled={showCreateSequence}>
		<Plus class="mr-2 h-4 w-4" />
		New Autoresponder
	</Button>
</div>

{#if sequencesLoading}
	<div class="flex justify-center py-12">
		<LoadingSpinner />
	</div>
{:else if sequencesForList.length === 0}
	<EmptyState
		icon={Workflow}
		title="No autoresponders yet"
		description="Create an autoresponder to automatically send emails when subscribers join this list."
	>
		<Button type="primary" onclick={openCreateSequence}>
			<Plus class="mr-2 h-4 w-4" />
			Create Autoresponder
		</Button>
	</EmptyState>
{:else}
	<div class="data-table">
		<table class="w-full">
			<thead>
				<tr>
					<th class="text-left">Autoresponder</th>
					<th class="text-left">Trigger</th>
					<th class="text-center">Status</th>
					<th class="text-right w-10"></th>
				</tr>
			</thead>
			<tbody>
				{#each sequencesForList as sequence}
					<tr>
						<td>
							<a href="{basePath}/lists/{listId}/autoresponders/{sequence.id}" class="block group">
								<div class="flex items-center gap-3">
									<div class="data-table-icon" class:active={sequence.is_active}>
										<Workflow class="h-4 w-4" />
									</div>
									<div>
										<span class="font-medium text-text group-hover:text-primary transition-colors">
											{sequence.name}
										</span>
									</div>
								</div>
							</a>
						</td>
						<td>
							<Badge variant="default" size="sm">
								{getTriggerLabel(sequence.trigger_event)}
							</Badge>
						</td>
						<td class="text-center">
							{#if sequence.is_active}
								<Badge variant="success" size="sm">Active</Badge>
							{:else}
								<Badge variant="default" size="sm">Paused</Badge>
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
										onclick: () => goto(`${basePath}/lists/${listId}/autoresponders/${sequence.id}`)
									},
									{
										label: sequence.is_active ? 'Pause' : 'Activate',
										icon: sequence.is_active ? Pause : Play,
										onclick: () => toggleSequenceActive(sequence)
									},
									{ divider: true },
									{
										label: 'Delete',
										icon: Trash2,
										variant: 'danger',
										onclick: () => confirmDeleteSequence(sequence)
									}
								]}
							/>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<!-- Create Autoresponder Modal -->
<Modal
	bind:show={showCreateSequence}
	title="Create Autoresponder"
	size="md"
>
	<div class="space-y-4">
		<div>
			<label for="seq-name" class="form-label">Name</label>
			<Input
				id="seq-name"
				type="text"
				bind:value={newSeqName}
				oninput={handleSeqNameInput}
				placeholder="Welcome Series"
			/>
		</div>
		<div>
			<label for="seq-slug" class="form-label">Slug</label>
			<Input
				id="seq-slug"
				type="text"
				bind:value={newSeqSlug}
				placeholder="welcome-series"
			/>
		</div>
		<div>
			<label for="seq-trigger" class="form-label">Trigger</label>
			<Select
				id="seq-trigger"
				bind:value={newSeqTrigger}
				options={[
					{ value: 'signup', label: 'On signup' },
					{ value: 'manual', label: 'Manual enrollment' }
				]}
			/>
		</div>
		<div class="flex items-center gap-3">
			<Toggle bind:checked={newSeqActive} />
			<span class="text-sm text-text">Active</span>
		</div>
	</div>

	{#snippet footer()}
		<Button type="secondary" onclick={() => showCreateSequence = false} disabled={creatingSequence}>
			Cancel
		</Button>
		<Button type="primary" onclick={submitCreateSequence} disabled={!newSeqName || !newSeqSlug || creatingSequence}>
			{creatingSequence ? 'Creating...' : 'Create'}
		</Button>
	{/snippet}
</Modal>

<AlertDialog
	bind:open={showDeleteConfirm}
	title="Delete Autoresponder"
	description={`Delete "${deleteSequenceItem?.name || ''}"? This cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDeleteSequence}
	onCancel={() => showDeleteConfirm = false}
/>
