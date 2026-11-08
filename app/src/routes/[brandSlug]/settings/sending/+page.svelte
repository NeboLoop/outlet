<script lang="ts">
	import {
		listSendingPathways,
		createSendingPathway,
		updateSendingPathway,
		deleteSendingPathway,
		moveSendingDomain,
		type SendingPathway,
		type SendingPathwayDomain,
		type SendingLimits
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Badge, Input, Select, Checkbox, SlideForm } from '$lib/components/ui';
	import { Plus, Pencil, Trash2, TriangleAlert } from '@lucide/svelte';
	import { errorMessage } from '../../outbound/shared';
	import { moveTargets } from './pathways';

	const OUTREACH_WARNING =
		'Amazon SES and most email platforms forbid cold email in their terms and can suspend the account.';

	const PROVIDERS = [
		{ value: 'platform', label: "This server's email settings" },
		{ value: 'ses', label: 'Your own Amazon SES' },
		{ value: 'smtp', label: 'Your own SMTP relay' },
		{ value: 'mailboxes', label: 'Your connected mailboxes' }
	];
	const KINDS = [
		{ value: 'transactional', label: 'Transactional', hint: 'Receipts, sign-in links, API sends' },
		{ value: 'marketing', label: 'Marketing', hint: 'Campaigns and sequences' },
		{ value: 'outreach', label: 'Outreach', hint: 'Cold email to prospects' }
	];

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let pathways = $state<SendingPathway[]>([]);

	// The form adds a pathway (editing === '') or changes one.
	let showForm = $state(false);
	let editing = $state('');
	let saving = $state(false);
	let name = $state('');
	let provider = $state('platform');
	let kinds = $state<Record<string, boolean>>({ transactional: false, marketing: false, outreach: false });
	let awsRegion = $state('');
	let awsAccessKey = $state('');
	let awsSecretKey = $state('');
	let smtpHost = $state('');
	let smtpPort = $state(587);
	let smtpUsername = $state('');
	let smtpPassword = $state('');
	let limits = $state<SendingLimits | null>(null);
	let bouncePercent = $state(3);
	let current = $state<SendingPathway | null>(null);
	let deleting = $state('');
	// The pathway each domain is to move to, by domain id.
	let moveTo = $state<Record<string, string>>({});
	let moving = $state('');

	let showWarning = $derived(kinds.outreach && provider !== 'mailboxes');

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			pathways = (await listSendingPathways()).pathways || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load the sending pathways');
		} finally {
			loading = false;
		}
	}

	function providerLabel(value: string): string {
		return PROVIDERS.find((p) => p.value === value)?.label || value;
	}

	function kindLabel(value: string): string {
		return KINDS.find((k) => k.value === value)?.label || value;
	}

	function openAdd() {
		editing = '';
		current = null;
		name = '';
		provider = 'platform';
		kinds = { transactional: false, marketing: false, outreach: false };
		awsRegion = '';
		awsAccessKey = '';
		awsSecretKey = '';
		smtpHost = '';
		smtpPort = 587;
		smtpUsername = '';
		smtpPassword = '';
		limits = null;
		showForm = true;
	}

	function openEdit(p: SendingPathway) {
		editing = p.id;
		current = p;
		name = p.name;
		provider = p.provider;
		kinds = {
			transactional: p.kinds.includes('transactional'),
			marketing: p.kinds.includes('marketing'),
			outreach: p.kinds.includes('outreach')
		};
		awsRegion = p.aws_region || '';
		awsAccessKey = '';
		awsSecretKey = '';
		smtpHost = p.smtp_host || '';
		smtpPort = p.smtp_port || 587;
		smtpUsername = p.smtp_username || '';
		smtpPassword = '';
		limits = { ...p.limits };
		bouncePercent = Math.round(p.limits.pause_bounce_rate * 1000) / 10;
		showForm = true;
	}

	function pickedKinds(): string[] {
		return KINDS.map((k) => k.value).filter((k) => kinds[k]);
	}

	function numbers(l: SendingLimits): SendingLimits {
		return {
			daily_limit: Number(l.daily_limit) || 0,
			warmup_start: Number(l.warmup_start) || 0,
			warmup_step: Number(l.warmup_step) || 0,
			min_gap_seconds: Number(l.min_gap_seconds) || 0,
			max_gap_seconds: Number(l.max_gap_seconds) || 0,
			pause_bounce_rate: (Number(bouncePercent) || 0) / 100,
			pause_min_bounces: Number(l.pause_min_bounces) || 0,
			pause_spam_blocks: Number(l.pause_spam_blocks) || 0,
			pause_complaints: Number(l.pause_complaints) || 0
		};
	}

	async function save() {
		saving = true;
		error = '';
		notice = '';
		const creds = {
			aws_region: provider === 'ses' ? awsRegion.trim() || undefined : undefined,
			aws_access_key: provider === 'ses' ? awsAccessKey.trim() || undefined : undefined,
			aws_secret_key: provider === 'ses' ? awsSecretKey.trim() || undefined : undefined,
			smtp_host: provider === 'smtp' ? smtpHost.trim() || undefined : undefined,
			smtp_port: provider === 'smtp' ? Number(smtpPort) || undefined : undefined,
			smtp_username: provider === 'smtp' ? smtpUsername.trim() || undefined : undefined,
			smtp_password: provider === 'smtp' ? smtpPassword || undefined : undefined
		};
		try {
			const saved = editing
				? await updateSendingPathway(
						{},
						{ name: name.trim() || undefined, kinds: pickedKinds(), limits: limits ? numbers(limits) : undefined, ...creds },
						editing
					)
				: await createSendingPathway({ name: name.trim() || undefined, provider, kinds: pickedKinds(), ...creds });
			notice = saved.warning ? `${saved.name} saved. ${saved.warning}` : `${saved.name} saved`;
			showForm = false;
			await load();
		} catch (err) {
			error = errorMessage(err, 'Could not save the pathway');
		} finally {
			saving = false;
		}
	}

	async function remove(p: SendingPathway) {
		deleting = p.id;
		error = '';
		notice = '';
		try {
			await deleteSendingPathway({}, p.id);
			notice = `${p.name} removed`;
			await load();
		} catch (err) {
			error = errorMessage(err, 'Could not remove the pathway');
		} finally {
			deleting = '';
		}
	}

	async function move(d: SendingPathwayDomain) {
		const target = moveTo[d.id];
		if (!target) return;
		moving = d.id;
		error = '';
		notice = '';
		try {
			const res = await moveSendingDomain({}, { pathway_id: target }, d.id);
			notice = `${d.domain}: ${res.message}`;
			moveTo = { ...moveTo, [d.id]: '' };
			await load();
		} catch (err) {
			error = errorMessage(err, `Could not move ${d.domain}`);
		} finally {
			moving = '';
		}
	}

	function gap(l: SendingLimits): string {
		if (!l.max_gap_seconds) return 'None';
		const min = Math.round(l.min_gap_seconds / 6) / 10;
		const max = Math.round(l.max_gap_seconds / 6) / 10;
		return min === max ? `${min} min` : `${min} to ${max} min`;
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}
{#if notice}
	<Alert type="success" dismissible onclose={() => (notice = '')} class="mb-4">
		<p>{notice}</p>
	</Alert>
{/if}

<div class="space-y-6">
	<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
		<div>
			<h2 class="text-lg font-semibold text-text">Sending</h2>
			<p class="text-sm text-text-muted mt-1">
				Each kind of mail goes out through one pathway. Ticking a kind on a pathway takes it off the others.
				A kind no pathway carries isn't sent.
			</p>
		</div>
		<Button type="secondary" size="sm" onclick={openAdd} disabled={showForm}>
			<Plus class="mr-2 h-4 w-4" />
			Add pathway
		</Button>
	</div>

	<SlideForm bind:show={showForm} title={editing ? `Edit ${current?.name || 'pathway'}` : 'Add a sending pathway'}>
		<div class="space-y-5">
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div>
					<label for="sp-name" class="form-label">Name</label>
					<Input id="sp-name" bind:value={name} placeholder={providerLabel(provider)} />
				</div>
				<div>
					<label for="sp-provider" class="form-label">Sends through</label>
					{#if editing}
						<p id="sp-provider" class="text-sm text-text py-2">{providerLabel(provider)}</p>
					{:else}
						<Select id="sp-provider" bind:value={provider} options={PROVIDERS} />
					{/if}
				</div>
			</div>

			<div>
				<p class="form-label">Carries</p>
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
					{#each KINDS as k (k.value)}
						<Checkbox bind:checked={kinds[k.value]} label={k.label} description={k.hint} />
					{/each}
				</div>
				{#if showWarning}
					<Alert type="warning" class="mt-3">
						<p>{OUTREACH_WARNING}</p>
					</Alert>
				{/if}
			</div>

			{#if provider === 'ses'}
				<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
					<div>
						<label for="sp-region" class="form-label">AWS region</label>
						<Input id="sp-region" bind:value={awsRegion} placeholder="us-east-1" />
					</div>
					<div>
						<label for="sp-access" class="form-label">Access key</label>
						<Input
							id="sp-access"
							type="password"
							bind:value={awsAccessKey}
							placeholder={current?.aws_keys_set ? 'Set; enter a new one to replace it' : 'AKIA...'}
						/>
					</div>
					<div>
						<label for="sp-secret" class="form-label">Secret key</label>
						<Input
							id="sp-secret"
							type="password"
							bind:value={awsSecretKey}
							placeholder={current?.aws_keys_set ? 'Set; enter a new one to replace it' : ''}
						/>
					</div>
				</div>
			{:else if provider === 'smtp'}
				<div class="grid grid-cols-1 sm:grid-cols-4 gap-4">
					<div class="sm:col-span-2">
						<label for="sp-host" class="form-label">SMTP host</label>
						<Input id="sp-host" bind:value={smtpHost} placeholder="smtp.provider.com" />
					</div>
					<div>
						<label for="sp-port" class="form-label">Port</label>
						<Input id="sp-port" type="number" bind:value={smtpPort} />
					</div>
					<div>
						<label for="sp-user" class="form-label">Username</label>
						<Input id="sp-user" bind:value={smtpUsername} />
					</div>
					<div class="sm:col-span-2">
						<label for="sp-pass" class="form-label">Password</label>
						<Input
							id="sp-pass"
							type="password"
							bind:value={smtpPassword}
							placeholder={current?.smtp_password_set ? 'Set; enter a new one to replace it' : ''}
						/>
					</div>
				</div>
			{:else if provider === 'mailboxes'}
				<p class="form-hint">Connect the pool's mailboxes and their domains in Outbound, Setup.</p>
			{/if}

			{#if limits}
				<div>
					<p class="form-label">Limits per sender</p>
					<p class="form-hint mb-3">
						A sender is {provider === 'mailboxes' ? 'each mailbox of the pool' : 'this pathway'}. 0 turns a
						limit off.
					</p>
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
						<div>
							<label for="sp-daily" class="form-label">Emails a day</label>
							<Input id="sp-daily" type="number" min={0} bind:value={limits.daily_limit} />
						</div>
						<div>
							<label for="sp-wstart" class="form-label">Warm-up first day</label>
							<Input id="sp-wstart" type="number" min={0} bind:value={limits.warmup_start} />
						</div>
						<div>
							<label for="sp-wstep" class="form-label">Warm-up added each day</label>
							<Input id="sp-wstep" type="number" min={0} bind:value={limits.warmup_step} />
						</div>
						<div>
							<label for="sp-gmin" class="form-label">Seconds between sends, at least</label>
							<Input id="sp-gmin" type="number" min={0} bind:value={limits.min_gap_seconds} />
						</div>
						<div>
							<label for="sp-gmax" class="form-label">Seconds between sends, at most</label>
							<Input id="sp-gmax" type="number" min={0} bind:value={limits.max_gap_seconds} />
						</div>
					</div>
					<p class="form-label mt-4">Pause a sender over 7 days at</p>
					<div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
						<div>
							<label for="sp-brate" class="form-label">Bounce rate (%)</label>
							<Input id="sp-brate" type="number" min={0} max={100} step={0.1} bind:value={bouncePercent} />
						</div>
						<div>
							<label for="sp-bmin" class="form-label">With bounces, at least</label>
							<Input id="sp-bmin" type="number" min={0} bind:value={limits.pause_min_bounces} />
						</div>
						<div>
							<label for="sp-spam" class="form-label">Spam blocks</label>
							<Input id="sp-spam" type="number" min={0} bind:value={limits.pause_spam_blocks} />
						</div>
						<div>
							<label for="sp-compl" class="form-label">Complaints</label>
							<Input id="sp-compl" type="number" min={0} bind:value={limits.pause_complaints} />
						</div>
					</div>
				</div>
			{:else}
				<p class="form-hint">It starts with this provider's default limits; edit it afterwards to change them.</p>
			{/if}
		</div>

		{#snippet footer()}
			<Button type="secondary" onclick={() => (showForm = false)} disabled={saving}>Cancel</Button>
			<Button type="primary" onclick={save} disabled={saving}>
				{saving ? 'Saving...' : editing ? 'Save' : 'Add pathway'}
			</Button>
		{/snippet}
	</SlideForm>

	{#if loading}
		<div class="flex justify-center py-12">
			<LoadingSpinner size="large" />
		</div>
	{:else if pathways.length === 0}
		<p class="text-sm text-text-muted">No sending pathways. This brand sends nothing until one carries its mail.</p>
	{:else}
		<div class="space-y-4">
			{#each pathways as p (p.id)}
				<section class="rounded-lg border border-border p-4">
					<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
						<div>
							<h3 class="font-medium text-text">{p.name}</h3>
							<p class="text-sm text-text-muted">{providerLabel(p.provider)}</p>
							<div class="flex flex-wrap gap-2 mt-2">
								{#each p.kinds as k (k)}
									<Badge variant="success" size="sm">{kindLabel(k)}</Badge>
								{:else}
									<Badge size="sm">Carries nothing</Badge>
								{/each}
							</div>
						</div>
						<div class="flex gap-2">
							<Button type="ghost" size="xs" onclick={() => openEdit(p)} disabled={showForm}>
								<Pencil class="mr-1 h-3.5 w-3.5" />
								Edit
							</Button>
							<Button type="ghost" size="xs" onclick={() => remove(p)} disabled={deleting !== '' || p.kinds.length > 0}>
								<Trash2 class="mr-1 h-3.5 w-3.5" />
								{deleting === p.id ? 'Removing...' : 'Remove'}
							</Button>
						</div>
					</div>

					{#if p.warning}
						<div class="flex items-start gap-2 mt-3 text-sm text-warning">
							<TriangleAlert class="h-4 w-4 shrink-0 mt-0.5" />
							<span>{p.warning}</span>
						</div>
					{/if}

					<dl class="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4 text-sm">
						<div>
							<dt class="text-text-muted">Emails a day</dt>
							<dd class="text-text">{p.limits.daily_limit || 'No limit'}</dd>
						</div>
						<div>
							<dt class="text-text-muted">Warm-up</dt>
							<dd class="text-text">
								{p.limits.warmup_start ? `${p.limits.warmup_start}, +${p.limits.warmup_step} a day` : 'None'}
							</dd>
						</div>
						<div>
							<dt class="text-text-muted">Between sends</dt>
							<dd class="text-text">{gap(p.limits)}</dd>
						</div>
						<div>
							<dt class="text-text-muted">Pauses at</dt>
							<dd class="text-text">
								{Math.round(p.limits.pause_bounce_rate * 1000) / 10}% bounces, {p.limits.pause_spam_blocks} spam blocks,
								{p.limits.pause_complaints} complaints
							</dd>
						</div>
					</dl>

					<p class="text-xs text-text-muted mt-3">
						{#if p.provider === 'mailboxes'}
							{p.mailboxes} mailbox{p.mailboxes === 1 ? '' : 'es'}{p.domains.length
								? ` on ${p.domains.map((d) => d.domain).join(', ')}`
								: ''}. Manage them in Outbound, Setup.
						{:else if p.domains.length}
							Sends from {p.domains.map((d) => d.domain).join(', ')}.
						{:else}
							No domains yet.
						{/if}
					</p>

					{#if p.domains.length}
						<div class="mt-3 space-y-2">
							{#each p.domains as d (d.id)}
								{@const targets = moveTargets(pathways, p.id, d.purpose)}
								<div class="flex flex-col sm:flex-row sm:items-center gap-2 text-sm">
									<span class="text-text sm:w-48 truncate">{d.domain}</span>
									{#if targets.length}
										<div class="flex items-center gap-2">
											<label for="move-{d.id}" class="sr-only">Move {d.domain} to</label>
											<Select
												id="move-{d.id}"
												size="sm"
												class="w-52"
												bind:value={moveTo[d.id]}
												options={[
													{ value: '', label: 'Move to...' },
													...targets.map((t) => ({ value: t.id, label: t.name }))
												]}
											/>
											<Button
												type="ghost"
												size="xs"
												onclick={() => move(d)}
												disabled={moving !== '' || !moveTo[d.id]}
											>
												{moving === d.id ? 'Moving...' : 'Move'}
											</Button>
										</div>
									{:else}
										<span class="text-xs text-text-muted">
											{d.purpose === 'outreach'
												? 'Add another mailbox pool to move it.'
												: 'Add another sending pathway to move it.'}
										</span>
									{/if}
								</div>
							{/each}
							<p class="form-hint">
								An outreach domain moves to another mailbox pool, with its mailboxes. A sending domain moves to
								another platform, SES or SMTP pathway.
							</p>
						</div>
					{/if}
				</section>
			{/each}
		</div>
	{/if}
</div>
