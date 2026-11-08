<script lang="ts">
	import {
		listMailboxConnections,
		addMailboxConnection,
		getOutboundOverview,
		type SendingPathway,
		type MailboxConnection as Connection,
		type OutboundMailboxHealth
	} from '$lib/api';
	import { Button, Alert, Input, Select } from '$lib/components/ui';
	import { Plus } from '@lucide/svelte';
	import { errorMessage } from '../../outbound/shared';
	import { SERVICES } from './connections';
	import MailboxConnection from './MailboxConnection.svelte';

	// A mailbox pool's connections: the mailbox services (and mailboxes added
	// by hand) its domains and mailboxes come from, side by side. The pool
	// sends from all of them under its limits.

	let { pool, onchange }: { pool: SendingPathway; onchange: () => void | Promise<void> } = $props();

	let connections = $state<Connection[]>([]);
	let mailboxes = $state<OutboundMailboxHealth[]>([]);
	let error = $state('');
	let notice = $state('');

	let adding = $state(false);
	let saving = $state(false);
	let service = $state('maildoso');
	let name = $state('');
	let apiKey = $state('');

	$effect(() => {
		if (pool.id) load();
	});

	async function load() {
		try {
			connections = (await listMailboxConnections({ pool_id: pool.id })).connections || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load the connections');
		}
		try {
			const overview = await getOutboundOverview();
			mailboxes = (overview.mailboxes || []).filter((m) => m.pool_id === pool.id);
		} catch {
			mailboxes = [];
		}
	}

	async function changed() {
		await load();
		await onchange();
	}

	async function add() {
		saving = true;
		error = '';
		notice = '';
		try {
			const c = await addMailboxConnection({
				pool_id: pool.id,
				service,
				name: name.trim() || undefined,
				api_key: service === 'manual' ? undefined : apiKey.trim()
			});
			notice =
				c.service === 'manual'
					? `${c.name} added. Add its domains and mailboxes in Outbound, Setup.`
					: `${c.name} added; its key works. Connect or buy a domain on it below.`;
			adding = false;
			name = '';
			apiKey = '';
			await changed();
		} catch (err) {
			error = errorMessage(err, 'Could not add the connection');
		} finally {
			saving = false;
		}
	}
</script>

<div class="mt-4 border-t border-border pt-4 space-y-4">
	<div class="flex items-start justify-between gap-3">
		<div>
			<h4 class="font-medium text-text">Connections</h4>
			<p class="text-sm text-text-muted">
				Where this pool's domains and mailboxes come from. Run several mailbox services side by side; the pool sends
				from all their mailboxes.
			</p>
		</div>
		{#if !adding}
			<Button type="secondary" size="sm" onclick={() => (adding = true)}>
				<Plus class="w-4 h-4" />
				Add connection
			</Button>
		{/if}
	</div>

	{#if error}
		<Alert type="error"><p>{error}</p></Alert>
	{/if}
	{#if notice}
		<Alert type="success" dismissible onclose={() => (notice = '')}><p>{notice}</p></Alert>
	{/if}

	{#if adding}
		<div class="rounded-lg border border-border p-3 space-y-3">
			<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
				<div>
					<label for="mc-service-{pool.id}" class="form-label">Service</label>
					<Select id="mc-service-{pool.id}" bind:value={service} options={SERVICES} />
				</div>
				<div>
					<label for="mc-name-{pool.id}" class="form-label">Name (optional)</label>
					<Input id="mc-name-{pool.id}" bind:value={name} placeholder="Maildoso main" />
				</div>
				{#if service !== 'manual'}
					<div>
						<label for="mc-key-{pool.id}" class="form-label">API key</label>
						<Input id="mc-key-{pool.id}" type="password" bind:value={apiKey} placeholder="Paste the API key" />
					</div>
				{/if}
			</div>
			{#if service === 'maildoso'}
				<p class="form-hint">
					Maildoso's terms cap cold email at 15 a day per mailbox, to business addresses only, so its mailboxes default
					to 15 a day. A mailbox's own limit still wins.
				</p>
			{:else if service !== 'manual'}
				<p class="form-hint">The key is checked with the service before it is saved, and never shown again.</p>
			{/if}
			<div class="flex gap-2">
				<Button type="primary" size="sm" onclick={add} disabled={saving || (service !== 'manual' && !apiKey.trim())}>
					{saving ? 'Checking the key...' : 'Add'}
				</Button>
				<Button type="ghost" size="sm" onclick={() => (adding = false)} disabled={saving}>Cancel</Button>
			</div>
		</div>
	{/if}

	{#if connections.length === 0 && !adding}
		<p class="text-sm text-text-muted">No connections yet. Add one to connect or buy domains and mailboxes.</p>
	{/if}

	{#each connections as c (c.id)}
		<MailboxConnection
			connection={c}
			{connections}
			mailboxes={mailboxes.filter((m) => m.connection_id === c.id)}
			onchange={changed}
		/>
	{/each}
</div>
