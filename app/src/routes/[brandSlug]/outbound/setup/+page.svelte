<script lang="ts">
	import { formatDate } from '$lib/dates';
	import {
		getOutboundOverview,
		setOutboundEnabled,
		updateOutboundSettings,
		setOutboundDNSToken,
		addOutboundDomain,
		checkOutboundDomain,
		addOutboundMailboxes,
		setMailboxDailyLimit,
		type OutboundOverview,
		type OutboundDNSRecord,
		type OutboundDomainRecord,
		type OutboundMailboxInput,
		type OutboundMailboxResult
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Badge, Input, Textarea, Toggle, Select, SlideForm } from '$lib/components/ui';
	import { Plus, Check, X, Copy } from '@lucide/svelte';
	import { page } from '$app/stores';
	import { errorMessage } from '../shared';

	let brandSlug = $derived($page.params.brandSlug);

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let overview = $state<OutboundOverview | null>(null);

	let enabled = $state(false);
	let switching = $state(false);

	let mailingAddress = $state('');
	let optOutLine = $state('');
	let timezone = $state('');
	let savingSettings = $state(false);

	let dnsToken = $state('');
	let savingToken = $state(false);

	let checking = $state('');

	let showDomain = $state(false);
	let addingDomain = $state(false);
	let domainName = $state('');
	let dnsHost = $state('manual');
	let mailProvider = $state('google');
	let mxHosts = $state('');
	let spfInclude = $state('');
	let dkimSelector = $state('google');
	let dkimValue = $state('');
	let dmarcReport = $state('');
	let clickTracking = $state(false);
	let records = $state<OutboundDNSRecord[]>([]);
	let dnsError = $state('');
	let copied = $state('');
	// The domain whose records are shown.
	let showRecords = $state('');

	let showMailbox = $state(false);
	let addingMailbox = $state(false);
	let mbEmail = $state('');
	let mbFromName = $state('');
	let mbSignature = $state('');
	let mbProvider = $state('google');
	let mbPassword = $state('');
	let mbDailyLimit = $state('');
	let smtpHost = $state('');
	let smtpPort = $state(587);
	let smtpUsername = $state('');
	let smtpPassword = $state('');
	let imapHost = $state('');
	let imapPort = $state(993);
	let imapUsername = $state('');
	let imapPassword = $state('');
	let mailboxResults = $state<OutboundMailboxResult[]>([]);

	// Each mailbox's own daily limit, as typed; empty uses its pool's.
	let ownLimits = $state<Record<string, string>>({});
	let savingLimit = $state('');

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			await refresh();
		} catch (err) {
			error = errorMessage(err, 'Could not load outbound setup');
		} finally {
			loading = false;
		}
	}

	async function refresh() {
		const o = await getOutboundOverview();
		overview = o;
		enabled = o.enabled;
		mailingAddress = o.mailing_address;
		optOutLine = o.opt_out_line;
		timezone = o.timezone;
		ownLimits = Object.fromEntries(o.mailboxes.map((m) => [m.id, m.daily_limit ? String(m.daily_limit) : '']));
	}

	async function saveLimit(id: string) {
		savingLimit = id;
		await run('Could not save the daily limit', async () => {
			const value = String(ownLimits[id] ?? '').trim();
			await setMailboxDailyLimit({}, { daily_limit: value === '' ? undefined : Number(value) }, id);
			notice = value === '' ? "The mailbox uses its pool's daily limit" : 'Daily limit saved';
			await refresh();
		});
		savingLimit = '';
	}

	async function run(fallback: string, call: () => Promise<void>) {
		error = '';
		notice = '';
		try {
			await call();
		} catch (err) {
			error = errorMessage(err, fallback);
		}
	}

	async function switchEnabled(on: boolean) {
		switching = true;
		await run(on ? 'Could not turn outbound on' : 'Could not turn outbound off', async () => {
			try {
				await setOutboundEnabled({ enabled: on });
				notice = on ? 'Outbound is on' : 'Outbound is off';
				await refresh();
			} catch (err) {
				enabled = !on;
				throw err;
			}
		});
		switching = false;
	}

	async function saveSettings() {
		savingSettings = true;
		await run('Could not save the settings', async () => {
			await updateOutboundSettings({ mailing_address: mailingAddress, opt_out_line: optOutLine, timezone });
			notice = 'Settings saved';
			await refresh();
		});
		savingSettings = false;
	}

	async function saveToken() {
		savingToken = true;
		await run('Could not save the token', async () => {
			await setOutboundDNSToken({ token: dnsToken.trim() });
			dnsToken = '';
			notice = 'DigitalOcean token saved';
			await refresh();
		});
		savingToken = false;
	}

	async function check(id: string) {
		checking = id;
		await run('Could not check the domain', async () => {
			const d = await checkOutboundDomain({}, id);
			if (overview) overview.domains = overview.domains.map((x) => (x.id === d.id ? d : x));
		});
		checking = '';
	}

	function openDomainForm() {
		domainName = '';
		mxHosts = '';
		spfInclude = '';
		dkimSelector = 'google';
		dkimValue = '';
		dmarcReport = '';
		clickTracking = false;
		records = [];
		dnsError = '';
		showDomain = true;
	}

	async function addDomain() {
		addingDomain = true;
		await run('Could not add the domain', async () => {
			const custom = mailProvider === 'other';
			const res = await addOutboundDomain({
				domain: domainName.trim(),
				dns_host: dnsHost,
				mail_provider: mailProvider,
				mx_hosts: custom ? mxHosts.split(/[\s,]+/).filter((h) => h !== '') : undefined,
				spf_include: custom ? spfInclude.trim() : undefined,
				dkim_selector: dkimSelector.trim() || undefined,
				dkim_value: dkimValue.trim() || undefined,
				dmarc_report: dmarcReport.trim() || undefined,
				click_tracking: clickTracking
			});
			records = res.records_to_add || [];
			dnsError = res.dns_error || '';
			notice = records.length
				? `${res.domain.domain} added. Add the records below at your DNS host, then press Check.`
				: `${res.domain.domain} added`;
			await refresh();
		});
		addingDomain = false;
	}

	async function copy(text: string) {
		await navigator.clipboard.writeText(text);
		copied = text;
	}

	function openMailboxForm() {
		mbEmail = '';
		mbFromName = '';
		mbSignature = '';
		mbPassword = '';
		mbDailyLimit = '';
		smtpHost = '';
		smtpPort = 587;
		smtpUsername = '';
		smtpPassword = '';
		imapHost = '';
		imapPort = 993;
		imapUsername = '';
		imapPassword = '';
		mailboxResults = [];
		showMailbox = true;
	}

	async function addMailbox() {
		addingMailbox = true;
		await run('Could not add the mailbox', async () => {
			const input: OutboundMailboxInput = {
				email: mbEmail.trim(),
				from_name: mbFromName.trim(),
				signature: mbSignature || undefined,
				password: mbPassword || undefined,
				daily_limit: String(mbDailyLimit).trim() === '' ? undefined : Number(mbDailyLimit)
			};
			if (mbProvider === 'google') {
				input.provider = 'google';
			} else {
				input.smtp_host = smtpHost.trim();
				input.smtp_port = Number(smtpPort) || undefined;
				input.smtp_username = smtpUsername.trim() || undefined;
				input.smtp_password = smtpPassword || undefined;
				input.imap_host = imapHost.trim();
				input.imap_port = Number(imapPort) || undefined;
				input.imap_username = imapUsername.trim() || undefined;
				input.imap_password = imapPassword || undefined;
			}
			const res = await addOutboundMailboxes({ mailboxes: [input] });
			mailboxResults = res.mailboxes || [];
			if (mailboxResults.every((m) => !m.error)) {
				notice = `${input.email} added`;
				showMailbox = false;
			}
			await refresh();
		});
		addingMailbox = false;
	}
</script>

{#snippet recordTable(rows: (OutboundDNSRecord | OutboundDomainRecord)[])}
	<div class="data-table overflow-x-auto">
		<table class="w-full text-sm">
			<thead>
				<tr>
					<th class="text-left">Type</th>
					<th class="text-left">Name</th>
					<th class="text-left">Value</th>
					<th class="text-left">Priority</th>
					{#if rows.some((r) => 'present' in r)}
						<th class="text-left">Found</th>
					{/if}
				</tr>
			</thead>
			<tbody>
				{#each rows as r}
					<tr>
						<td>{r.type}</td>
						<td><code class="text-xs">{r.name}</code></td>
						<td>
							<button type="button" class="inline-flex items-start gap-2 text-left" onclick={() => copy(r.data)}>
								<code class="text-xs break-all">{r.data}</code>
								{#if copied === r.data}
									<Check class="h-4 w-4 shrink-0 text-success" />
								{:else}
									<Copy class="h-4 w-4 shrink-0 text-text-muted" />
								{/if}
							</button>
						</td>
						<td>{r.priority || ''}</td>
						{#if 'present' in r}
							<td>{@render mark(r.present, r.present ? 'Found' : 'Missing')}</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/snippet}

{#snippet mark(ok: boolean | undefined, label: string)}
	<span class="inline-flex items-center gap-1 text-xs" class:text-success={ok} class:text-error={!ok}>
		{#if ok}<Check class="h-3.5 w-3.5" />{:else}<X class="h-3.5 w-3.5" />{/if}
		{label}
	</span>
{/snippet}

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

{#if loading}
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if overview}
	<div class="space-y-8">
		<section class="rounded-lg border border-border p-4">
			<Toggle
				bind:checked={enabled}
				disabled={switching}
				onchange={switchEnabled}
				label="Outbound is {enabled ? 'on' : 'off'}"
				description="Only the brand's admin can turn it on. Outreach goes through the sending pathway that carries it, normally the mailboxes below."
			/>
		</section>

		<section>
			<h2 class="text-lg font-semibold text-text mb-3">Settings</h2>
			<div class="space-y-4">
				<div>
					<label for="ob-address" class="form-label">Mailing address</label>
					<Textarea id="ob-address" bind:value={mailingAddress} rows={2} placeholder="123 Main St, Denver, CO 80202" />
					<p class="form-hint">Every outreach email carries it. Required before any campaign.</p>
				</div>
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label for="ob-optout" class="form-label">Opt-out line</label>
						<Input id="ob-optout" bind:value={optOutLine} placeholder="Not for you? Just reply and I won't write again." />
					</div>
					<div>
						<label for="ob-tz" class="form-label">Time zone</label>
						<Input id="ob-tz" bind:value={timezone} placeholder="America/Denver" />
					</div>
				</div>
				<div class="flex justify-end">
					<Button type="primary" onclick={saveSettings} disabled={savingSettings}>
						{savingSettings ? 'Saving...' : 'Save settings'}
					</Button>
				</div>
			</div>
		</section>

		<section>
			<div class="flex items-center gap-2 mb-3">
				<h2 class="text-lg font-semibold text-text">DigitalOcean DNS token</h2>
				{#if overview.dns_token_set}
					<Badge variant="success" size="sm">Set</Badge>
				{/if}
			</div>
			<p class="form-hint mb-3">With a token, domains hosted on DigitalOcean get their mail records written for you.</p>
			<div class="flex flex-col sm:flex-row gap-3">
				<div class="flex-1">
					<label for="ob-token" class="sr-only">Token</label>
					<Input
						id="ob-token"
						type="password"
						bind:value={dnsToken}
						placeholder={overview.dns_token_set ? 'Enter a new token to replace it' : 'dop_v1_...'}
					/>
				</div>
				<Button type="primary" onclick={saveToken} disabled={savingToken || !dnsToken.trim()}>
					{savingToken ? 'Saving...' : 'Save token'}
				</Button>
			</div>
		</section>

		<section>
			<div class="flex items-center justify-between gap-4 mb-3">
				<h2 class="text-lg font-semibold text-text">Domains</h2>
				<Button type="secondary" size="sm" onclick={openDomainForm} disabled={showDomain}>
					<Plus class="mr-2 h-4 w-4" />
					Add domain
				</Button>
			</div>

			<SlideForm bind:show={showDomain} title="Add an outreach domain">
				<div class="space-y-4">
					<p class="form-hint">Use a domain of its own for outreach, not the brand's main domain.</p>
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
						<div>
							<label for="dm-domain" class="form-label">Domain</label>
							<Input id="dm-domain" bind:value={domainName} placeholder="trybrand.com" />
						</div>
						<div>
							<label for="dm-host" class="form-label">DNS host</label>
							<Select
								id="dm-host"
								bind:value={dnsHost}
								options={[
									{ value: 'manual', label: 'I add records myself' },
									{ value: 'digitalocean', label: 'DigitalOcean' }
								]}
							/>
						</div>
						<div>
							<label for="dm-provider" class="form-label">Mail provider</label>
							<Select
								id="dm-provider"
								bind:value={mailProvider}
								options={[
									{ value: 'google', label: 'Google Workspace' },
									{ value: 'other', label: 'Other' }
								]}
							/>
						</div>
					</div>
					{#if mailProvider === 'other'}
						<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
							<div>
								<label for="dm-mx" class="form-label">MX hosts</label>
								<Input id="dm-mx" bind:value={mxHosts} placeholder="mx1.provider.com, mx2.provider.com" />
							</div>
							<div>
								<label for="dm-spf" class="form-label">SPF include</label>
								<Input id="dm-spf" bind:value={spfInclude} placeholder="_spf.provider.com" />
							</div>
						</div>
					{/if}
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
						<div>
							<label for="dm-selector" class="form-label">DKIM selector</label>
							<Input id="dm-selector" bind:value={dkimSelector} placeholder="google" />
						</div>
						<div class="sm:col-span-2">
							<label for="dm-dkim" class="form-label">DKIM value</label>
							<Input id="dm-dkim" bind:value={dkimValue} placeholder="v=DKIM1; k=rsa; p=..." />
							<p class="form-hint">From your mail provider's admin console.</p>
						</div>
					</div>
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-4 items-end">
						<div>
							<label for="dm-dmarc" class="form-label">DMARC reports to (optional)</label>
							<Input id="dm-dmarc" bind:value={dmarcReport} placeholder="dmarc@trybrand.com" />
						</div>
						<Toggle bind:checked={clickTracking} label="Click tracking" description="Adds go.{domainName || 'yourdomain'} for tracked links" />
					</div>

					{#if dnsError}
						<Alert type="warning" title="DigitalOcean didn't take the records">
							<p>{dnsError}</p>
						</Alert>
					{/if}
					{#if records.length}
						<div>
							<p class="text-sm font-medium text-text mb-2">Add these records at your DNS host</p>
							{@render recordTable(records)}
						</div>
					{/if}
				</div>

				{#snippet footer()}
					<Button type="secondary" onclick={() => (showDomain = false)} disabled={addingDomain}>Close</Button>
					<Button type="primary" onclick={addDomain} disabled={addingDomain || !domainName.trim()}>
						{addingDomain ? 'Adding...' : 'Add domain'}
					</Button>
				{/snippet}
			</SlideForm>

			{#if overview.domains.length === 0}
				<p class="text-sm text-text-muted">No outreach domains yet. Add one here, or let the AI add one over MCP.</p>
			{:else}
				<div class="data-table overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr>
								<th class="text-left">Domain</th>
								<th class="text-left">Records</th>
								<th class="text-left hidden sm:table-cell">Checked</th>
								<th class="text-right"></th>
							</tr>
						</thead>
						<tbody>
							{#each overview.domains as d (d.id)}
								<tr>
									<td>
										<span class="font-medium text-text block">{d.domain}</span>
										<Badge
											variant={d.status === 'success' ? 'success' : d.status === 'failed' ? 'error' : 'warning'}
											size="sm"
										>
											{d.status === 'success' ? 'Ready' : d.status === 'failed' ? 'Missing records' : 'Waiting'}
										</Badge>
									</td>
									<td>
										<div class="flex flex-wrap gap-3">
											{@render mark(d.check.mx, 'MX')}
											{@render mark(d.check.spf, 'SPF')}
											{@render mark(d.check.dkim, 'DKIM')}
											{@render mark(d.check.dmarc, 'DMARC')}
											{#if d.tracking_host}
												{@render mark(d.check.tracking, 'Tracking')}
											{/if}
										</div>
										{#if d.check.missing?.length}
											<span class="text-xs text-error block mt-1">Missing: {d.check.missing.join(', ')}</span>
										{/if}
									</td>
									<td class="hidden sm:table-cell text-text-muted whitespace-nowrap">{formatDate(d.checked_at, 'datetime')}</td>
									<td class="text-right whitespace-nowrap">
										<Button type="ghost" size="xs" onclick={() => (showRecords = showRecords === d.id ? '' : d.id)}>
											{showRecords === d.id ? 'Hide records' : 'Records'}
										</Button>
										<Button type="ghost" size="xs" onclick={() => check(d.id)} disabled={checking !== ''}>
											{checking === d.id ? 'Checking...' : 'Check'}
										</Button>
									</td>
								</tr>
								{#if showRecords === d.id}
									<tr>
										<td colspan="4">
											{#if d.records.length}
												<p class="form-hint mb-2">
													The records {d.domain} needs at its DNS host. Found is as of the last check.
												</p>
												{@render recordTable(d.records)}
											{:else}
												<p class="form-hint">
													No records are kept for {d.domain}. Add the domain again with the same settings to see them.
												</p>
											{/if}
										</td>
									</tr>
								{/if}
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>

		<section>
			<div class="flex items-center justify-between gap-4 mb-3">
				<h2 class="text-lg font-semibold text-text">Mailboxes</h2>
				<Button type="secondary" size="sm" onclick={openMailboxForm} disabled={showMailbox}>
					<Plus class="mr-2 h-4 w-4" />
					Add mailbox
				</Button>
			</div>
			<p class="form-hint mb-3">
				The mailboxes form a pool. Its daily limit, warm-up, spacing between sends and health pauses are set in
				<a href="/{brandSlug}/settings/sending" class="link link-primary">Settings, Sending</a>. A mailbox can have a daily
				limit of its own below.
			</p>

			<SlideForm bind:show={showMailbox} title="Add a mailbox">
				<div class="space-y-4">
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
						<div>
							<label for="mb-email" class="form-label">Email</label>
							<Input id="mb-email" type="email" bind:value={mbEmail} placeholder="alex@trybrand.com" />
							<p class="form-hint">Must be on one of the outreach domains above.</p>
						</div>
						<div>
							<label for="mb-from" class="form-label">From name</label>
							<Input id="mb-from" bind:value={mbFromName} placeholder="Alex Rivera" />
							<p class="form-hint">The real person who stands behind this mailbox.</p>
						</div>
					</div>
					<div>
						<label for="mb-signature" class="form-label">Signature (optional)</label>
						<Textarea id="mb-signature" bind:value={mbSignature} rows={3} />
					</div>
					<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
						<div>
							<label for="mb-provider" class="form-label">Provider</label>
							<Select
								id="mb-provider"
								bind:value={mbProvider}
								options={[
									{ value: 'google', label: 'Google Workspace' },
									{ value: 'custom', label: 'Custom (SMTP and IMAP)' }
								]}
							/>
						</div>
						<div>
							<label for="mb-password" class="form-label">{mbProvider === 'google' ? 'App password' : 'Password'}</label>
							<Input id="mb-password" type="password" bind:value={mbPassword} />
						</div>
						<div>
							<label for="mb-cap" class="form-label">Own daily limit (optional)</label>
							<Input id="mb-cap" type="number" min={1} bind:value={mbDailyLimit} placeholder="The pool's" />
							<p class="form-hint">Empty uses the pool's limit.</p>
						</div>
					</div>
					{#if mbProvider === 'google'}
						<p class="form-hint">For Google Workspace, turn on 2-step verification and use an app password.</p>
					{:else}
						<div class="grid grid-cols-1 sm:grid-cols-4 gap-4">
							<div class="sm:col-span-2">
								<label for="mb-smtp-host" class="form-label">SMTP host</label>
								<Input id="mb-smtp-host" bind:value={smtpHost} placeholder="smtp.provider.com" />
							</div>
							<div>
								<label for="mb-smtp-port" class="form-label">SMTP port</label>
								<Input id="mb-smtp-port" type="number" bind:value={smtpPort} />
							</div>
							<div>
								<label for="mb-smtp-user" class="form-label">SMTP username</label>
								<Input id="mb-smtp-user" bind:value={smtpUsername} placeholder="Same as email" />
							</div>
							<div class="sm:col-span-2">
								<label for="mb-imap-host" class="form-label">IMAP host</label>
								<Input id="mb-imap-host" bind:value={imapHost} placeholder="imap.provider.com" />
							</div>
							<div>
								<label for="mb-imap-port" class="form-label">IMAP port</label>
								<Input id="mb-imap-port" type="number" bind:value={imapPort} />
							</div>
							<div>
								<label for="mb-imap-user" class="form-label">IMAP username</label>
								<Input id="mb-imap-user" bind:value={imapUsername} placeholder="Same as email" />
							</div>
							<div class="sm:col-span-2">
								<label for="mb-smtp-pass" class="form-label">SMTP password (if different)</label>
								<Input id="mb-smtp-pass" type="password" bind:value={smtpPassword} />
							</div>
							<div class="sm:col-span-2">
								<label for="mb-imap-pass" class="form-label">IMAP password (if different)</label>
								<Input id="mb-imap-pass" type="password" bind:value={imapPassword} />
							</div>
						</div>
					{/if}
					{#each mailboxResults.filter((m) => m.error) as m}
						<Alert type="error" title={m.email}>
							<p>{m.error}</p>
						</Alert>
					{/each}
				</div>

				{#snippet footer()}
					<Button type="secondary" onclick={() => (showMailbox = false)} disabled={addingMailbox}>Cancel</Button>
					<Button
						type="primary"
						onclick={addMailbox}
						disabled={addingMailbox || !mbEmail.trim() || !mbFromName.trim()}
					>
						{addingMailbox ? 'Checking the login...' : 'Add mailbox'}
					</Button>
				{/snippet}
			</SlideForm>

			{#if overview.mailboxes.length === 0}
				<p class="text-sm text-text-muted">No mailboxes yet. Add one here, or let the AI add one over MCP.</p>
			{:else}
				<div class="data-table overflow-x-auto">
					<table class="w-full text-sm">
						<thead>
							<tr>
								<th class="text-left">Mailbox</th>
								<th class="text-left">Status</th>
								<th class="text-right">Sent today</th>
								<th class="text-left">Own daily limit</th>
							</tr>
						</thead>
						<tbody>
							{#each overview.mailboxes as m (m.id)}
								<tr>
									<td>
										<span class="font-medium text-text block">{m.email}</span>
										<span class="text-xs text-text-muted">{m.from_name}</span>
									</td>
									<td>
										<Badge variant={m.status === 'paused' ? 'error' : 'success'} size="sm">
											{m.status === 'paused' ? 'Paused' : 'Active'}
										</Badge>
										{#if m.reason}
											<span class="text-xs text-error block mt-1">{m.reason}</span>
										{/if}
									</td>
									<td class="text-right whitespace-nowrap">
										{m.sent_today} of {m.limited ? m.allowed_today : 'no limit'}
									</td>
									<td>
										<div class="flex items-center gap-2">
											<label for="limit-{m.id}" class="sr-only">Own daily limit for {m.email}</label>
											<Input
												id="limit-{m.id}"
												type="number"
												size="sm"
												min={1}
												bind:value={ownLimits[m.id]}
												placeholder="The pool's"
											/>
											<Button type="ghost" size="xs" onclick={() => saveLimit(m.id)} disabled={savingLimit !== ''}>
												{savingLimit === m.id ? 'Saving...' : 'Save'}
											</Button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</section>
	</div>
{/if}
