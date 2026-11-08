<script lang="ts">
	import {
		updateMailboxConnection,
		removeMailboxConnection,
		moveOutboundDomainConnection,
		getMailboxServiceInventory,
		connectMailboxServiceDomain,
		quoteMailboxService,
		buyMailboxService,
		syncMailboxService,
		deleteMailboxServiceMailbox,
		type MailboxConnection,
		type MailboxConnectionDomain,
		type MailboxServiceInventory,
		type MailboxServiceDomainResponse,
		type MailboxServiceQuote,
		type MailboxServiceBuyResponse,
		type MailboxServiceSyncResponse,
		type OutboundMailboxHealth
	} from '$lib/api';
	import { Button, Alert, Input, Select, Textarea, Badge, AlertDialog } from '$lib/components/ui';
	import { formatDate } from '$lib/dates';
	import { errorMessage } from '../../outbound/shared';
	import { connectionTargets, limitText, limitValue, parsePeople, serviceName } from './connections';

	// One of a pool's connections: its service, key, domains and mailboxes,
	// and what can be done there. Buying is a quote, then a buy the person
	// confirms with the price in front of them; a quote buys only here.

	let {
		connection,
		connections,
		mailboxes,
		onchange
	}: {
		connection: MailboxConnection;
		connections: MailboxConnection[];
		mailboxes: OutboundMailboxHealth[];
		onchange: () => void | Promise<void>;
	} = $props();

	const KINDS = [
		{ value: 'mailboxes', label: 'Mailboxes on a domain' },
		{ value: 'domain', label: 'A new domain' },
		{ value: 'connect', label: 'Connect a domain I own (Infraforge)' }
	];

	let open = $state(false);
	let busy = $state('');
	let error = $state('');
	let notice = $state('');

	let editName = $state('');
	let editKey = $state('');
	let editLimit = $state<number | undefined>(undefined);
	let editWarmupStart = $state<number | undefined>(undefined);
	let editWarmupStep = $state<number | undefined>(undefined);
	let confirmRemove = $state(false);
	let moveTo = $state<Record<string, string>>({});

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
	let deleting = $state<OutboundMailboxHealth | null>(null);
	let confirmDelete = $state(false);

	let manual = $derived(connection.service === 'manual');
	let label = $derived(serviceName(connection.service));
	let active = $derived(!manual && connection.key_set);
	let targets = $derived(connectionTargets(connections, connection));

	$effect(() => {
		editName = connection.name;
		editLimit = connection.daily_limit;
		editWarmupStart = connection.warmup_start;
		editWarmupStep = connection.warmup_step;
	});

	async function run(name: string, fn: () => Promise<void>) {
		busy = name;
		error = '';
		notice = '';
		try {
			await fn();
		} catch (err) {
			error = errorMessage(err, manual ? 'That was refused' : `${label} refused`);
		} finally {
			busy = '';
		}
	}

	function save() {
		return run('save', async () => {
			await updateMailboxConnection(
				{},
				{
					name: editName.trim() || undefined,
					api_key: editKey.trim() || undefined,
					daily_limit: limitValue(editLimit, 1),
					warmup_start: limitValue(editWarmupStart, 0),
					warmup_step: limitValue(editWarmupStep, 0)
				},
				connection.id
			);
			editKey = '';
			notice = 'Saved.';
			await onchange();
		});
	}

	function remove() {
		return run('remove', async () => {
			await removeMailboxConnection({}, connection.id);
			confirmRemove = false;
			await onchange();
		});
	}

	function move(d: MailboxConnectionDomain) {
		const target = moveTo[d.id];
		if (!target) return;
		return run('move-' + d.id, async () => {
			const to = await moveOutboundDomainConnection({}, { connection_id: target }, d.id);
			moveTo = { ...moveTo, [d.id]: '' };
			notice = `${d.domain} and its mailboxes moved to ${to.name}.`;
			await onchange();
		});
	}

	function connect() {
		return run('connect', async () => {
			connected = await connectMailboxServiceDomain({}, { domain: connectDomain.trim() }, connection.id);
			connectDomain = '';
			await onchange();
		});
	}

	function getQuote() {
		return run('quote', async () => {
			bought = null;
			quote = await quoteMailboxService(
				{},
				{ kind, domain: orderDomain.trim(), people: kind === 'connect' ? undefined : parsePeople(people) },
				connection.id
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
			bought = await buyMailboxService({}, { quote_id: q.id, confirm: true }, connection.id);
			quote = null;
			people = '';
			await onchange();
		});
	}

	function showInventory() {
		return run('inventory', async () => {
			inventory = await getMailboxServiceInventory({}, connection.id);
		});
	}

	function sync(importNew: boolean) {
		return run(importNew ? 'import' : 'sync', async () => {
			synced = await syncMailboxService({}, { import: importNew }, connection.id);
			if (importNew) await onchange();
		});
	}

	function deleteMailbox() {
		const mb = deleting;
		if (!mb) return;
		return run('delete', async () => {
			await deleteMailboxServiceMailbox({}, connection.id, mb.id);
			notice = `${mb.email} deleted at ${label}; it stays in the pool, paused.`;
			deleting = null;
			await onchange();
		});
	}
</script>

<div class="rounded-lg border border-border p-3 space-y-3">
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
		<div class="min-w-0">
			<div class="flex flex-wrap items-center gap-2">
				<span class="font-medium text-text truncate">{connection.name}</span>
				<Badge size="sm" variant="ghost">{label}</Badge>
				{#if !manual}
					<Badge size="sm" variant={connection.key_set ? 'success' : 'warning'}>
						{connection.key_set ? 'Key set' : 'No key'}
					</Badge>
				{/if}
			</div>
			<p class="text-xs text-text-muted mt-1">
				{connection.domains.length} domain{connection.domains.length === 1 ? '' : 's'}{connection.domains.length
					? ` (${connection.domains.map((d) => d.domain).join(', ')})`
					: ''}, {connection.mailboxes} mailbox{connection.mailboxes === 1 ? '' : 'es'}. {limitText(connection)}.
			</p>
		</div>
		<Button type="ghost" size="sm" onclick={() => (open = !open)}>{open ? 'Close' : 'Manage'}</Button>
	</div>

	{#if error}
		<Alert type="error"><p>{error}</p></Alert>
	{/if}
	{#if notice}
		<Alert type="success" dismissible onclose={() => (notice = '')}><p>{notice}</p></Alert>
	{/if}

	{#if open}
		<div class="space-y-2">
			<p class="form-label">Connection</p>
			<div class="grid grid-cols-1 sm:grid-cols-3 gap-3 items-end">
				<div>
					<label for="mc-edit-name-{connection.id}" class="form-label">Name</label>
					<Input id="mc-edit-name-{connection.id}" bind:value={editName} />
				</div>
				{#if !manual}
					<div>
						<label for="mc-edit-key-{connection.id}" class="form-label">API key</label>
						<Input
							id="mc-edit-key-{connection.id}"
							type="password"
							bind:value={editKey}
							placeholder={connection.key_set ? 'Set; enter a new one to replace it' : 'Paste the API key'}
						/>
					</div>
				{/if}
				<div>
					<label for="mc-edit-limit-{connection.id}" class="form-label">Daily limit per mailbox</label>
					<Input
						id="mc-edit-limit-{connection.id}"
						type="number"
						min={1}
						bind:value={editLimit}
						placeholder="The pool's"
					/>
				</div>
				<div>
					<label for="mc-edit-wstart-{connection.id}" class="form-label">Warm-up first day</label>
					<Input
						id="mc-edit-wstart-{connection.id}"
						type="number"
						min={0}
						bind:value={editWarmupStart}
						placeholder="The pool's"
					/>
				</div>
				<div>
					<label for="mc-edit-wstep-{connection.id}" class="form-label">Warm-up added each day</label>
					<Input
						id="mc-edit-wstep-{connection.id}"
						type="number"
						min={0}
						bind:value={editWarmupStep}
						placeholder="The pool's"
					/>
				</div>
			</div>
			<p class="form-hint">
				Its mailboxes start at the first day's number and add the daily step until they reach the daily limit. A
				mailbox's own daily limit still wins; leave a field empty for the pool's, and a first day of 0 turns warm-up
				off. The pool's spacing and health pauses still apply.
			</p>
			<div class="flex flex-wrap gap-2">
				<Button type="secondary" size="sm" onclick={save} disabled={busy !== ''}>
					{busy === 'save' ? (editKey.trim() ? 'Checking the key...' : 'Saving...') : 'Save'}
				</Button>
				<Button
					type="ghost"
					size="sm"
					onclick={() => (confirmRemove = true)}
					disabled={busy !== '' || connection.domains.length > 0}
				>
					Remove
				</Button>
			</div>
			{#if connection.domains.length > 0}
				<p class="form-hint">To remove it, first move its domains to another connection of this pool.</p>
			{/if}
		</div>

		{#if connection.domains.length}
			<div class="space-y-2">
				<p class="form-label">Domains</p>
				{#each connection.domains as d (d.id)}
					<div class="flex flex-col sm:flex-row sm:items-center gap-2 text-sm">
						<span class="text-text sm:w-48 truncate">{d.domain}</span>
						{#if targets.length}
							<div class="flex items-center gap-2">
								<label for="mc-move-{d.id}" class="sr-only">Move {d.domain} to</label>
								<Select
									id="mc-move-{d.id}"
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
									disabled={busy !== '' || !moveTo[d.id]}
								>
									{busy === 'move-' + d.id ? 'Moving...' : 'Move'}
								</Button>
							</div>
						{/if}
					</div>
				{/each}
				<p class="form-hint">A domain moves with its mailboxes to another connection of this pool.</p>
			</div>
		{/if}

		{#if manual}
			<p class="text-sm text-text-muted">
				Add this connection's domains and mailboxes by hand in Outbound, Setup. A domain added there joins this pool's
				first connection added by hand; move it here if it belongs here.
			</p>
		{:else if active}
			<div class="space-y-2">
				<p class="form-label">Connect a domain you own</p>
				<p class="form-hint">
					{label} runs the domain's DNS: you set the nameservers it gives at the domain's registrar.
					{#if connection.service === 'infraforge'}Infraforge bills for this, so use a quote below.{/if}
				</p>
				{#if connection.service !== 'infraforge'}
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
				<p class="form-label">Buy at {label}</p>
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
				<p class="form-label">Sync with {label}</p>
				<div class="flex flex-wrap gap-2">
					<Button type="ghost" size="sm" onclick={showInventory} disabled={busy !== ''}>
						{busy === 'inventory' ? 'Loading...' : `What ${label} has`}
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
						<p class="text-text">
							{inventory.domains.length} domain{inventory.domains.length === 1 ? '' : 's'}, {inventory.mailboxes.length} mailbox{inventory
								.mailboxes.length === 1
								? ''
								: 'es'}
						</p>
						{#each inventory.domains as d (d.id)}
							<p class="text-text-muted">{d.name} <Badge size="sm">{d.status}</Badge></p>
						{/each}
					</div>
				{/if}
				{#if synced}
					<div class="rounded-lg bg-base-200 p-3 text-sm space-y-1">
						{#if !synced.only_at_service.length && !synced.only_in_pool.length && !synced.imported.length}
							<p class="text-text">This connection and {label} match.</p>
						{/if}
						{#each synced.imported as r (r.email)}
							<p class={r.error ? 'text-error' : 'text-text'}>{r.email}: {r.error || 'imported'}</p>
						{/each}
						{#each synced.only_at_service as m (m.id)}
							<p class="text-text-muted">At {label}, not in the pool: {m.email} ({m.status})</p>
						{/each}
						{#each synced.only_in_pool as m (m.id)}
							<p class="text-text-muted">In the pool, not at {label}: {m.email}</p>
						{/each}
						{#each synced.pending as p (p)}
							<p class="text-text-muted">{p}</p>
						{/each}
					</div>
				{/if}
			</div>

			{#if mailboxes.length}
				<div class="space-y-1">
					<p class="form-label">Delete at {label}</p>
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
		{:else}
			<p class="text-sm text-text-muted">Enter its {label} API key above to connect or buy domains and mailboxes.</p>
		{/if}
	{/if}
</div>

<AlertDialog
	bind:open={confirmRemove}
	loading={busy === 'remove'}
	title={`Remove ${connection.name}?`}
	description={manual
		? 'The connection goes; nothing else changes.'
		: `Outlet forgets the connection and its key. Nothing changes at ${label}.`}
	actionLabel="Remove"
	actionType="danger"
	onAction={remove}
/>

<AlertDialog
	bind:open={confirmBuy}
	loading={busy === 'buy'}
	title={quote ? `Buy for ${price(quote)}?` : 'Buy?'}
	description={quote ? `${quote.summary} This spends money at ${label} (${connection.name}).` : ''}
	actionLabel={quote ? `Buy for ${price(quote)}` : 'Buy'}
	onAction={buy}
/>

<AlertDialog
	bind:open={confirmDelete}
	loading={busy === 'delete'}
	title={deleting ? `Delete ${deleting.email}?` : 'Delete?'}
	description={`${label} deletes it (CheapInboxes cancels it at the end of its billing period); Maildoso and Infraforge delete its messages too. It stays in the pool, paused, with its threads.`}
	actionLabel="Delete"
	actionType="danger"
	onAction={deleteMailbox}
/>
