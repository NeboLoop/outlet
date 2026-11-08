<script lang="ts">
	import { formatDate } from '$lib/dates';
	import { Card, Button, Input, Alert, LoadingSpinner, Badge, Modal, Tabs, CodeBlock, AlertDialog } from '$lib/components/ui';
	import { Copy, Plus, Trash2, Key, CheckCircle, ExternalLink, Code, Package } from '@lucide/svelte';
	import { listMCPAPIKeys, createMCPAPIKey, revokeMCPAPIKey, type MCPAPIKeyInfo } from '$lib/api';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';

	// Dynamic endpoint based on install domain
	// In dev mode, frontend runs on 5173 but API is on 8888
	// In production, both are on the same origin
	let installDomain = $state(browser ? window.location.origin.replace(':5173', ':8888') : 'https://your-domain.com');
	let apiBaseUrl = $derived(`${installDomain}/sdk/v1`);

	let loading = $state(true);
	let creating = $state(false);
	let error = $state('');
	let success = $state('');
	let copiedText = $state('');

	let apiKeys = $state<MCPAPIKeyInfo[]>([]);
	let showCreateModal = $state(false);
	let newKeyName = $state('');
	let newKeyValue = $state('');
	let showRevokeConfirm = $state(false);
	let revokeKeyId = $state('');
	let revoking = $state(false);

	// Code example tabs
	const codeTabs = [
		{ id: 'curl', label: 'cURL' },
		{ id: 'nodejs', label: 'Node.js' },
		{ id: 'python', label: 'Python' },
		{ id: 'go', label: 'Go' },
		{ id: 'php', label: 'PHP' },
		{ id: 'smtp', label: 'SMTP' }
	];
	let activeCodeTab = $state('curl');

	// Code examples. Every brand has its own API key (brand settings), sent in
	// the X-API-Key header.
	const curlCode = $derived(`# Send a transactional template
curl -X POST "${apiBaseUrl}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "template_slug": "password-reset",
    "variables": { "name": "Ana", "reset_url": "https://app.example.com/reset/abc" }
  }'

# Or send without a template
curl -X POST "${apiBaseUrl}/emails" \\
  -H "X-API-Key: YOUR_BRAND_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "customer@example.com",
    "subject": "Your receipt",
    "body": "<p>Thanks for your order.</p>",
    "text_body": "Thanks for your order."
  }'

# Check delivery
curl "${apiBaseUrl}/emails/MESSAGE_ID" -H "X-API-Key: YOUR_BRAND_API_KEY"`);

	const nodejsCode = $derived(`const res = await fetch('${apiBaseUrl}/emails', {
  method: 'POST',
  headers: {
    'X-API-Key': process.env.OUTLET_API_KEY,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    to: 'customer@example.com',
    template_slug: 'password-reset',
    variables: { name: 'Ana', reset_url: 'https://app.example.com/reset/abc' }
  })
});
const result = await res.json();
if (!result.success) throw new Error(result.message); // e.g. status "suppressed"
console.log(result.message_id);`);

	const pythonCode = $derived(`import os, requests

res = requests.post(
    '${apiBaseUrl}/emails',
    headers={'X-API-Key': os.environ['OUTLET_API_KEY']},
    json={
        'to': 'customer@example.com',
        'template_slug': 'password-reset',
        'variables': {'name': 'Ana', 'reset_url': 'https://app.example.com/reset/abc'},
    },
)
result = res.json()
if not result['success']:
    raise RuntimeError(result.get('message'))  # e.g. status "suppressed"
print(result['message_id'])`);

	const goCode = $derived(`payload, _ := json.Marshal(map[string]any{
    "to":            "customer@example.com",
    "template_slug": "password-reset",
    "variables":     map[string]string{"name": "Ana", "reset_url": "https://app.example.com/reset/abc"},
})
req, _ := http.NewRequest("POST", "${apiBaseUrl}/emails", bytes.NewReader(payload))
req.Header.Set("X-API-Key", os.Getenv("OUTLET_API_KEY"))
req.Header.Set("Content-Type", "application/json")
resp, err := http.DefaultClient.Do(req)
if err != nil {
    return err
}
defer resp.Body.Close()

var result struct {
    Success   bool   \`json:"success"\`
    MessageID string \`json:"message_id"\`
    Message   string \`json:"message"\`
}
json.NewDecoder(resp.Body).Decode(&result)`);

	const phpCode = $derived(`<?php
$ch = curl_init('${apiBaseUrl}/emails');
curl_setopt_array($ch, [
    CURLOPT_POST => true,
    CURLOPT_RETURNTRANSFER => true,
    CURLOPT_HTTPHEADER => [
        'X-API-Key: ' . getenv('OUTLET_API_KEY'),
        'Content-Type: application/json',
    ],
    CURLOPT_POSTFIELDS => json_encode([
        'to' => 'customer@example.com',
        'template_slug' => 'password-reset',
        'variables' => ['name' => 'Ana', 'reset_url' => 'https://app.example.com/reset/abc'],
    ]),
]);
$result = json_decode(curl_exec($ch), true);`);

	const smtpCode = $derived(`Host:      ${installDomain.replace(/^https?:\/\//, '').replace(/:\d+$/, '')}
Port:      587 (STARTTLS)
Username:  your brand slug
Password:  your brand API key

Optional headers:
  X-Outlet-Tags: welcome,onboarding
  X-Outlet-Meta-OrderID: 12345`);

	let codeExamples = $derived<Record<string, { code: string; language: string }>>({
		curl: { code: curlCode, language: 'bash' },
		nodejs: { code: nodejsCode, language: 'javascript' },
		python: { code: pythonCode, language: 'python' },
		go: { code: goCode, language: 'go' },
		php: { code: phpCode, language: 'php' },
		smtp: { code: smtpCode, language: 'text' }
	});

	let activeExample = $derived(codeExamples[activeCodeTab]);

	onMount(async () => {
		await loadKeys();
	});

	async function loadKeys() {
		loading = true;
		error = '';
		try {
			const response = await listMCPAPIKeys();
			apiKeys = response.keys || [];
		} catch (err: any) {
			console.log('No API keys yet');
		} finally {
			loading = false;
		}
	}

	async function createKey() {
		if (!newKeyName.trim()) {
			error = 'Please enter a name for the API key';
			return;
		}

		creating = true;
		error = '';
		try {
			const response = await createMCPAPIKey({ name: newKeyName.trim() });
			newKeyValue = response.key || '';
			await loadKeys();
			success = "API key created successfully. Copy it now - you won't be able to see it again!";
		} catch (err: any) {
			error = err.message || 'Failed to create API key';
		} finally {
			creating = false;
		}
	}

	function confirmRevokeKey(id: string) {
		revokeKeyId = id;
		showRevokeConfirm = true;
	}

	async function executeRevokeKey() {
		revoking = true;
		try {
			await revokeMCPAPIKey({}, revokeKeyId);
			await loadKeys();
			success = 'API key revoked';
		} catch (err: any) {
			error = err.message || 'Failed to revoke API key';
		} finally {
			revoking = false;
		}
	}

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		copiedText = text;
		setTimeout(() => (copiedText = ''), 2000);
	}

	function closeCreateModal() {
		showCreateModal = false;
		newKeyName = '';
		newKeyValue = '';
	}

</script>

<svelte:head>
	<title>API - Settings</title>
</svelte:head>

<div class="space-y-6">
	{#if error}
		<Alert type="error" title="Error" onclose={() => (error = '')}>
			<p>{error}</p>
		</Alert>
	{/if}

	{#if success}
		<Alert type="success" title="Success" onclose={() => (success = '')}>
			<p>{success}</p>
		</Alert>
	{/if}

	<!-- API Keys -->
	<Card>
		<div class="flex items-center justify-between">
			<div>
				<h2 class="text-lg font-medium text-text mb-1">MCP keys</h2>
				<p class="text-sm text-text-muted">For AI assistants such as Claude that manage Outlet through MCP. They don't work for sending email: apps use their brand's API key, below.</p>
			</div>
			<Button type="primary" size="sm" onclick={() => (showCreateModal = true)}>
				<Plus class="mr-1.5 h-4 w-4" />
				Create Key
			</Button>
		</div>

		{#if loading}
			<div class="flex justify-center py-8">
				<LoadingSpinner />
			</div>
		{:else if apiKeys.length === 0}
			<div class="mt-6 text-center py-8 text-text-muted">
				<Key class="mx-auto h-12 w-12 opacity-50" />
				<p class="mt-2">No MCP keys yet</p>
				<p class="text-sm">Create one to connect an AI assistant over MCP</p>
			</div>
		{:else}
			<div class="mt-6 overflow-x-auto">
				<table class="w-full text-sm">
					<thead>
						<tr class="border-b border-border">
							<th class="text-left py-2 font-medium text-text-muted">Name</th>
							<th class="text-left py-2 font-medium text-text-muted">Key</th>
							<th class="text-left py-2 font-medium text-text-muted">Last Used</th>
							<th class="text-left py-2 font-medium text-text-muted">Created</th>
							<th class="text-right py-2 font-medium text-text-muted">Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each apiKeys as key (key.id)}
							<tr class="border-b border-border/50">
								<td class="py-3 font-medium">{key.name}</td>
								<td class="py-3 font-mono text-text-muted">{key.key_prefix}...</td>
								<td class="py-3 text-text-muted">{formatDate(key.last_used) || 'Never'}</td>
								<td class="py-3 text-text-muted">{formatDate(key.created_at) || 'Never'}</td>
								<td class="py-3 text-right">
									<Button type="danger" size="icon" onclick={() => confirmRevokeKey(key.id)}>
										<Trash2 class="h-4 w-4" />
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</Card>

	<!-- REST API Connection -->
	<Card>
		<div class="flex items-center gap-3 mb-4">
			<Code class="h-5 w-5 text-primary" />
			<div>
				<h2 class="text-lg font-medium text-text">Sending API and SMTP</h2>
				<p class="text-sm text-text-muted">How your apps send email. Each brand has its own API key, shown in that brand's settings.</p>
			</div>
		</div>

		<div class="space-y-4">
			<div>
				<label class="form-label">Base URL</label>
				<div class="flex gap-2">
					<Input type="text" value={apiBaseUrl} readonly class="font-mono text-sm" />
					<Button type="secondary" onclick={() => copyToClipboard(apiBaseUrl)}>
						{#if copiedText === apiBaseUrl}
							<CheckCircle class="h-4 w-4 text-green-500" />
						{:else}
							<Copy class="h-4 w-4" />
						{/if}
					</Button>
				</div>
			</div>

			<div>
				<label class="form-label">Authentication</label>
				<p class="text-sm text-text-muted mb-2">Send the brand's API key in the X-API-Key header:</p>
				<pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">X-API-Key: YOUR_BRAND_API_KEY</pre>
			</div>
		</div>
	</Card>

	<!-- Code Examples -->
	<Card>
		<h2 class="text-lg font-medium text-text mb-4">Code Examples</h2>

		<div class="mb-4">
			<Tabs tabs={codeTabs} bind:activeTab={activeCodeTab} variant="pills" />
		</div>

		<CodeBlock code={activeExample.code} language={activeExample.language} />
	</Card>

</div>

<!-- Create Key Modal -->
<Modal bind:show={showCreateModal} title="Create API Key" onclose={closeCreateModal}>
	{#if newKeyValue}
		<div class="space-y-4">
			<Alert type="warning" title="Save your API key">
				<p>This is the only time you'll see this key. Copy it now and store it securely.</p>
			</Alert>
			<div>
				<label class="form-label">API Key</label>
				<div class="flex gap-2">
					<Input type="text" value={newKeyValue} readonly class="font-mono text-sm" />
					<Button type="secondary" onclick={() => copyToClipboard(newKeyValue)}>
						{#if copiedText === newKeyValue}
							<CheckCircle class="h-4 w-4 text-green-500" />
						{:else}
							<Copy class="h-4 w-4" />
						{/if}
					</Button>
				</div>
			</div>
		</div>
	{:else}
		<div class="space-y-4">
			<div>
				<label for="key-name" class="form-label">Key Name</label>
				<Input
					type="text"
					id="key-name"
					bind:value={newKeyName}
					placeholder="e.g., Production API, Development"
				/>
				<p class="mt-1 text-xs text-text-muted">A descriptive name to identify this key</p>
			</div>
		</div>
	{/if}

	{#snippet footer()}
		<div class="flex justify-end gap-3">
			{#if newKeyValue}
				<Button type="primary" onclick={closeCreateModal}>Done</Button>
			{:else}
				<Button type="secondary" onclick={closeCreateModal}>Cancel</Button>
				<Button type="primary" onclick={createKey} disabled={creating || !newKeyName.trim()}>
					{creating ? 'Creating...' : 'Create Key'}
				</Button>
			{/if}
		</div>
	{/snippet}
</Modal>

<AlertDialog
	bind:open={showRevokeConfirm}
	title="Revoke API Key"
	description="Are you sure you want to revoke this API key? This cannot be undone."
	actionLabel={revoking ? 'Revoking...' : 'Revoke'}
	actionType="danger"
	onAction={executeRevokeKey}
	onCancel={() => showRevokeConfirm = false}
/>
