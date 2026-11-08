<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		listLists,
		listEmailDesigns,
		createCampaign,
		updateCampaign,
		type ListInfo,
		type EmailDesignInfo,
		type CampaignInfo,
		type CreateCampaignRequest
	} from '$lib/api';
	import { sendWithBrandKey } from '$lib/sendTest';
	import {
		Button,
		Input,
		Checkbox,
		Alert,
		LoadingSpinner,
		Textarea,
		Toggle,
		EmailEditor,
		PersonalizationTags
	} from '$lib/components/ui';
	import { ArrowLeft, Save, FileText, X, TestTube } from 'lucide-svelte';

	// With a campaign, edits it; without one, creates a draft
	let {
		campaign = undefined,
		onsaved = undefined,
		onclose = undefined
	}: {
		campaign?: CampaignInfo;
		onsaved?: (c: CampaignInfo) => void;
		onclose?: () => void;
	} = $props();

	// Form state, starting from the campaign being edited
	// svelte-ignore state_referenced_locally
	const initial = campaign;
	let name = $state(initial?.name ?? '');
	let subject = $state(initial?.subject ?? '');
	let previewText = $state(initial?.preview_text ?? '');
	let fromName = $state(initial?.from_name ?? '');
	let fromEmail = $state(initial?.from_email ?? '');
	let replyTo = $state(initial?.reply_to ?? '');
	let htmlBody = $state(initial?.html_body ?? '');
	let plainText = $state(initial?.plain_text ?? '');
	let trackOpens = $state(initial?.track_opens ?? true);
	let trackClicks = $state(initial?.track_clicks ?? true);
	let selectedListIds = $state<string[]>(initial?.list_ids ?? []);
	let excludeListIds = $state<string[]>(initial?.exclude_list_ids ?? []);
	let selectedDesignId = $state(initial?.design_id ?? '');

	// Data
	let lists = $state<ListInfo[]>([]);
	let designs = $state<EmailDesignInfo[]>([]);
	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');

	// Test email
	let testEmailAddress = $state('');
	let sendingTest = $state(false);
	let testSuccess = $state('');

	// Variable insertion
	let insertVariable = $state<((variable: string) => void) | null>(null);

	// Derived
	let basePath = $derived(`/${$page.params.brandSlug}`);
	let totalSubscribers = $derived(() => {
		return lists
			.filter((l) => selectedListIds.includes(l.id))
			.reduce((sum, l) => sum + l.subscriber_count, 0);
	});

	// Form validation
	let isValid = $derived(() => {
		return (
			name.trim() !== '' &&
			subject.trim() !== '' &&
			htmlBody.trim() !== '' &&
			selectedListIds.length > 0
		);
	});

	// Personalization variables for EmailEditor
	const campaignVariables = [
		{ name: 'email', label: 'Subscriber email' },
		{ name: 'name', label: 'Subscriber name' },
		{ name: 'name,fallback=Friend', label: 'Name with fallback' },
		{ name: 'unsubscribe_url', label: 'Unsubscribe link' },
		{ name: 'web_version_url', label: 'Web version link' }
	];

	$effect(() => {
		loadData();
	});

	async function loadData() {
		loading = true;
		error = '';

		try {
			const [listsRes, designsRes] = await Promise.all([
				listLists(),
				listEmailDesigns({})
			]);
			lists = listsRes.lists || [];
			designs = (designsRes.designs || []).filter((d) => d.is_active);
		} catch (err) {
			console.error('Failed to load data:', err);
			error = 'Failed to load lists and templates';
		} finally {
			loading = false;
		}
	}

	function handleDesignSelect(designId: string) {
		selectedDesignId = designId;
		if (designId) {
			const design = designs.find((d) => d.id === designId);
			if (design) {
				htmlBody = design.html_body;
				if (design.plain_text) {
					plainText = design.plain_text;
				}
			}
		}
	}

	function toggleList(listId: string) {
		if (selectedListIds.includes(listId)) {
			selectedListIds = selectedListIds.filter((id) => id !== listId);
		} else {
			selectedListIds = [...selectedListIds, listId];
		}
	}

	function toggleExcludeList(listId: string) {
		if (excludeListIds.includes(listId)) {
			excludeListIds = excludeListIds.filter((id) => id !== listId);
		} else {
			excludeListIds = [...excludeListIds, listId];
		}
	}

	async function handleSave() {
		if (!isValid()) {
			error = 'Please fill in all required fields';
			return;
		}

		saving = true;
		error = '';

		try {
			const request: CreateCampaignRequest = {
				name: name.trim(),
				subject: subject.trim(),
				preview_text: previewText.trim() || undefined,
				from_name: fromName.trim() || undefined,
				from_email: fromEmail.trim() || undefined,
				reply_to: replyTo.trim() || undefined,
				html_body: htmlBody,
				plain_text: plainText.trim() || undefined,
				list_ids: selectedListIds,
				exclude_list_ids: excludeListIds.length > 0 ? excludeListIds : undefined,
				track_opens: trackOpens,
				track_clicks: trackClicks,
				design_id: selectedDesignId || undefined
			};

			const saved = campaign
				? await updateCampaign({}, { ...request, preview_text: previewText.trim(), from_name: fromName.trim(), from_email: fromEmail.trim(), reply_to: replyTo.trim(), plain_text: plainText.trim(), exclude_list_ids: excludeListIds }, campaign.id)
				: await createCampaign(request);
			if (onsaved) {
				onsaved(saved);
			} else {
				goto(`${basePath}/campaigns/${saved.id}`);
			}
		} catch (err: any) {
			console.error('Failed to save campaign:', err);
			error = err.message || 'Failed to save campaign';
		} finally {
			saving = false;
		}
	}

	async function sendTestEmail() {
		if (!testEmailAddress.trim() || !subject.trim() || !htmlBody.trim()) {
			error = 'Please enter a test email address, subject, and content';
			return;
		}

		sendingTest = true;
		error = '';
		testSuccess = '';

		try {
			// Fill subscriber tags with sample values, as a subscriber would see them
			const sample = (text: string) =>
				text
					.replace(/\{\{\s*(name|first_name)\s*(,\s*fallback\s*=[^}]*)?\}\}/g, 'Test Subscriber')
					.replace(/\{\{\s*email\s*\}\}/g, testEmailAddress.trim())
					.replace(/\{\{\s*(unsubscribe_url|web_version_url)\s*\}\}/g, '#');
			await sendWithBrandKey($page.params.brandSlug as string, {
				to: testEmailAddress.trim(),
				subject: `[Test] ${sample(subject.trim())}`,
				body: sample(htmlBody),
				text_body: plainText.trim() ? sample(plainText.trim()) : undefined,
				from_name: fromName.trim() || undefined,
				from_email: fromEmail.trim() || undefined,
				reply_to: replyTo.trim() || undefined,
				tags: ['campaign-test']
			});
			testSuccess = `Test email sent to ${testEmailAddress}`;
			setTimeout(() => { testSuccess = ''; }, 5000);
		} catch (err: any) {
			console.error('Failed to send test email:', err);
			error = err.message || 'Failed to send test email';
		} finally {
			sendingTest = false;
		}
	}

	function regeneratePlainText() {
		const div = document.createElement('div');
		div.innerHTML = htmlBody;
		plainText = div.textContent || div.innerText || '';
	}

	function handleCancel() {
		if (onclose) {
			onclose();
		} else {
			goto(`${basePath}/campaigns`);
		}
	}
</script>

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="lg" />
	</div>
{:else}
	<!-- Full Screen Layout -->
	<div class="fixed inset-0 z-50 bg-base-200">
		<div class="h-full flex flex-col">
			<!-- Header -->
			<div class="bg-base-100 border-b border-base-300 px-4 sm:px-6 py-3 sm:py-4 flex flex-wrap items-center justify-between gap-3 flex-shrink-0">
				<div class="flex items-center gap-4">
					<button
						type="button"
						onclick={handleCancel}
						class="p-2 rounded-lg hover:bg-base-200 transition-colors text-base-content/60 hover:text-base-content"
					>
						<ArrowLeft class="h-5 w-5" />
					</button>
					<div>
						<input
							type="text"
							bind:value={name}
							placeholder="Campaign Name"
							class="text-lg font-semibold text-base-content bg-transparent border-none focus:outline-none focus:ring-0 placeholder:text-base-content/40 w-full sm:w-64"
						/>
						<p class="text-xs text-base-content/50">{campaign ? 'Editing draft' : 'New Campaign'}</p>
					</div>
				</div>
				<div class="flex flex-wrap items-center gap-2 sm:gap-3">
					<Button type="secondary" onclick={handleCancel}>
						<X class="mr-2 h-4 w-4" />
						Cancel
					</Button>
					<Button
						type="secondary"
						onclick={sendTestEmail}
						disabled={sendingTest || !subject.trim() || !htmlBody.trim() || !testEmailAddress.trim()}
					>
						<TestTube class="mr-2 h-4 w-4" />
						{sendingTest ? 'Sending...' : 'Send Test'}
					</Button>
					<Button
						type="primary"
						onclick={handleSave}
						disabled={saving || !isValid()}
					>
						<Save class="mr-2 h-4 w-4" />
						{saving ? 'Saving...' : campaign ? 'Save' : 'Save as Draft'}
					</Button>
				</div>
			</div>

			{#if error}
				<div class="px-4 sm:px-6 pt-4">
					<Alert type="error" title="Error" onclose={() => (error = '')}>
						<p>{error}</p>
					</Alert>
				</div>
			{/if}

			{#if testSuccess}
				<div class="px-4 sm:px-6 pt-4">
					<Alert type="success" title="Success" onclose={() => (testSuccess = '')}>
						<p>{testSuccess}</p>
					</Alert>
				</div>
			{/if}

			<!-- Content -->
			<div class="flex-1 overflow-y-auto lg:overflow-hidden flex flex-col lg:flex-row">
				<!-- Settings: a sidebar on wide screens, stacked above the editor on phones -->
				<div class="w-full lg:w-80 bg-base-100 border-b lg:border-b-0 lg:border-r border-base-300 p-4 sm:p-6 lg:overflow-y-auto flex-shrink-0">
					<div class="space-y-6">
						<!-- Subject Line -->
						<div>
							<label for="subject" class="form-label">Subject Line</label>
							<Input
								id="subject"
								type="text"
								bind:value={subject}
								placeholder="e.g., Check out what's new!"
							/>
						</div>

						<!-- Preview Text -->
						<div>
							<label for="preview-text" class="form-label">Preview Text</label>
							<Input
								id="preview-text"
								type="text"
								bind:value={previewText}
								placeholder="Text shown after subject..."
							/>
							<p class="mt-1 text-xs text-base-content/50 italic">
								Shown in inbox previews after the subject line.
							</p>
						</div>

						<div class="border-t border-base-300 pt-6">
							<h4 class="text-sm font-medium text-base-content mb-3">Recipients</h4>
							{#if lists.length === 0}
								<p class="text-xs text-base-content/60">No lists available.</p>
							{:else}
								<div class="space-y-2 max-h-48 overflow-y-auto">
									{#each lists as list}
										<label class="flex items-center gap-2 p-2 rounded-lg hover:bg-base-200 cursor-pointer transition-colors">
											<input
												type="checkbox"
												checked={selectedListIds.includes(list.id)}
												onchange={() => toggleList(list.id)}
												class="checkbox checkbox-primary checkbox-sm"
											/>
											<div class="flex-1 min-w-0 flex items-center justify-between">
												<span class="text-sm text-base-content truncate">{list.name}</span>
												<span class="text-xs text-base-content/50 ml-2">{list.subscriber_count.toLocaleString()}</span>
											</div>
										</label>
									{/each}
								</div>
								{#if selectedListIds.length > 0}
									<p class="text-xs text-base-content/60 mt-2">
										Total: <strong class="text-base-content">{totalSubscribers().toLocaleString()}</strong> recipients
									</p>
								{/if}
							{/if}
						</div>

						<!-- Exclude Lists -->
						{#if selectedListIds.length > 0 && lists.filter((l) => !selectedListIds.includes(l.id)).length > 0}
							<div class="border-t border-base-300 pt-6">
								<h4 class="text-sm font-medium text-base-content mb-3">Exclude Lists</h4>
								<div class="space-y-2 max-h-32 overflow-y-auto">
									{#each lists.filter((l) => !selectedListIds.includes(l.id)) as list}
										<label class="flex items-center gap-2 p-2 rounded-lg hover:bg-base-200 cursor-pointer transition-colors">
											<input
												type="checkbox"
												checked={excludeListIds.includes(list.id)}
												onchange={() => toggleExcludeList(list.id)}
												class="checkbox checkbox-warning checkbox-sm"
											/>
											<div class="flex-1 min-w-0">
												<span class="text-sm text-base-content block truncate">{list.name}</span>
											</div>
										</label>
									{/each}
								</div>
							</div>
						{/if}

						<!-- Sender (optional) -->
						<div class="border-t border-base-300 pt-6">
							<h4 class="text-sm font-medium text-base-content mb-3">Sender (optional)</h4>
							<div class="space-y-3">
								<Input
									placeholder="From Name"
									bind:value={fromName}
								/>
								<Input
									type="email"
									placeholder="From Email"
									bind:value={fromEmail}
								/>
								<Input
									type="email"
									placeholder="Reply-To Email"
									bind:value={replyTo}
								/>
							</div>
							<p class="mt-2 text-xs text-base-content/50 italic">
								Leave blank to use organization defaults.
							</p>
						</div>

						<!-- Template Selection -->
						{#if designs.length > 0}
							<div class="border-t border-base-300 pt-6">
								<h4 class="text-sm font-medium text-base-content mb-3">Template</h4>
								<div class="space-y-2">
									<button
										type="button"
										onclick={() => handleDesignSelect('')}
										class="w-full p-3 rounded-lg border text-left transition-colors flex items-center gap-3 {selectedDesignId === '' ? 'border-primary bg-primary/5' : 'border-base-300 hover:border-base-content/20'}"
									>
										<FileText class="h-5 w-5 text-base-content/50" />
										<div>
											<span class="text-sm font-medium text-base-content block">Blank</span>
											<span class="text-xs text-base-content/50">Start from scratch</span>
										</div>
									</button>
									{#each designs as design}
										<button
											type="button"
											onclick={() => handleDesignSelect(design.id)}
											class="w-full p-3 rounded-lg border text-left transition-colors flex items-center gap-3 {selectedDesignId === design.id ? 'border-primary bg-primary/5' : 'border-base-300 hover:border-base-content/20'}"
										>
											<FileText class="h-5 w-5 text-primary" />
											<div class="min-w-0">
												<span class="text-sm font-medium text-base-content block truncate">{design.name}</span>
												<span class="text-xs text-base-content/50">{design.category}</span>
											</div>
										</button>
									{/each}
								</div>
							</div>
						{/if}

						<!-- Tracking -->
						<div class="border-t border-base-300 pt-6">
							<h4 class="text-sm font-medium text-base-content mb-3">Tracking</h4>
							<div class="space-y-3">
								<label class="flex items-center gap-3 cursor-pointer">
									<input
										type="checkbox"
										bind:checked={trackOpens}
										class="checkbox checkbox-primary checkbox-sm"
									/>
									<span class="text-sm text-base-content">Track opens</span>
								</label>
								<label class="flex items-center gap-3 cursor-pointer">
									<input
										type="checkbox"
										bind:checked={trackClicks}
										class="checkbox checkbox-primary checkbox-sm"
									/>
									<span class="text-sm text-base-content">Track clicks</span>
								</label>
							</div>
						</div>

						<!-- Test Email -->
						<div class="border-t border-base-300 pt-6">
							<h4 class="text-sm font-medium text-base-content mb-3">Test Email</h4>
							<Input
								type="email"
								placeholder="your@email.com"
								bind:value={testEmailAddress}
							/>
							<p class="mt-1 text-xs text-base-content/50 italic">
								Enter an email address and click "Send Test" to preview.
							</p>
						</div>

						<!-- Personalization Tags -->
						<PersonalizationTags
							variables={campaignVariables}
							{insertVariable}
						/>
					</div>
				</div>

				<!-- Main Editor Area -->
				<div class="flex-1 flex flex-col lg:overflow-hidden p-4 sm:p-6 gap-4 min-h-[600px] lg:min-h-0">
					<div class="flex-1 min-h-[400px] lg:min-h-0">
						<EmailEditor
							bind:value={htmlBody}
							placeholder="Start writing your campaign email..."
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
							bind:value={plainText}
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
