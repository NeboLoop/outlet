<script lang="ts">
	import { page } from '$app/stores';
	import {
		getTransactionalEmail,
		updateTransactionalEmail,
		getTransactionalEmailStats,
		sendTransactionalTest,
		getOrganizationBySlug,
		listEmailDesigns,
		type EmailDesignInfo,
		type TransactionalEmailInfo,
		type TransactionalStatsResponse,
		type OrgInfo
	} from '$lib/api';
	import { Button, Alert, LoadingSpinner, Input, Textarea, HtmlEditor, Card, Badge, Select } from '$lib/components/ui';
	import { ArrowLeft, Send, Copy, Check } from '@lucide/svelte';

	let id = $derived($page.params.id ?? '');
	let basePath = $derived(`/${$page.params.brandSlug}`);

	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let notice = $state('');

	let saved = $state<TransactionalEmailInfo | null>(null);
	let org = $state<OrgInfo | null>(null);
	let stats = $state<TransactionalStatsResponse | null>(null);

	let name = $state('');
	let slug = $state('');
	let subject = $state('');
	let htmlBody = $state('');
	let plainText = $state('');
	let fromName = $state('');
	let fromEmail = $state('');
	let replyTo = $state('');
	let isActive = $state(true);
	let designId = $state('');
	let designs = $state<EmailDesignInfo[]>([]);

	let samples = $state<Record<string, string>>({});
	let testTo = $state('');
	let testing = $state(false);
	let copied = $state(false);

	// Every {{variable}} used in the subject and bodies, in order of first use
	let variables = $derived.by(() => {
		const found = new Set<string>();
		for (const m of `${subject}\n${htmlBody}\n${plainText}`.matchAll(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g)) {
			found.add(m[1]);
		}
		return [...found];
	});

	let dirty = $derived(
		!!saved &&
			(name !== saved.name ||
				slug !== saved.slug ||
				subject !== saved.subject ||
				htmlBody !== saved.html_body ||
				plainText !== (saved.plain_text ?? '') ||
				fromName !== (saved.from_name ?? '') ||
				fromEmail !== (saved.from_email ?? '') ||
				replyTo !== (saved.reply_to ?? '') ||
				isActive !== saved.is_active ||
				designId !== (saved.design_id ?? ''))
	);

	function fill(text: string): string {
		return text.replace(/\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}/g, (match, key) => samples[key] || match);
	}

	let previewSubject = $derived(fill(subject));
	// The send places the body into the design at {{content}}; so does the preview.
	let design = $derived(designs.find((d) => d.id === designId));
	let previewHTML = $derived(fill(design ? design.html_body.replaceAll('{{content}}', htmlBody) : htmlBody));

	let snippet = $derived.by(() => {
		const vars = Object.fromEntries(variables.map((v) => [v, samples[v] || `<${v}>`]));
		const body = JSON.stringify({ to: 'customer@example.com', template_slug: slug, variables: vars }, null, 2);
		const origin = typeof window === 'undefined' ? '' : window.location.origin;
		return `curl -X POST ${origin}/sdk/v1/emails \\\n  -H "X-API-Key: $OUTLET_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '${body}'`;
	});

	$effect(() => {
		load(id);
	});

	async function load(templateId: string) {
		loading = true;
		error = '';
		try {
			const [t, o, d] = await Promise.all([
				getTransactionalEmail({}, templateId),
				getOrganizationBySlug({}, $page.params.brandSlug as string),
				listEmailDesigns({})
			]);
			org = o;
			designs = (d.designs || []).filter((x) => x.is_active || x.id === t.design_id);
			setFrom(t);
			getTransactionalEmailStats({}, templateId)
				.then((s) => (stats = s))
				.catch(() => (stats = null));
		} catch (err: any) {
			error = err.message || 'Failed to load template';
		} finally {
			loading = false;
		}
	}

	function setFrom(t: TransactionalEmailInfo) {
		saved = t;
		name = t.name;
		slug = t.slug;
		subject = t.subject;
		htmlBody = t.html_body;
		plainText = t.plain_text ?? '';
		fromName = t.from_name ?? '';
		fromEmail = t.from_email ?? '';
		replyTo = t.reply_to ?? '';
		isActive = t.is_active;
		designId = t.design_id ?? '';
	}

	async function save() {
		saving = true;
		error = '';
		notice = '';
		try {
			const updated = await updateTransactionalEmail(
				{},
				{
					name,
					slug,
					subject,
					html_body: htmlBody,
					plain_text: plainText,
					from_name: fromName,
					from_email: fromEmail,
					reply_to: replyTo,
					is_active: isActive,
					design_id: designId
				},
				id
			);
			setFrom(updated);
			notice = 'Saved';
		} catch (err: any) {
			error = err.message || 'Failed to save';
		} finally {
			saving = false;
		}
	}

	// Sends the saved template through the same send as the API your apps
	// use, so a test proves the slug, sender and suppression rules all work
	async function sendTest() {
		if (!testTo) return;
		testing = true;
		error = '';
		notice = '';
		try {
			await sendTransactionalTest({ to: testTo, template_slug: slug, variables: samples });
			notice = `Test sent to ${testTo}`;
		} catch (err: any) {
			error = err.message || 'Test send failed';
		} finally {
			testing = false;
		}
	}

	async function copySnippet() {
		await navigator.clipboard.writeText(snippet);
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}
</script>

<svelte:head>
	<title>{name || 'Template'} | Transactional | Outlet</title>
</svelte:head>

<div class="p-4 sm:p-6 max-w-6xl mx-auto">
	<a href="{basePath}/transactional" class="inline-flex items-center gap-1 text-sm text-text-muted hover:text-text mb-4">
		<ArrowLeft class="h-4 w-4" /> Transactional
	</a>

	{#if loading}
		<div class="flex justify-center py-12"><LoadingSpinner size="large" /></div>
	{:else if !saved}
		<Alert type="error" title="Error"><p>{error || 'Template not found'}</p></Alert>
	{:else}
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
			<div class="min-w-0">
				<h1 class="text-2xl font-semibold text-text truncate">{name || 'Untitled'}</h1>
				<p class="mt-1 text-sm text-text-muted">
					Sent with <code>template_slug: "{slug}"</code>
					{#if !isActive}<Badge variant="warning" size="sm" class="ml-2">Inactive: sends are refused</Badge>{/if}
				</p>
			</div>
			<Button type="primary" onclick={save} disabled={!dirty || saving || !name || !slug || !subject}>
				{saving ? 'Saving...' : dirty ? 'Save changes' : 'Saved'}
			</Button>
		</div>

		{#if error}
			<Alert type="error" title="Error" class="mb-4"><p>{error}</p></Alert>
		{:else if notice}
			<Alert type="success" title="Done" class="mb-4"><p>{notice}</p></Alert>
		{/if}

		{#if stats}
			<div class="grid grid-cols-3 sm:grid-cols-6 gap-3 mb-6">
				{#each [['Sent', stats.sent], ['Delivered', stats.delivered], ['Opened', stats.opened], ['Clicked', stats.clicked], ['Bounced', stats.bounced], ['Failed', stats.failed]] as [label, value]}
					<div class="rounded-lg border border-border p-3">
						<div class="text-xs text-text-muted">{label}</div>
						<div class="text-lg font-semibold text-text">{value}</div>
					</div>
				{/each}
			</div>
		{/if}

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
			<!-- Editor -->
			<div class="space-y-4 min-w-0">
				<Card>
					<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
						<div>
							<label for="name" class="form-label">Name</label>
							<Input id="name" bind:value={name} />
						</div>
						<div>
							<label for="slug" class="form-label">Slug</label>
							<Input id="slug" bind:value={slug} />
						</div>
						<div class="sm:col-span-2">
							<label for="subject" class="form-label">Subject</label>
							<Input id="subject" bind:value={subject} />
						</div>
						<div>
							<label for="from-name" class="form-label">From name</label>
							<Input id="from-name" bind:value={fromName} placeholder={org?.from_name || 'Brand default'} />
						</div>
						<div>
							<label for="from-email" class="form-label">From email</label>
							<Input id="from-email" type="email" bind:value={fromEmail} placeholder={org?.from_email || 'Brand default'} />
						</div>
						<div>
							<label for="reply-to" class="form-label">Reply-To</label>
							<Input id="reply-to" type="email" bind:value={replyTo} placeholder={org?.reply_to || 'Brand default'} />
						</div>
						<div class="flex items-end">
							<label class="inline-flex items-center gap-2 text-sm text-text cursor-pointer pb-2">
								<input type="checkbox" bind:checked={isActive} class="h-4 w-4" />
								Active
							</label>
						</div>
					</div>
					<p class="mt-3 text-xs text-text-muted">Empty sender fields use the brand's settings.</p>
					<div class="mt-4">
						<label for="design" class="form-label">Design</label>
						<Select
							id="design"
							bind:value={designId}
							options={[{ value: '', label: 'None: send the body as it is' }, ...designs.map((d) => ({ value: d.id, label: d.name }))]}
						/>
						<p class="mt-1 text-xs text-text-muted">The body is placed into the design when the email is sent.</p>
					</div>
				</Card>

				<Card title="HTML body">
					<HtmlEditor bind:value={htmlBody} minHeight="280px" initialMode="html" />
				</Card>

				<Card title="Plain-text version">
					<Textarea bind:value={plainText} rows={6} placeholder="Shown by mail apps that don't display HTML. Recommended: it helps delivery." />
				</Card>
			</div>

			<!-- Preview, test, snippet -->
			<div class="space-y-4 min-w-0">
				{#if variables.length > 0}
					<Card title="Sample values">
						<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
							{#each variables as v (v)}
								<div>
									<label for="var-{v}" class="form-label"><code>{`{{${v}}}`}</code></label>
									<Input id="var-{v}" bind:value={samples[v]} placeholder={v} />
								</div>
							{/each}
						</div>
					</Card>
				{/if}

				<Card title="Preview">
					<div class="text-sm mb-2"><span class="text-text-muted">Subject:</span> <span class="text-text font-medium">{previewSubject}</span></div>
					<iframe
						title="Email preview"
						sandbox=""
						srcdoc={previewHTML}
						class="w-full h-96 rounded border border-border bg-white"
					></iframe>
				</Card>

				<Card title="Send a test">
					<div class="flex flex-col sm:flex-row gap-2">
						<Input type="email" bind:value={testTo} placeholder="you@example.com" class="flex-1" />
						<Button type="secondary" onclick={sendTest} disabled={!testTo || testing || dirty || !isActive}>
							<Send class="mr-2 h-4 w-4" />
							{testing ? 'Sending...' : 'Send test'}
						</Button>
					</div>
					<p class="mt-2 text-xs text-text-muted">
						{#if dirty}Save first: the test sends the saved version.{:else}Uses the sample values above and goes through the same API as your apps.{/if}
					</p>
				</Card>

				<Card title="Send it from your app">
					<div class="relative">
						<pre class="text-xs bg-bg-secondary rounded p-3 overflow-x-auto"><code>{snippet}</code></pre>
						<button type="button" onclick={copySnippet} class="absolute top-2 right-2 p-1.5 rounded hover:bg-bg text-text-muted" aria-label="Copy">
							{#if copied}<Check class="h-4 w-4" />{:else}<Copy class="h-4 w-4" />{/if}
						</button>
					</div>
					<p class="mt-2 text-xs text-text-muted">
						Your API key is under <a class="underline" href="{basePath}/settings">brand settings</a>. Over SMTP, use the brand slug as the username and the API key as the password.
					</p>
				</Card>
			</div>
		</div>
	{/if}
</div>
