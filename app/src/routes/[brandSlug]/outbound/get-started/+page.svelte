<script lang="ts">
	import { tick } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import {
		getOutboundOverview,
		listOutboundSegments,
		type OutboundProfile,
		type OutboundSegment,
		type OutboundSetupStep
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Badge, Select } from '$lib/components/ui';
	import { Check, ArrowLeft, ArrowRight } from '@lucide/svelte';
	import {
		APPROVAL_MODES,
		SETUP_STEPS,
		askAIPrompt,
		errorMessage,
		openingStep,
		setupDone,
		skipSetup
	} from '../shared';
	import AskAI from '../AskAI.svelte';
	import ProfileForm from '../ProfileForm.svelte';
	import SegmentsEditor from '../SegmentsEditor.svelte';
	import SendingSetup from '../SendingSetup.svelte';
	import DncLists from '../DncLists.svelte';
	import CampaignForm from '../CampaignForm.svelte';

	// The guided setup: five steps from nothing to a first campaign, each
	// built from the same forms as the Outbound tabs. Every change saves as it
	// is made, so leaving and coming back picks up where things stand; which
	// steps are done comes from the server, read from the brand's data.

	const INTROS: Record<string, string> = {
		business:
			"Outreach is written from what Outlet knows about the business. Enter the website and let your AI fill in the rest, or type it in yourself. Read it over and fix anything that's off.",
		targeting:
			'Segments are who a campaign goes to, such as "Dental clinics in Utah". Let your AI propose them from the profile, or make your own. Keep the ones that fit, edit them, and remove the rest.',
		sending:
			'Outreach goes from mailboxes on a domain of its own, never your main one. Add the domain, put its records in your DNS and check them, add a mailbox, and set how many new people get a first email each day.',
		do_not_contact:
			"People Outlet already knows, on your lists or sent a transactional email, are skipped by themselves. Import your other customers and anyone else who should never get outreach, typed or from a CSV.",
		campaign:
			"Pick a segment, have your AI draft the emails from your materials or write them yourself, and choose how they're approved. Nothing sends until it's approved, unless you choose to send without approval."
	};

	let brandSlug = $derived($page.params.brandSlug || '');
	let base = $derived(`/${brandSlug}/outbound`);

	let loading = $state(true);
	let error = $state('');
	let setup = $state<OutboundSetupStep[]>([]);

	let current = $derived(openingStep(setup, $page.url.searchParams.get('step')));
	let index = $derived(SETUP_STEPS.findIndex((s) => s.step === current));
	let info = $derived(SETUP_STEPS[index]);
	let status = $derived(setup.find((s) => s.step === current));
	let doneCount = $derived(setup.filter((s) => s.done).length);
	let allDone = $derived(setupDone(setup));

	let heading = $state<HTMLHeadingElement | null>(null);
	// Bumped to read a step's forms again, after an AI saved to them.
	let reloads = $state(0);

	let profile = $state<OutboundProfile>({});
	let segments = $state<OutboundSegment[]>([]);
	let aiSegment = $state('');
	let aiApproval = $state('every');
	let writeOwn = $state(false);
	let createdId = $state('');

	$effect(() => {
		load();
	});

	// The first step shown, and each one moved to, takes the focus at its
	// heading so a keyboard or screen reader lands on it.
	let shown = '';
	$effect(() => {
		const step = current;
		if (loading || step === shown) return;
		const first = shown === '';
		shown = step;
		if (step === 'campaign') loadSegments();
		if (!first) tick().then(() => heading?.focus());
	});

	async function load() {
		loading = true;
		error = '';
		try {
			await refreshSetup();
			// Pin the step it opens on, so finishing it doesn't move the page
			// on before the person has read what was saved.
			if (!$page.url.searchParams.get('step')) {
				await goto(stepHref(openingStep(setup, null)), { replaceState: true, noScroll: true, keepFocus: true });
			}
		} catch (err) {
			error = errorMessage(err, 'Could not load the setup');
		} finally {
			loading = false;
		}
	}

	async function refreshSetup() {
		setup = (await getOutboundOverview()).setup || [];
	}

	// A change in a step: read which steps are done again.
	async function changed() {
		try {
			await refreshSetup();
		} catch (err) {
			error = errorMessage(err, 'Could not read the setup');
		}
	}

	async function loadSegments() {
		try {
			segments = (await listOutboundSegments()).segments || [];
			if (!segments.some((s) => s.name === aiSegment)) {
				aiSegment = (segments.find((s) => s.description) || segments[0])?.name || '';
			}
		} catch (err) {
			error = errorMessage(err, 'Could not load the segments');
		}
	}

	// After asking an AI: show what it saved.
	async function reread() {
		reloads++;
		if (current === 'campaign') await loadSegments();
		await changed();
	}

	function stepHref(step: string): string {
		return `${base}/get-started?step=${step}`;
	}

	function skip() {
		skipSetup(brandSlug, true);
		goto(base);
	}

	function created(id: string) {
		createdId = id;
		writeOwn = false;
		changed();
	}
</script>

<svelte:head>
	<title>Set up outbound | Outlet</title>
</svelte:head>

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else}
	<div class="flex flex-wrap items-center justify-between gap-2 mb-4">
		<p class="text-sm text-text-muted">
			Set up outbound: {doneCount} of {SETUP_STEPS.length} done
		</p>
		<Button type="ghost" size="sm" onclick={skip}>Skip setup for now</Button>
	</div>

	<nav aria-label="Setup steps" class="mb-6">
		<ol class="setup-progress">
			{#each SETUP_STEPS as s, i (s.step)}
				{@const done = setup.find((x) => x.step === s.step)?.done}
				<li>
					<a
						href={stepHref(s.step)}
						class="setup-progress-item"
						class:current={s.step === current}
						aria-current={s.step === current ? 'step' : undefined}
						data-sveltekit-noscroll
					>
						<span class="setup-progress-dot" class:done>
							{#if done}
								<Check class="h-4 w-4" aria-hidden="true" />
							{:else}
								{i + 1}
							{/if}
						</span>
						<span class="setup-progress-label">{s.label}</span>
						<span class="sr-only">{done ? ', done' : ', not done yet'}</span>
					</a>
				</li>
			{/each}
		</ol>
	</nav>

	{#if error}
		<Alert type="error" title="Error" class="mb-4">
			<p>{error}</p>
		</Alert>
	{/if}

	<section aria-labelledby="setup-step-heading" class="space-y-4">
		<div>
			<p class="text-xs font-medium text-text-muted">Step {index + 1} of {SETUP_STEPS.length}</p>
			<div class="flex flex-wrap items-center gap-2 mt-1">
				<h2 id="setup-step-heading" tabindex="-1" bind:this={heading} class="text-xl font-semibold text-text focus:outline-none">
					{info.title}
				</h2>
				{#if status?.done}
					<Badge variant="success" size="sm">Done</Badge>
				{:else}
					<Badge variant="default" size="sm">Not done yet</Badge>
				{/if}
			</div>
			<p class="form-hint mt-1">{INTROS[current]}</p>
			{#if status && !status.done && status.needs}
				<p class="text-sm text-text mt-2"><span class="text-text-muted">Still needs:</span> {status.needs}.</p>
			{/if}
		</div>

		{#if current === 'business'}
			{#key reloads}
				<ProfileForm bind:profile onsaved={changed}>
					{#snippet assist()}
						<AskAI
							prompt={askAIPrompt('business', { brand: brandSlug, website: profile.website?.trim() })}
							disabled={!profile.website?.trim()}
							hint={profile.website?.trim() ? '' : 'Enter the website above first.'}
							onrefresh={reread}
						/>
					{/snippet}
				</ProfileForm>
			{/key}
		{:else if current === 'targeting'}
			<AskAI
				prompt={askAIPrompt('targeting', { brand: brandSlug })}
				disabled={!setup.find((s) => s.step === 'business')?.done}
				hint={setup.find((s) => s.step === 'business')?.done ? '' : 'The AI proposes segments from the profile: finish Your business first.'}
				onrefresh={reread}
			/>
			{#key reloads}
				<SegmentsEditor onchange={changed} />
			{/key}
		{:else if current === 'sending'}
			<SendingSetup onchange={changed} />
		{:else if current === 'do_not_contact'}
			<DncLists onchange={changed} />
			<p class="form-hint">
				More ways to keep people out, such as skipping everyone at a customer's company or asking your CRM, are on
				<a href="{base}/do-not-contact" class="link link-primary">Do not contact</a>.
			</p>
		{:else if current === 'campaign'}
			{#if status?.done}
				<Alert type="success" title={createdId ? 'Campaign created' : 'Your first campaign is in'}>
					<p>
						Its first emails wait on <a href="{base}/drafts" class="link link-primary">Drafts</a> for approval, unless it
						sends without approval.
						{#if createdId}
							<a href="{base}/campaigns/{createdId}" class="link link-primary">Open the campaign</a>.
						{:else}
							See it on <a href="{base}/campaigns" class="link link-primary">Campaigns</a>.
						{/if}
					</p>
				</Alert>
			{/if}
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div>
					<label for="gs-segment" class="form-label">Segment</label>
					<Select
						id="gs-segment"
						bind:value={aiSegment}
						disabled={segments.length === 0}
						options={segments.map((s) => ({ value: s.name, label: s.name }))}
					/>
				</div>
				<div>
					<label for="gs-approval" class="form-label">Approval</label>
					<Select id="gs-approval" bind:value={aiApproval} options={APPROVAL_MODES} />
				</div>
			</div>
			<AskAI
				prompt={askAIPrompt('campaign', { brand: brandSlug, segment: aiSegment, approval: aiApproval })}
				disabled={!aiSegment}
				hint={aiSegment ? '' : 'Make a segment in Who to target first.'}
				onrefresh={reread}
			/>
			<div>
				<Button type="secondary" size="sm" onclick={() => (writeOwn = !writeOwn)}>
					{writeOwn ? 'Hide the campaign form' : 'Write the emails yourself'}
				</Button>
			</div>
			{#if writeOwn}
				<div class="rounded-lg border border-border p-4">
					<CampaignForm oncreated={created} />
				</div>
			{/if}
		{/if}
	</section>

	<div class="setup-footer">
		{#if index > 0}
			<Button type="secondary" href={stepHref(SETUP_STEPS[index - 1].step)}>
				<ArrowLeft class="mr-2 h-4 w-4" />
				Back
			</Button>
		{:else}
			<span></span>
		{/if}
		{#if index < SETUP_STEPS.length - 1}
			<Button type="primary" href={stepHref(SETUP_STEPS[index + 1].step)}>
				{status?.done ? 'Next' : 'Skip this step'}
				<ArrowRight class="ml-2 h-4 w-4" />
			</Button>
		{:else if allDone}
			<Button type="primary" href={base}>Go to Overview</Button>
		{:else}
			<Button type="primary" onclick={skip}>Finish later</Button>
		{/if}
	</div>
{/if}
