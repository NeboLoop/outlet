<script lang="ts">
	import {
		getOutboundDoNotContact,
		addOutboundDoNotContact,
		deleteOutboundDoNotContactList,
		checkOutboundDoNotContact,
		updateOutboundDoNotContactSettings,
		type OutboundDNC,
		type OutboundDNCCheck
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Input, Textarea, Toggle } from '$lib/components/ui';
	import { errorMessage } from '../shared';

	// People outreach never writes to. Before anything is queued, and again
	// before each follow-up goes, everyone is checked against suppression,
	// the brand's own domains, its customers, these lists and the outside
	// check. The first that matches is the reason, and every skip says it.

	const MAX_ENTRIES = 10000;

	let loading = $state(true);
	let error = $state('');
	let notice = $state('');
	let dnc = $state<OutboundDNC | null>(null);
	let busy = $state('');

	let listName = $state('Customers');
	let entriesText = $state('');
	let entries = $derived(entriesText.split(/[\s,;]+/).map((e) => e.trim()).filter(Boolean));

	let checkEmail = $state('');
	let checked = $state<OutboundDNCCheck | null>(null);

	let customerDomains = $state(false);
	let checkURL = $state('');
	let checkSecret = $state('');

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			dnc = await getOutboundDoNotContact();
			customerDomains = dnc.settings.customer_domains;
			checkURL = dnc.settings.check_url;
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

	async function check() {
		busy = 'check';
		error = '';
		try {
			checked = await checkOutboundDoNotContact({ email: checkEmail.trim() });
		} catch (err) {
			error = errorMessage(err, 'Could not check the address');
		} finally {
			busy = '';
		}
	}

	function saveSettings() {
		act('settings', async () => {
			const url = checkURL.trim();
			await updateOutboundDoNotContactSettings({
				customer_domains: customerDomains,
				check_url: url,
				check_secret: checkSecret || undefined
			});
			checkSecret = '';
			return 'Settings saved.';
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
	<div class="flex justify-center py-12">
		<LoadingSpinner size="large" />
	</div>
{:else if dnc}
	<div class="space-y-8">
		<section>
			<h2 class="text-lg font-semibold text-text mb-1">Who is never contacted</h2>
			<p class="form-hint">
				Before any outreach is queued, and again before each follow-up goes, everyone is checked against: the suppression
				list and past bounces and complaints; your own domains; your customers (anyone on one of your lists, or sent a
				transactional email); the lists below; and the outside check, if set. Anyone skipped says why.
			</p>
		</section>

		<section>
			<h2 class="text-lg font-semibold text-text mb-3">Check an address</h2>
			<div class="flex flex-col sm:flex-row gap-2">
				<div class="flex-1">
					<Input bind:value={checkEmail} placeholder="someone@company.com" />
				</div>
				<Button type="secondary" onclick={check} disabled={busy !== '' || !checkEmail.includes('@')}>Check</Button>
			</div>
			{#if checked}
				<p class="text-sm mt-2" class:text-warning={checked.excluded} class:text-success={!checked.excluded}>
					{checked.excluded ? `Not contacted (${checked.source}): ${checked.reason}` : 'Nothing stops outreach to this address.'}
				</p>
			{/if}
		</section>

		<section>
			<h2 class="text-lg font-semibold text-text mb-3">Lists</h2>
			{#if dnc.lists.length === 0}
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
							{#each dnc.lists as l (l.name)}
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
		</section>

		<section>
			<h2 class="text-lg font-semibold text-text mb-3">Options</h2>
			<div class="space-y-4">
				<Toggle
					bind:checked={customerDomains}
					label="Skip everyone at a customer's company"
					description="Anyone whose address is at the same domain as a customer's. Free mail domains like gmail.com never count."
				/>
				<div>
					<label for="dnc-url" class="form-label">Outside check</label>
					<Input id="dnc-url" bind:value={checkURL} placeholder="https://crm.example.com/outlet/do-not-contact" />
					<p class="form-hint">
						Optional. Outlet POSTs <code>{'{email, domain, brand, timestamp}'}</code> with
						<code>X-Outlet-Signature</code> (HMAC-SHA256 of the body with the secret) and reads
						<code>{'{"do_not_contact": true or false, "reason": "..."}'}</code>. While it doesn't answer, nobody new is
						enrolled and emails already in a campaign wait: nothing goes until it answers. Leave it empty to turn it off.
					</p>
				</div>
				<div>
					<label for="dnc-secret" class="form-label">Secret</label>
					<Input id="dnc-secret" type="password" bind:value={checkSecret} placeholder={dnc.settings.check_secret_set ? 'Set (leave empty to keep it)' : 'The shared secret'} />
				</div>
				<p class="form-hint">Changing these needs the brand's admin.</p>
				<div class="flex justify-end">
					<Button type="primary" onclick={saveSettings} disabled={busy !== ''}>
						{busy === 'settings' ? 'Saving...' : 'Save options'}
					</Button>
				</div>
			</div>
		</section>
	</div>
{/if}
