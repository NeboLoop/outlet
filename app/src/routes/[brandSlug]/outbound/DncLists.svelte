<script lang="ts">
	import {
		getOutboundDoNotContact,
		addOutboundDoNotContact,
		deleteOutboundDoNotContactList,
		type OutboundDNCList
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Input, Textarea } from '$lib/components/ui';
	import { errorMessage } from './shared';

	// The brand's do-not-contact lists, and adding addresses or domains to
	// one, typed or read from a CSV. onchange hears each list changed.

	let { onchange }: { onchange?: () => void } = $props();

	const MAX_ENTRIES = 10000;

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let lists = $state<OutboundDNCList[]>([]);
	let busy = $state('');

	let listName = $state('Customers');
	let entriesText = $state('');
	let entries = $derived(entriesText.split(/[\s,;]+/).map((e) => e.trim()).filter(Boolean));

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			lists = (await getOutboundDoNotContact()).lists || [];
		} catch (err) {
			error = errorMessage(err, 'Could not load do not contact');
		} finally {
			loading = false;
		}
	}

	async function act(key: string, call: () => Promise<string>) {
		busy = key;
		error = '';
		notice = '';
		try {
			notice = await call();
			await load();
			onchange?.();
		} catch (err) {
			error = errorMessage(err, 'Something went wrong');
		} finally {
			busy = '';
		}
	}

	function upload() {
		act('upload', async () => {
			const res = await addOutboundDoNotContact({ list: listName.trim(), entries: entries.slice(0, MAX_ENTRIES) });
			entriesText = '';
			const refused = Object.keys(res.refused || {}).length;
			return `Added ${res.added} to ${listName.trim()}${res.duplicates ? `, ${res.duplicates} already there` : ''}${refused ? `, ${refused} not an address or domain` : ''}.`;
		});
	}

	async function readFile(e: Event) {
		const file = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		// A CSV's cells: every address or domain in it, whatever the columns.
		entriesText = (await file.text()).replace(/"/g, '');
	}

	function removeList(name: string) {
		act(`delete:${name}`, async () => {
			await deleteOutboundDoNotContactList({ list: name });
			return `${name} deleted.`;
		});
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

{#if loading}
	<div class="flex justify-center py-6">
		<LoadingSpinner />
	</div>
{:else}
	{#if lists.length === 0}
		<p class="text-sm text-text-muted mb-4">No lists yet.</p>
	{:else}
		<div class="data-table overflow-x-auto mb-4">
			<table class="w-full text-sm">
				<thead>
					<tr>
						<th class="text-left">List</th>
						<th class="text-right">Addresses</th>
						<th class="text-right">Domains</th>
						<th></th>
					</tr>
				</thead>
				<tbody>
					{#each lists as l (l.name)}
						<tr>
							<td>{l.name}</td>
							<td class="text-right">{l.emails}</td>
							<td class="text-right">{l.domains}</td>
							<td class="text-right">
								<Button type="ghost" size="xs" onclick={() => removeList(l.name)} disabled={busy !== ''}>Delete</Button>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
	<div class="rounded-lg border border-border p-4 space-y-4">
		<div>
			<label for="dnc-list" class="form-label">List</label>
			<Input id="dnc-list" bind:value={listName} placeholder="Customers, Open deals" />
		</div>
		<div>
			<label for="dnc-entries" class="form-label">Addresses or domains</label>
			<Textarea id="dnc-entries" bind:value={entriesText} rows={5} placeholder="ann@client.com&#10;client2.com" />
			<p class="form-hint">
				One per line, or separated by commas. A domain skips everyone at it; a website counts as its domain.
				{entries.length} entries.
			</p>
		</div>
		<div>
			<label for="dnc-file" class="form-label">Or read a CSV</label>
			<input id="dnc-file" type="file" accept=".csv,.txt" class="file-input file-input-bordered file-input-sm w-full" onchange={readFile} />
		</div>
		<div class="flex justify-end">
			<Button type="primary" onclick={upload} disabled={busy !== '' || !listName.trim() || entries.length === 0}>
				{busy === 'upload' ? 'Adding...' : 'Add to list'}
			</Button>
		</div>
	</div>
{/if}
