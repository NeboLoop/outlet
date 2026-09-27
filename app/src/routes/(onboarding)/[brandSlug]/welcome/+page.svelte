<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import * as api from '$lib/api';
	import type { OrgInfo } from '$lib/api';
	import { Card, Input, Button, Alert, LoadingSpinner } from '$lib/components/ui';
	import { ChevronRight, Check, Loader2, HelpCircle, Lightbulb, Shield } from '@lucide/svelte';

	let loading = $state(false);
	let pageLoading = $state(true);
	let error = $state('');
	let success = $state(false);
	let org = $state<OrgInfo | null>(null);

	// Brand form fields
	let fromName = $state('');
	let fromEmail = $state('');
	let replyTo = $state('');

	// Get org slug from URL
	let brandSlug = $derived($page.params.brandSlug);

	$effect(() => {
		if (brandSlug) {
			loadOrg();
		}
	});

	async function loadOrg() {
		if (!brandSlug) return;
		pageLoading = true;
		try {
			org = await api.getOrganizationBySlug({}, brandSlug);
			// Pre-fill form with existing values
			if (org) {
				fromName = org.from_name || org.name;
				fromEmail = org.from_email || '';
				replyTo = org.reply_to || '';
			}
		} catch (err) {
			console.error('Failed to load organization:', err);
			error = 'Organization not found';
		} finally {
			pageLoading = false;
		}
	}

	async function saveAndComplete() {
		if (!org) return;

		loading = true;
		error = '';

		try {
			await api.updateOrgEmailSettings(
				{},
				{
					from_name: fromName,
					from_email: fromEmail,
					reply_to: replyTo || fromEmail
				},
				org.id
			);

			success = true;
		} catch (err: any) {
			console.error('Failed to save org email settings:', err);
			error = err.message || 'Failed to save email settings';
		} finally {
			loading = false;
		}
	}

	function goToDashboard() {
		goto(`/${brandSlug}`);
	}

	function skipForNow() {
		localStorage.setItem(`orgSetupSkipped_${brandSlug}`, 'true');
		goto(`/${brandSlug}`);
	}
</script>

<svelte:head>
	<title>Welcome - {org?.name || 'Setup'}</title>
</svelte:head>

{#if pageLoading}
	<Card>
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	</Card>
{:else if !org}
	<Card>
		<Alert type="error" title="Organization not found">
			<p>The organization you're looking for doesn't exist.</p>
		</Alert>
		<div class="mt-4">
			<Button type="secondary" onclick={() => goto('/')}>Go Back</Button>
		</div>
	</Card>
{:else if success}
	<!-- Success State -->
	<div class="grid grid-cols-1 lg:grid-cols-5 gap-8">
		<div class="lg:col-span-3">
			<Card>
				<div class="text-center">
					<div class="mx-auto w-16 h-16 rounded-full bg-green-500/10 flex items-center justify-center mb-6">
						<Check class="w-8 h-8 text-green-500" />
					</div>
					<h2 class="text-2xl font-bold text-text mb-2">You're all set!</h2>
					<p class="text-text-muted mb-8">
						Your sender settings have been saved. You're ready to start sending emails from {org.name}.
					</p>

					<Button type="primary" size="lg" onclick={goToDashboard} class="w-full justify-center">
						Go to Dashboard
						<ChevronRight class="w-4 h-4 ml-2" />
					</Button>
				</div>
			</Card>
		</div>

		<div class="lg:col-span-2">
			<div class="sticky top-8 space-y-6">
				<Card>
					<div class="flex items-center gap-2 mb-4">
						<Lightbulb class="w-5 h-5 text-amber-500" />
						<h3 class="font-semibold text-text">What's next?</h3>
					</div>
					<ul class="space-y-3 text-sm text-text-muted">
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5" />
							<span>Create your first email list</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5" />
							<span>Add subscribers manually or import a CSV</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5" />
							<span>Create your first campaign</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5" />
							<span>Set up welcome email automation</span>
						</li>
					</ul>
				</Card>
			</div>
		</div>
	</div>
{:else}
	<!-- Form State -->
	<div class="grid grid-cols-1 lg:grid-cols-5 gap-8">
		<div class="lg:col-span-3">
			<Card>
				<div class="mb-6">
					<h1 class="text-2xl font-bold text-text">Welcome to {org.name}</h1>
					<p class="text-text-muted mt-1">Set up your email sender identity so subscribers know who's emailing them.</p>
				</div>

				{#if error}
					<Alert type="error" title="Error" class="mb-6">
						<p>{error}</p>
					</Alert>
				{/if}

				<form class="space-y-5" onsubmit={(e) => { e.preventDefault(); saveAndComplete(); }}>
					<div>
						<label for="from-name" class="form-label">From Name</label>
						<Input
							id="from-name"
							type="text"
							placeholder="Your Company Name"
							bind:value={fromName}
							required
						/>
						<p class="mt-1 text-xs text-text-muted">The name that appears as the sender</p>
					</div>

					<div>
						<label for="from-email" class="form-label">From Email</label>
						<Input
							id="from-email"
							type="email"
							placeholder="newsletter@example.com"
							bind:value={fromEmail}
							required
						/>
						<p class="mt-1 text-xs text-text-muted">Must be a verified email address</p>
					</div>

					<div>
						<label for="reply-to" class="form-label">Reply-To Email (optional)</label>
						<Input
							id="reply-to"
							type="email"
							placeholder="support@example.com"
							bind:value={replyTo}
						/>
						<p class="mt-1 text-xs text-text-muted">Where replies will be sent</p>
					</div>

					<div class="flex gap-3 pt-4">
						<Button htmlType="submit" type="primary" disabled={loading || !fromName || !fromEmail} class="flex-1 justify-center">
							{#if loading}
								<Loader2 class="w-4 h-4 mr-2 animate-spin" />
								Saving...
							{:else}
								Save & Continue
								<ChevronRight class="w-4 h-4 ml-2" />
							{/if}
						</Button>
					</div>
				</form>

				<button
					type="button"
					onclick={skipForNow}
					class="mt-4 text-sm text-text-muted hover:text-text transition-colors w-full text-center"
				>
					Skip for now
				</button>
			</Card>
		</div>

		<div class="lg:col-span-2">
			<div class="sticky top-8 space-y-6">
				<Card>
					<div class="flex items-center gap-2 mb-4">
						<HelpCircle class="w-5 h-5 text-primary" />
						<h3 class="font-semibold text-text">Why set this up?</h3>
					</div>
					<ul class="space-y-3 text-sm text-text-muted">
						<li class="flex items-start gap-2">
							<span class="text-primary mt-0.5">1.</span>
							<span>Subscribers see a familiar name in their inbox</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary mt-0.5">2.</span>
							<span>Improves email deliverability and trust</span>
						</li>
						<li class="flex items-start gap-2">
							<span class="text-primary mt-0.5">3.</span>
							<span>Replies go to the right place</span>
						</li>
					</ul>
				</Card>

				<Card>
					<div class="flex items-center gap-2 mb-4">
						<Lightbulb class="w-5 h-5 text-amber-500" />
						<h3 class="font-semibold text-text">Best practices</h3>
					</div>
					<ul class="space-y-2 text-sm text-text-muted">
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
							<span>Use a recognizable sender name</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
							<span>Use a domain you own (not gmail.com)</span>
						</li>
						<li class="flex items-start gap-2">
							<Check class="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" />
							<span>Monitor your reply-to inbox regularly</span>
						</li>
					</ul>
				</Card>

				<Card>
					<div class="flex items-center gap-2 mb-4">
						<Shield class="w-5 h-5 text-green-500" />
						<h3 class="font-semibold text-text">Email verification</h3>
					</div>
					<p class="text-sm text-text-muted">
						Make sure your from email is verified with your email provider (like Amazon SES) before sending campaigns.
					</p>
				</Card>
			</div>
		</div>
	</div>
{/if}
