<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getList, updateList, deleteList, type ListInfo } from '$lib/api';
	import {
		Button,
		Card,
		Input,
		Alert,
		LoadingSpinner,
		Checkbox,
		Toggle,
		AlertDialog,
		EmailEditor,
		Textarea,
		PersonalizationTags
	} from '$lib/components/ui';
	import { Trash2, Pencil, X, Save } from '@lucide/svelte';
	import { getListContext } from '../listContext';

	const ctx = getListContext();
	let listId = $derived($page.params.id!);
	let basePath = $derived(`/${$page.params.brandSlug}`);

	// State
	let loading = $state(true);
	let list = $state<ListInfo | null>(null);
	let error = $state('');
	let saving = $state(false);
	let saved = $state(false);

	// Basic settings
	let editName = $state('');
	let editDescription = $state('');
	let editDoubleOptin = $state(true);

	// Confirmation email settings (for double opt-in)
	let editConfirmationSubject = $state('');
	let editConfirmationBody = $state('');
	let editConfirmationPlainText = $state('');

	// Subscribe redirect URLs
	let editThankYouUrl = $state('');
	let editConfirmRedirectUrl = $state('');
	let editAlreadySubscribedUrl = $state('');

	// Thank you email settings
	let editThankYouEmailEnabled = $state(false);
	let editThankYouEmailSubject = $state('');
	let editThankYouEmailBody = $state('');
	let editThankYouPlainText = $state('');

	// Unsubscribe settings
	let editUnsubscribeBehavior = $state('single');
	let editUnsubscribeScope = $state('list');
	let editUnsubscribeRedirectUrl = $state('');

	// Goodbye email settings
	let editGoodbyeEmailEnabled = $state(false);
	let editGoodbyeEmailSubject = $state('');
	let editGoodbyeEmailBody = $state('');
	let editGoodbyePlainText = $state('');

	// Delete confirmation
	let showDeleteConfirm = $state(false);
	let deleting = $state(false);

	// Email editor overlays
	let showConfirmationEditor = $state(false);
	let showThankYouEditor = $state(false);
	let showGoodbyeEditor = $state(false);
	let editorSaving = $state(false);
	let editorSaved = $state(false);

	// Variable insertion for each editor
	let insertConfirmationVar = $state<((variable: string) => void) | null>(null);
	let insertThankYouVar = $state<((variable: string) => void) | null>(null);
	let insertGoodbyeVar = $state<((variable: string) => void) | null>(null);

	// Variables for each email type
	const confirmationVariables = [
		{ name: 'confirm_url', label: 'Confirmation link', required: true },
		{ name: 'name', label: 'Subscriber name' },
		{ name: 'email', label: 'Subscriber email' },
		{ name: 'name,fallback=Friend', label: 'Name with fallback' }
	];

	const thankYouVariables = [
		{ name: 'name', label: 'Subscriber name' },
		{ name: 'email', label: 'Subscriber email' },
		{ name: 'name,fallback=Friend', label: 'Name with fallback' },
		{ name: 'currentday', label: 'Day name' },
		{ name: 'currentmonth', label: 'Month name' },
		{ name: 'currentyear', label: 'Year' },
		{ name: 'unsubscribe_url', label: 'Unsubscribe link' },
		{ name: 'webversion_url', label: 'Web version link' }
	];

	const goodbyeVariables = [
		{ name: 'name', label: 'Subscriber name' },
		{ name: 'email', label: 'Subscriber email' },
		{ name: 'name,fallback=Friend', label: 'Name with fallback' },
		{ name: 'resubscribe_url', label: 'Resubscribe link' }
	];

	// Strip HTML tags and convert to plain text
	function htmlToPlainText(html: string): string {
		if (!html) return '';
		return html
			// Remove style tags and their content
			.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
			// Remove script tags and their content
			.replace(/<script[^>]*>[\s\S]*?<\/script>/gi, '')
			// Convert <br> and <p> to newlines
			.replace(/<br\s*\/?>/gi, '\n')
			.replace(/<\/p>/gi, '\n\n')
			.replace(/<\/div>/gi, '\n')
			.replace(/<\/li>/gi, '\n')
			// Convert <a> tags to text with URL
			.replace(/<a[^>]*href=["']([^"']*)["'][^>]*>([^<]*)<\/a>/gi, '$2 ($1)')
			// Remove all remaining HTML tags
			.replace(/<[^>]+>/g, '')
			// Decode HTML entities
			.replace(/&nbsp;/g, ' ')
			.replace(/&amp;/g, '&')
			.replace(/&lt;/g, '<')
			.replace(/&gt;/g, '>')
			.replace(/&quot;/g, '"')
			.replace(/&#39;/g, "'")
			// Clean up extra whitespace
			.replace(/\n{3,}/g, '\n\n')
			.trim();
	}

	// Auto-generate plain text from HTML (only if plain text is empty)
	$effect(() => {
		if (editConfirmationBody && !editConfirmationPlainText.trim()) {
			editConfirmationPlainText = htmlToPlainText(editConfirmationBody);
		}
	});

	$effect(() => {
		if (editThankYouEmailBody && !editThankYouPlainText.trim()) {
			editThankYouPlainText = htmlToPlainText(editThankYouEmailBody);
		}
	});

	$effect(() => {
		if (editGoodbyeEmailBody && !editGoodbyePlainText.trim()) {
			editGoodbyePlainText = htmlToPlainText(editGoodbyeEmailBody);
		}
	});

	// Functions to regenerate plain text from HTML
	function regenerateConfirmationPlainText() {
		editConfirmationPlainText = htmlToPlainText(editConfirmationBody);
	}

	function regenerateThankYouPlainText() {
		editThankYouPlainText = htmlToPlainText(editThankYouEmailBody);
	}

	function regenerateGoodbyePlainText() {
		editGoodbyePlainText = htmlToPlainText(editGoodbyeEmailBody);
	}

	$effect(() => {
		loadList();
	});

	async function loadList() {
		loading = true;
		error = '';
		try {
			list = await getList({}, listId);
			editName = list.name;
			editDescription = list.description || '';
			editDoubleOptin = list.double_optin || false;
			// Confirmation email
			editConfirmationSubject = list.confirmation_subject || '';
			editConfirmationBody = list.confirmation_body || '';
			// Subscribe redirect URLs
			editThankYouUrl = list.thank_you_url || '';
			editConfirmRedirectUrl = list.confirm_redirect_url || '';
			editAlreadySubscribedUrl = list.already_subscribed_url || '';
			// Thank you email
			editThankYouEmailEnabled = list.thank_you_email_enabled || false;
			editThankYouEmailSubject = list.thank_you_email_subject || '';
			editThankYouEmailBody = list.thank_you_email_body || '';
			// Unsubscribe settings
			editUnsubscribeBehavior = list.unsubscribe_behavior || 'single';
			editUnsubscribeScope = list.unsubscribe_scope || 'list';
			editUnsubscribeRedirectUrl = list.unsubscribe_redirect_url || '';
			// Goodbye email
			editGoodbyeEmailEnabled = list.goodbye_email_enabled || false;
			editGoodbyeEmailSubject = list.goodbye_email_subject || '';
			editGoodbyeEmailBody = list.goodbye_email_body || '';
		} catch (err) {
			console.error('Failed to load list:', err);
			error = 'Failed to load list settings';
		} finally {
			loading = false;
		}
	}

	async function saveSettings() {
		if (!list || !editName.trim()) return;
		saving = true;
		saved = false;
		error = '';
		try {
			await updateList({}, {
				name: editName.trim(),
				description: editDescription.trim(),
				double_optin: editDoubleOptin,
				confirmation_subject: editConfirmationSubject.trim() || undefined,
				confirmation_body: editConfirmationBody.trim() || undefined,
				thank_you_url: editThankYouUrl.trim() || undefined,
				confirm_redirect_url: editConfirmRedirectUrl.trim() || undefined,
				already_subscribed_url: editAlreadySubscribedUrl.trim() || undefined,
				thank_you_email_enabled: editThankYouEmailEnabled,
				thank_you_email_subject: editThankYouEmailSubject.trim() || undefined,
				thank_you_email_body: editThankYouEmailBody.trim() || undefined,
				unsubscribe_behavior: editUnsubscribeBehavior,
				unsubscribe_scope: editUnsubscribeScope,
				unsubscribe_redirect_url: editUnsubscribeRedirectUrl.trim() || undefined,
				goodbye_email_enabled: editGoodbyeEmailEnabled,
				goodbye_email_subject: editGoodbyeEmailSubject.trim() || undefined,
				goodbye_email_body: editGoodbyeEmailBody.trim() || undefined
			}, listId);
			// Reload to get updated values and trigger layout refresh
			await loadList();
			await ctx.reload?.();
			saved = true;
			setTimeout(() => { saved = false; }, 2000);
		} catch (err: any) {
			error = err.message || 'Failed to save settings';
		} finally {
			saving = false;
		}
	}

	async function saveEmailAndClose(closeEditor: () => void) {
		editorSaving = true;
		editorSaved = false;
		try {
			await saveSettings();
			editorSaved = true;
			setTimeout(() => {
				closeEditor();
				editorSaved = false;
			}, 500);
		} catch (err) {
			// Error already handled in saveSettings
		} finally {
			editorSaving = false;
		}
	}

	async function executeDelete() {
		deleting = true;
		try {
			await deleteList({}, listId);
			goto(`${basePath}/lists`);
		} catch (err: any) {
			error = err.message || 'Failed to delete list';
		} finally {
			deleting = false;
		}
	}

	function getEmailPreview(body: string): string {
		if (!body) return 'No content yet';
		const text = body
			.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '')
			.replace(/<[^>]*>/g, ' ')
			.replace(/\s+/g, ' ')
			.trim();
		return text.slice(0, 100) + (text.length > 100 ? '...' : '');
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner />
	</div>
{:else if list}
	<div class="space-y-6">
		<!-- General Settings -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-4">General</h2>
			<div class="space-y-4">
				<div>
					<label for="list-name" class="form-label">List Name</label>
					<Input id="list-name" type="text" bind:value={editName} />
				</div>
				<div>
					<label for="list-desc" class="form-label">Description</label>
					<Input id="list-desc" type="text" bind:value={editDescription} placeholder="Optional description" />
				</div>
				<div>
					<label class="form-label">Slug</label>
					<p class="text-sm text-text-muted bg-bg-secondary px-3 py-2 rounded-md">{list.slug}</p>
					<p class="mt-1 text-xs text-text-muted">The slug cannot be changed after creation</p>
				</div>
				<div>
					<Checkbox bind:checked={editDoubleOptin} label="Require double opt-in" />
					<p class="text-xs text-text-muted mt-1 ml-6">
						Subscribers must confirm their email before being added
					</p>
				</div>
			</div>
		</Card>

		<!-- Confirmation Email (Double Opt-in) -->
		{#if editDoubleOptin}
			<Card>
				<div class="flex items-center justify-between mb-4">
					<div>
						<h2 class="text-lg font-medium text-text">Confirmation Email</h2>
						<p class="text-sm text-text-muted">Email sent to confirm subscription (double opt-in)</p>
					</div>
					<Button type="secondary" onclick={() => showConfirmationEditor = true}>
						<Pencil class="mr-2 h-4 w-4" />
						Edit Email
					</Button>
				</div>
				<div class="bg-bg-secondary rounded-lg p-4">
					<div class="text-sm">
						<span class="font-medium text-text">Subject:</span>
						{#if editConfirmationSubject}
							<span class="text-text-muted ml-2">{editConfirmationSubject}</span>
						{:else}
							<span class="text-text-muted/50 ml-2 italic">Using default: "Please confirm your subscription"</span>
						{/if}
					</div>
					<div class="text-sm mt-2">
						<span class="font-medium text-text">Preview:</span>
						{#if editConfirmationBody}
							<span class="text-text-muted ml-2">{getEmailPreview(editConfirmationBody)}</span>
						{:else}
							<span class="text-text-muted/50 ml-2 italic">Using default confirmation email template</span>
						{/if}
					</div>
				</div>
			</Card>
		{/if}

		<!-- Subscribe Settings -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-2">Subscribe Settings</h2>
			<p class="text-sm text-text-muted mb-4">
				Configure redirect URLs for new subscribers.
			</p>
			<div class="space-y-4">
				<div>
					<label for="thank-you-url" class="form-label">Subscribe Success Page URL</label>
					<Input
						id="thank-you-url"
						type="url"
						bind:value={editThankYouUrl}
						placeholder="https://yoursite.com/thank-you"
					/>
					<p class="mt-1 text-xs text-text-muted">
						Where to redirect after someone submits the subscribe form (before confirmation if double opt-in).
					</p>
					<p class="mt-0.5 text-xs text-text-muted/60 italic">
						A generic success page will be shown if left empty.
					</p>
				</div>

				{#if editDoubleOptin}
					<div>
						<label for="confirm-redirect-url" class="form-label">Subscription Confirmed Page URL</label>
						<Input
							id="confirm-redirect-url"
							type="url"
							bind:value={editConfirmRedirectUrl}
							placeholder="https://yoursite.com/subscription-confirmed"
						/>
						<p class="mt-1 text-xs text-text-muted">
							Where to redirect after someone confirms their subscription (clicks the confirmation link).
						</p>
						<p class="mt-0.5 text-xs text-text-muted/60 italic">
							A generic confirmation page will be shown if left empty.
						</p>
					</div>
				{/if}

				<div>
					<label for="already-subscribed-url" class="form-label">Already Subscribed Redirect URL</label>
					<Input
						id="already-subscribed-url"
						type="url"
						bind:value={editAlreadySubscribedUrl}
						placeholder="https://yoursite.com/already-subscribed"
					/>
					<p class="mt-1 text-xs text-text-muted">
						Where to redirect if someone tries to subscribe but is already on the list.
					</p>
					<p class="mt-0.5 text-xs text-text-muted/60 italic">
						A generic "already subscribed" page will be shown if left empty.
					</p>
				</div>
			</div>
		</Card>

		<!-- Thank You Email -->
		<Card>
			<div class="flex items-center justify-between mb-4">
				<div>
					<h2 class="text-lg font-medium text-text">Thank You Email</h2>
					<p class="text-sm text-text-muted">Send a welcome email after successful subscription</p>
				</div>
				<Toggle bind:checked={editThankYouEmailEnabled} />
			</div>

			{#if editThankYouEmailEnabled}
				<div class="border-t border-border pt-4">
					<div class="flex items-center justify-between mb-3">
						<div class="bg-bg-secondary rounded-lg p-4 flex-1 mr-4">
							<div class="text-sm">
								<span class="font-medium text-text">Subject:</span>
								{#if editThankYouEmailSubject}
									<span class="text-text-muted ml-2">{editThankYouEmailSubject}</span>
								{:else}
									<span class="text-text-muted/50 ml-2 italic">Using default: "Welcome!"</span>
								{/if}
							</div>
							<div class="text-sm mt-2">
								<span class="font-medium text-text">Preview:</span>
								{#if editThankYouEmailBody}
									<span class="text-text-muted ml-2">{getEmailPreview(editThankYouEmailBody)}</span>
								{:else}
									<span class="text-text-muted/50 ml-2 italic">Using default welcome email template</span>
								{/if}
							</div>
						</div>
						<Button type="secondary" onclick={() => showThankYouEditor = true}>
							<Pencil class="mr-2 h-4 w-4" />
							Edit Email
						</Button>
					</div>
				</div>
			{/if}
		</Card>

		<!-- Unsubscribe Settings -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-4">Unsubscribe Settings</h2>
			<div class="space-y-4">
				<div>
					<label class="form-label">Unsubscribe Behavior</label>
					<div class="space-y-2">
						<label class="flex items-center gap-3 p-3 rounded-lg border border-border hover:border-primary/50 cursor-pointer transition-colors">
							<input
								type="radio"
								name="unsubscribe-behavior"
								value="single"
								bind:group={editUnsubscribeBehavior}
								class="radio radio-primary"
							/>
							<div>
								<span class="font-medium text-text">Single opt-out</span>
								<p class="text-sm text-text-muted">Immediately unsubscribe when they click the link</p>
							</div>
						</label>
						<label class="flex items-center gap-3 p-3 rounded-lg border border-border hover:border-primary/50 cursor-pointer transition-colors">
							<input
								type="radio"
								name="unsubscribe-behavior"
								value="double"
								bind:group={editUnsubscribeBehavior}
								class="radio radio-primary"
							/>
							<div>
								<span class="font-medium text-text">Double opt-out</span>
								<p class="text-sm text-text-muted">Require confirmation before unsubscribing</p>
							</div>
						</label>
					</div>
				</div>

				<div>
					<label class="form-label">Unsubscribe Scope</label>
					<div class="space-y-2">
						<label class="flex items-center gap-3 p-3 rounded-lg border border-border hover:border-primary/50 cursor-pointer transition-colors">
							<input
								type="radio"
								name="unsubscribe-scope"
								value="list"
								bind:group={editUnsubscribeScope}
								class="radio radio-primary"
							/>
							<div>
								<span class="font-medium text-text">This list only</span>
								<p class="text-sm text-text-muted">Unsubscribe from this list, but stay on other lists</p>
							</div>
						</label>
						<label class="flex items-center gap-3 p-3 rounded-lg border border-border hover:border-primary/50 cursor-pointer transition-colors">
							<input
								type="radio"
								name="unsubscribe-scope"
								value="all"
								bind:group={editUnsubscribeScope}
								class="radio radio-primary"
							/>
							<div>
								<span class="font-medium text-text">All lists</span>
								<p class="text-sm text-text-muted">Unsubscribe from all lists in the organization</p>
							</div>
						</label>
					</div>
				</div>

				<div>
					<label for="unsubscribe-redirect-url" class="form-label">Unsubscribe Confirmation Page URL</label>
					<Input
						id="unsubscribe-redirect-url"
						type="url"
						bind:value={editUnsubscribeRedirectUrl}
						placeholder="https://yoursite.com/unsubscribed"
					/>
					<p class="mt-1 text-xs text-text-muted">
						Where to redirect after someone unsubscribes.
					</p>
					<p class="mt-0.5 text-xs text-text-muted/60 italic">
						A generic unsubscribe confirmation page will be shown if left empty.
					</p>
				</div>
			</div>
		</Card>

		<!-- Goodbye Email -->
		<Card>
			<div class="flex items-center justify-between mb-4">
				<div>
					<h2 class="text-lg font-medium text-text">Goodbye Email</h2>
					<p class="text-sm text-text-muted">Send a farewell email after unsubscribing</p>
				</div>
				<Toggle bind:checked={editGoodbyeEmailEnabled} />
			</div>

			{#if editGoodbyeEmailEnabled}
				<div class="border-t border-border pt-4">
					<div class="flex items-center justify-between mb-3">
						<div class="bg-bg-secondary rounded-lg p-4 flex-1 mr-4">
							<div class="text-sm">
								<span class="font-medium text-text">Subject:</span>
								{#if editGoodbyeEmailSubject}
									<span class="text-text-muted ml-2">{editGoodbyeEmailSubject}</span>
								{:else}
									<span class="text-text-muted/50 ml-2 italic">Using default: "Sorry to see you go"</span>
								{/if}
							</div>
							<div class="text-sm mt-2">
								<span class="font-medium text-text">Preview:</span>
								{#if editGoodbyeEmailBody}
									<span class="text-text-muted ml-2">{getEmailPreview(editGoodbyeEmailBody)}</span>
								{:else}
									<span class="text-text-muted/50 ml-2 italic">Using default goodbye email template</span>
								{/if}
							</div>
						</div>
						<Button type="secondary" onclick={() => showGoodbyeEditor = true}>
							<Pencil class="mr-2 h-4 w-4" />
							Edit Email
						</Button>
					</div>
				</div>
			{/if}
		</Card>

		<!-- Save Button -->
		<Card>
			<div class="flex items-center justify-between">
				<p class="text-sm text-text-muted">Save all changes to this list's settings</p>
				<Button
					type="primary"
					onclick={saveSettings}
					disabled={saving}
				>
					{#if saving}
						Saving...
					{:else if saved}
						Saved!
					{:else}
						Save Changes
					{/if}
				</Button>
			</div>
		</Card>

		<Card>
			<h2 class="text-lg font-medium text-red-600 mb-4">Danger Zone</h2>
			<p class="text-sm text-text-muted mb-4">
				Permanently delete this list and all its subscribers. This action cannot be undone.
			</p>
			<Button type="danger" onclick={() => showDeleteConfirm = true}>
				<Trash2 class="mr-2 h-4 w-4" />
				Delete List
			</Button>
		</Card>
	</div>
{/if}

<!-- Confirmation Email Editor (Full Screen Overlay) -->
{#if showConfirmationEditor}
	<div class="fixed inset-0 z-50 bg-base-200">
		<div class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-base-100 border-b border-base-300 px-6 py-4 flex items-center justify-between flex-shrink-0">
				<h3 class="text-lg font-semibold text-base-content">Edit Confirmation Email</h3>
				<div class="flex items-center gap-3">
					<Button type="secondary" onclick={() => showConfirmationEditor = false}>
						<X class="mr-2 h-4 w-4" />
						Cancel
					</Button>
					<Button
						type="primary"
						onclick={() => saveEmailAndClose(() => showConfirmationEditor = false)}
						disabled={editorSaving}
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
							<label for="conf-subject" class="form-label">Subject Line</label>
							<Input
								id="conf-subject"
								type="text"
								bind:value={editConfirmationSubject}
								placeholder="Please confirm your subscription"
							/>
							<p class="mt-1 text-xs text-base-content/50 italic">
								A generic subject line will be used if you leave this field empty.
							</p>
						</div>

						<div class="bg-base-200 rounded-lg p-4">
							<h4 class="text-sm font-medium text-base-content mb-2">About This Email</h4>
							<p class="text-xs text-base-content/70">
								This email is sent when someone subscribes to your list with double opt-in enabled.
								It must include a confirmation link for the subscriber to click.
							</p>
							<p class="text-xs text-base-content/50 italic mt-2">
								A generic email message will be used if you leave the body empty.
							</p>
						</div>

						<PersonalizationTags
							variables={confirmationVariables}
							insertVariable={insertConfirmationVar}
						/>
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col overflow-hidden p-6 gap-4">
					<div class="flex-1 min-h-0">
						<EmailEditor
							bind:value={editConfirmationBody}
							placeholder="Click the link below to confirm your subscription..."
							showVariableInserts={false}
							onInsertVariable={(fn) => insertConfirmationVar = fn}
							class="h-full"
						/>
					</div>
					<div class="flex-shrink-0">
						<div class="flex items-center justify-between mb-1">
							<label for="conf-plaintext" class="form-label mb-0">Plain Text Version</label>
							<button
								type="button"
								class="text-xs text-primary hover:text-primary/80"
								onclick={regenerateConfirmationPlainText}
							>
								Regenerate from HTML
							</button>
						</div>
						<Textarea
							id="conf-plaintext"
							bind:value={editConfirmationPlainText}
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

<!-- Thank You Email Editor (Full Screen Overlay) -->
{#if showThankYouEditor}
	<div class="fixed inset-0 z-50 bg-base-200">
		<div class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-base-100 border-b border-base-300 px-6 py-4 flex items-center justify-between flex-shrink-0">
				<h3 class="text-lg font-semibold text-base-content">Edit Thank You Email</h3>
				<div class="flex items-center gap-3">
					<Button type="secondary" onclick={() => showThankYouEditor = false}>
						<X class="mr-2 h-4 w-4" />
						Cancel
					</Button>
					<Button
						type="primary"
						onclick={() => saveEmailAndClose(() => showThankYouEditor = false)}
						disabled={editorSaving}
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
							<label for="ty-subject" class="form-label">Subject Line</label>
							<Input
								id="ty-subject"
								type="text"
								bind:value={editThankYouEmailSubject}
								placeholder="Welcome to our list!"
							/>
							<p class="mt-1 text-xs text-base-content/50 italic">
								A generic subject line will be used if you leave this field empty.
							</p>
						</div>

						<div class="bg-base-200 rounded-lg p-4">
							<h4 class="text-sm font-medium text-base-content mb-2">About This Email</h4>
							<p class="text-xs text-base-content/70">
								This email is sent after someone successfully subscribes to your list.
								Use it to welcome new subscribers and set expectations.
							</p>
							<p class="text-xs text-base-content/50 italic mt-2">
								A generic email message will be used if you leave the body empty.
							</p>
						</div>

						<PersonalizationTags
							variables={thankYouVariables}
							insertVariable={insertThankYouVar}
						/>
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col overflow-hidden p-6 gap-4">
					<div class="flex-1 min-h-0">
						<EmailEditor
							bind:value={editThankYouEmailBody}
							placeholder="Thank you for subscribing!"
							showVariableInserts={false}
							onInsertVariable={(fn) => insertThankYouVar = fn}
							class="h-full"
						/>
					</div>
					<div class="flex-shrink-0">
						<div class="flex items-center justify-between mb-1">
							<label for="ty-plaintext" class="form-label mb-0">Plain Text Version</label>
							<button
								type="button"
								class="text-xs text-primary hover:text-primary/80"
								onclick={regenerateThankYouPlainText}
							>
								Regenerate from HTML
							</button>
						</div>
						<Textarea
							id="ty-plaintext"
							bind:value={editThankYouPlainText}
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

<!-- Goodbye Email Editor (Full Screen Overlay) -->
{#if showGoodbyeEditor}
	<div class="fixed inset-0 z-50 bg-base-200">
		<div class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-base-100 border-b border-base-300 px-6 py-4 flex items-center justify-between flex-shrink-0">
				<h3 class="text-lg font-semibold text-base-content">Edit Goodbye Email</h3>
				<div class="flex items-center gap-3">
					<Button type="secondary" onclick={() => showGoodbyeEditor = false}>
						<X class="mr-2 h-4 w-4" />
						Cancel
					</Button>
					<Button
						type="primary"
						onclick={() => saveEmailAndClose(() => showGoodbyeEditor = false)}
						disabled={editorSaving}
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
							<label for="bye-subject" class="form-label">Subject Line</label>
							<Input
								id="bye-subject"
								type="text"
								bind:value={editGoodbyeEmailSubject}
								placeholder="Sorry to see you go"
							/>
							<p class="mt-1 text-xs text-base-content/50 italic">
								A generic subject line will be used if you leave this field empty.
							</p>
						</div>

						<div class="bg-base-200 rounded-lg p-4">
							<h4 class="text-sm font-medium text-base-content mb-2">About This Email</h4>
							<p class="text-xs text-base-content/70">
								This email is sent after someone unsubscribes from your list.
								Consider including a way for them to resubscribe if they change their mind.
							</p>
							<p class="text-xs text-base-content/50 italic mt-2">
								A generic email message will be used if you leave the body empty.
							</p>
						</div>

						<PersonalizationTags
							variables={goodbyeVariables}
							insertVariable={insertGoodbyeVar}
						/>
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col overflow-hidden p-6 gap-4">
					<div class="flex-1 min-h-0">
						<EmailEditor
							bind:value={editGoodbyeEmailBody}
							placeholder="You have been unsubscribed."
							showVariableInserts={false}
							onInsertVariable={(fn) => insertGoodbyeVar = fn}
							class="h-full"
						/>
					</div>
					<div class="flex-shrink-0">
						<div class="flex items-center justify-between mb-1">
							<label for="bye-plaintext" class="form-label mb-0">Plain Text Version</label>
							<button
								type="button"
								class="text-xs text-primary hover:text-primary/80"
								onclick={regenerateGoodbyePlainText}
							>
								Regenerate from HTML
							</button>
						</div>
						<Textarea
							id="bye-plaintext"
							bind:value={editGoodbyePlainText}
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
	bind:open={showDeleteConfirm}
	title="Delete List"
	description={`Are you sure you want to delete "${list?.name}"? This will permanently delete all subscribers and autoresponders. This action cannot be undone.`}
	actionLabel={deleting ? 'Deleting...' : 'Delete'}
	actionType="danger"
	onAction={executeDelete}
	onCancel={() => showDeleteConfirm = false}
/>
