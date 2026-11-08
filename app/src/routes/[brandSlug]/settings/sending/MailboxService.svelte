<script lang="ts">
	import {
		setMailboxService,
		getMailboxServiceInventory,
		connectMailboxServiceDomain,
		quoteMailboxService,
		buyMailboxService,
		syncMailboxService,
		deleteMailboxServiceMailbox,
		getOutboundOverview,
		type SendingPathway,
		type MailboxServiceInventory,
		type MailboxServiceDomainResponse,
		type MailboxServiceQuote,
		type MailboxServiceBuyResponse,
		type MailboxServiceSyncResponse,
		type MailboxServicePerson,
		type OutboundMailboxHealth
	} from '$lib/api';
	import { Button, Alert, Input, Select, Textarea, Badge, AlertDialog } from '$lib/components/ui';
	import { formatDate } from '$lib/dates';
	import { errorMessage } from '../../outbound/shared';

	// One mailbox pool's mailbox service: where its domains and mailboxes come
	// from. Buying is a quote, then a buy the person confirms with the price
	// in front of them.

	let { pool, onchange }: { pool: SendingPathway; onchange: () => void | Promise<void> } = $props();

	const SERVICES = [
		{ value: 'manual', label: 'By hand (SMTP and IMAP logins)' },
		{ value: 'maildoso', label: 'Maildoso' },
		{ value: 'cheapinboxes', label: 'CheapInboxes' },
		{ value: 'infraforge', label: 'Infraforge' }
	];
	const KINDS = [
		{ value: 'mailboxes', label: 'Mailboxes on a domain' },
		{ value: 'domain', label: 'A new domain' },
		{ value: 'connect', label: 'Connect a domain I own (Infraforge)' }
	];

	let service = $state('manual');
	let apiKey = $state('');
	let busy = $state('');
	let error = $state('');
	let notice = $state('');

	let connectDomain = $state('');
	let connected = $state<MailboxServiceDomainResponse | null>(null);

	let kind = $state('mailboxes');
	let orderDomain = $state('');
	let people = $state('');
	let quote = $state<MailboxServiceQuote | null>(null);
	let confirmBuy = $state(false);
	let bought = $state<MailboxServiceBuyResponse | null>(null);

	let inventory = $state<MailboxServiceInventory | null>(null);
	let synced = $state<MailboxServiceSyncResponse | null>(null);
	let mailboxes = $state<OutboundMailboxHealth[]>([]);
	let deleting = $state<OutboundMailboxHealth | null>(null);
	let confirmDelete = $state(false);

	let active = $derived((pool.mailbox_service || 'manual') !== 'manual' && pool.mailbox_service_key_set);
	let serviceLabel = $derived(SERVICES.find((s) => s.value === pool.mailbox_service)?.label || 'the service');

	$effect(() => {
		service = pool.mailbox_service || 'manual';
	});

	$effect(() => {
		if (active) loadMailboxes();
	});

	async function run(name: string, fn: () => Promise<void>) {
		busy = name;
		error = '';
		notice = '';
		try {
			await fn();
		} catch (err) {
			error = errorMessage(err, 'The mailbox service refused');
		} finally {
			busy = '';
		}
	}

	async function loadMailboxes() {
		try {
			const overview = await getOutboundOverview();
			mailboxes = (overview.mailboxes || []).filter((m) => m.pool_id === pool.id);
		} catch {
			mailboxes = [];
		}
	}

	function saveService() {
		return run('service', async () => {
			const st = await setMailboxService({}, { service, api_key: apiKey.trim() || undefined }, pool.id);
			apiKey = '';
			notice =
				st.service === 'manual'
					? 'This pool connects mailboxes by hand.'
					: `Saved. The pool's limits are now ${st.limits.daily_limit} a day, warming up from ${st.limits.warmup_start}.`;
			await onchange();
		});
	}

	function connect() {
		return run('connect', async () => {
			connected = await connectMailboxServiceDomain({}, { domain: connectDomain.trim() }, pool.id);
			connectDomain = '';
			await onchange();
		});
	}

	// Each line is "First Last", or "First Last, local" to choose the address.
	function parsePeople(text: string): MailboxServicePerson[] {
		return text
			.split('\n')
			.map((line) => line.trim())
			.filter(Boolean)
			.map((line) => {
				const [name, local] = line.split(',').map((s) => s.trim());
				const parts = name.split(/\s+/);
				return { first_name: parts[0] || '', last_name: parts.slice(1).join(' '), local: local || undefined };
			});
	}

	function getQuote() {
		return run('quote', async () => {
			bought = null;
			quote = await quoteMailboxService(
				{},
				{ kind, domain: orderDomain.trim(), people: kind === 'connect' ? undefined : parsePeople(people) },
				pool.id
			);
		});
	}

	function price(q: MailboxServiceQuote): string {
		if (!q.price_known || q.price_cents === undefined) return 'Price unknown';
		return `$${(q.price_cents / 100).toFixed(2)}${q.currency && q.currency !== 'USD' ? ' ' + q.currency : ''}`;
	}

	function buy() {
		const q = quote;
		if (!q) return;
		return run('buy', async () => {
			bought = await buyMailboxService({}, { quote_id: q.id, confirm: true }, pool.id);
			quote = null;
			people = '';
			await onchange();
			await loadMailboxes();
		});
	}

	function showInventory() {
		return run('inventory', async () => {
			inventory = await getMailboxServiceInventory({}, pool.id);
		});
	}

	function sync(importNew: boolean) {
		return run(importNew ? 'import' : 'sync', async () => {
			synced = await syncMailboxService({}, { import: importNew }, pool.id);
			if (importNew) {
				await onchange();
				await loadMailboxes();
			}
		});
	}

	function remove() {
		const mb = deleting;
		if (!mb) return;
		return run('delete', async () => {
			await deleteMailboxServiceMailbox({}, pool.id, mb.id);
			notice = `${mb.email} deleted at ${serviceLabel}; it stays here, paused.`;
			deleting = null;
			await loadMailboxes();
		});
	}
</script>

<div class="mt-4 border-t border-border pt-4 space-y-4">
	<div>
		<h4 class="font-medium text-text">Mailbox service</h4>
		<p class="text-sm text-text-muted">Where this pool's domains and mailboxes come from.</p>
	</div>

	{#if error}
		<Alert type="error"><p>{error}</p></Alert>
	{/if}
	{#if notice}
		<Alert type="success" dismissible onclose={() => (notice = '')}><p>{notice}</p></Alert>
	{/if}

	<div class="grid grid-cols-1 sm:grid-cols-3 gap-3 items-end">
		<div>
			<label for="ms-service-{pool.id}" class="form-label">Service</label>
			<Select id="ms-service-{pool.id}" bind:value={service} options={SERVICES} />
		</div>
		{#if service !== 'manual'}
			<div>
				<label for="ms-key-{pool.id}" class="form-label">API key</label>
				<Input
					id="ms-key-{pool.id}"
					type="password"
					bind:value={apiKey}
					placeholder={pool.mailbox_service === service && pool.mailbox_service_key_set
						? 'Set; enter a new one to replace it'
						: 'Paste the API key'}
				/>
			</div>
		{/if}
		<div>
			<Button type="secondary" onclick={saveService} disabled={busy !== ''}>
				{busy === 'service' ? 'Checking the key...' : 'Save'}
			</Button>
		</div>
	</div>
	{#if service === 'maildoso'}
		<p class="form-hint">
			Maildoso's terms cap cold email at 15 a day per mailbox, after 14 days of warm-up, to business addresses only.
			Picking it sets this pool to 15 a day, warming up from 5. You can change the limits afterwards.
		</p>
	{/if}

	{#if active}
		<div class="space-y-2">
			<p class="form-label">Connect a domain you own</p>
			<p class="form-hint">
				{serviceLabel} runs the domain's DNS: you set the nameservers it gives at the domain's registrar.
				{#if pool.mailbox_service === 'infraforge'}Infraforge bills for this, so use a quote below.{/if}
			</p>
			{#if pool.mailbox_service !== 'infraforge'}
				<div class="flex flex-col sm:flex-row gap-2">
					<Input bind:value={connectDomain} placeholder="getexample.com" />
					<Button type="secondary" onclick={connect} disabled={busy !== '' || !connectDomain.trim()}>
						{busy === 'connect' ? 'Connecting...' : 'Connect'}
					</Button>
				</div>
			{/if}
			{#if connected}
				<div class="rounded-lg bg-base-200 p-3 text-sm space-y-1">
					<p class="text-text">{connected.connection.domain}: {connected.connection.note}</p>
					{#if connected.connection.nameservers?.length}
						<p class="text-text-muted">Nameservers: {connected.connection.nameservers.join(', ')}</p>
					{/if}
				</div>
			{/if}
		</div>

		<div class="space-y-2">
			<p class="form-label">Buy</p>
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
				<Select bind:value={kind} options={KINDS} />
				<Input bind:value={orderDomain} placeholder="getexample.com" />
			</div>
			{#if kind !== 'connect'}
				<Textarea
					bind:value={people}
					rows={3}
					placeholder={'One mailbox a line: First Last, or First Last, local\nSam Rivera\nPat Rivera, pat'}
				/>
			{/if}
			<Button type="secondary" onclick={getQuote} disabled={busy !== '' || !orderDomain.trim()}>
				{busy === 'quote' ? 'Pricing...' : 'Get a quote'}
			</Button>
			{#if quote}
				<div class="rounded-lg bg-base-200 p-3 text-sm space-y-2">
					<p class="text-text font-medium">{price(quote)}</p>
					<p class="text-text">{quote.summary}</p>
					<p class="text-text-muted">Good until {formatDate(quote.expires_at, 'short')}.</p>
					<Button type="primary" size="sm" onclick={() => (confirmBuy = true)} disabled={busy !== ''}>
						{busy === 'buy' ? 'Buying...' : 'Buy'}
					</Button>
				</div>
			{/if}
			{#if bought}
				<div class="rounded-lg bg-base-200 p-3 text-sm space-y-1">
					<p class="text-text">{bought.summary}</p>
					{#if bought.connection.nameservers?.length}
						<p class="text-text-muted">Set the nameservers: {bought.connection.nameservers.join(', ')}</p>
					{/if}
					{#each bought.imported as r (r.email)}
						<p class={r.error ? 'text-error' : 'text-text'}>{r.email}: {r.error || 'connected'}</p>
					{/each}
					{#each bought.pending as p (p)}
						<p class="text-text-muted">{p}</p>
					{/each}
					{#if bought.pending.length}
						<p class="text-text-muted">Sync and import once the service has finished them.</p>
					{/if}
				</div>
			{/if}
		</div>

		<div class="space-y-2">
			<p class="form-label">Sync with {serviceLabel}</p>
			<div class="flex flex-wrap gap-2">
				<Button type="ghost" size="sm" onclick={showInventory} disabled={busy !== ''}>
					{busy === 'inventory' ? 'Loading...' : `What ${serviceLabel} has`}
				</Button>
				<Button type="ghost" size="sm" onclick={() => sync(false)} disabled={busy !== ''}>
					{busy === 'sync' ? 'Comparing...' : 'Compare'}
				</Button>
				<Button type="secondary" size="sm" onclick={() => sync(true)} disabled={busy !== ''}>
					{busy === 'import' ? 'Importing...' : 'Import missing mailboxes'}
				</Button>
			</div>
			{#if inventory}
				<div class="rounded-lg bg-base-200 p-3 text-sm space-y-1">
					<p class="text-text">{inventory.domains.length} domain{inventory.domains.length === 1 ? '' : 's'}, {inventory.mailboxes.length} mailbox{inventory.mailboxes.length === 1 ? '' : 'es'}</p>
					{#each inventory.domains as d (d.id)}
						<p class="text-text-muted">{d.name} <Badge size="sm">{d.status}</Badge></p>
					{/each}
				</div>
			{/if}
			{#if synced}
				<div class="rounded-lg bg-base-200 p-3 text-sm space-y-1">
					{#if !synced.only_at_service.length && !synced.only_in_pool.length && !synced.imported.length}
						<p class="text-text">The pool and {serviceLabel} match.</p>
					{/if}
					{#each synced.imported as r (r.email)}
						<p class={r.error ? 'text-error' : 'text-text'}>{r.email}: {r.error || 'imported'}</p>
					{/each}
					{#each synced.only_at_service as m (m.id)}
						<p class="text-text-muted">At {serviceLabel}, not in the pool: {m.email} ({m.status})</p>
					{/each}
					{#each synced.only_in_pool as m (m.id)}
						<p class="text-text-muted">In the pool, not at {serviceLabel}: {m.email}</p>
					{/each}
					{#each synced.pending as p (p)}
						<p class="text-text-muted">{p}</p>
					{/each}
				</div>
			{/if}
		</div>

		{#if mailboxes.length}
			<div class="space-y-1">
				<p class="form-label">Delete at {serviceLabel}</p>
				{#each mailboxes as mb (mb.id)}
					<div class="flex items-center justify-between gap-2 text-sm">
						<span class="text-text">{mb.email}</span>
						<Button
							type="ghost"
							size="xs"
							onclick={() => {
								deleting = mb;
								confirmDelete = true;
							}}
							disabled={busy !== '' || (mb.reason || '').startsWith('Deleted:')}
						>
							{busy === 'delete' && deleting?.id === mb.id ? 'Deleting...' : 'Delete'}
						</Button>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={confirmBuy}
	loading={busy === 'buy'}
	title={quote ? `Buy for ${price(quote)}?` : 'Buy?'}
	description={quote ? `${quote.summary} This spends money at ${serviceLabel}.` : ''}
	actionLabel={quote ? `Buy for ${price(quote)}` : 'Buy'}
	onAction={buy}
/>

<AlertDialog
	bind:open={confirmDelete}
	loading={busy === 'delete'}
	title={deleting ? `Delete ${deleting.email}?` : 'Delete?'}
	description={`${serviceLabel} deletes it (CheapInboxes cancels it at the end of its billing period); Maildoso and Infraforge delete its messages too. It stays in this pool, paused, with its threads.`}
	actionLabel="Delete"
	actionType="danger"
	onAction={remove}
/>
