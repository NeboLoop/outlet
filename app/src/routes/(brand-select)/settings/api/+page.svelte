<script lang="ts">
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
		{ id: 'php', label: 'PHP' }
	];
	let activeCodeTab = $state('curl');

	// Code examples
	const curlCode = $derived(`# List contacts
curl -X GET "${apiBaseUrl}/contacts" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json"

# Create a contact
curl -X POST "${apiBaseUrl}/contacts" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "email": "user@example.com",
    "name": "John Doe",
    "tags": ["customer", "newsletter"]
  }'

# Send transactional email
curl -X POST "${apiBaseUrl}/emails/send" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "to": "recipient@example.com",
    "subject": "Welcome!",
    "html": "<h1>Hello</h1><p>Welcome to our service.</p>"
  }'`);

	const nodejsCode = $derived(`import axios from 'axios';

const api = axios.create({
  baseURL: '${apiBaseUrl}',
  headers: {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json'
  }
});

// List contacts
const contacts = await api.get('/contacts');
console.log(contacts.data);

// Create a contact
const newContact = await api.post('/contacts', {
  email: 'user@example.com',
  name: 'John Doe',
  tags: ['customer', 'newsletter']
});

// Send transactional email
await api.post('/emails/send', {
  to: 'recipient@example.com',
  subject: 'Welcome!',
  html: '<h1>Hello</h1><p>Welcome to our service.</p>'
});`);

	const pythonCode = $derived(`import requests

API_KEY = 'YOUR_API_KEY'
BASE_URL = '${apiBaseUrl}'
headers = {
    'Authorization': f'Bearer {API_KEY}',
    'Content-Type': 'application/json'
}

# List contacts
response = requests.get(f'{BASE_URL}/contacts', headers=headers)
contacts = response.json()

# Create a contact
new_contact = requests.post(f'{BASE_URL}/contacts', headers=headers, json={
    'email': 'user@example.com',
    'name': 'John Doe',
    'tags': ['customer', 'newsletter']
})

# Send transactional email
requests.post(f'{BASE_URL}/emails/send', headers=headers, json={
    'to': 'recipient@example.com',
    'subject': 'Welcome!',
    'html': '<h1>Hello</h1><p>Welcome to our service.</p>'
})`);

	const goCode = $derived(`package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

const (
    apiKey  = "YOUR_API_KEY"
    baseURL = "${apiBaseUrl}"
)

func main() {
    client := &http.Client{}

    // Create contact
    contact := map[string]interface{}{
        "email": "user@example.com",
        "name":  "John Doe",
        "tags":  []string{"customer", "newsletter"},
    }
    body, _ := json.Marshal(contact)

    req, _ := http.NewRequest("POST", baseURL+"/contacts", bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, _ := client.Do(req)
    defer resp.Body.Close()
}`);

	const phpCode = $derived(`<?php

$apiKey = 'YOUR_API_KEY';
$baseUrl = '${apiBaseUrl}';

// Helper function for API requests
function apiRequest($method, $endpoint, $data = null) {
    global $apiKey, $baseUrl;

    $ch = curl_init($baseUrl . $endpoint);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_CUSTOMREQUEST, $method);
    curl_setopt($ch, CURLOPT_HTTPHEADER, [
        'Authorization: Bearer ' . $apiKey,
        'Content-Type: application/json'
    ]);

    if ($data) {
        curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
    }

    $response = curl_exec($ch);
    curl_close($ch);
    return json_decode($response, true);
}

// List contacts
$contacts = apiRequest('GET', '/contacts');

// Create a contact
$newContact = apiRequest('POST', '/contacts', [
    'email' => 'user@example.com',
    'name' => 'John Doe',
    'tags' => ['customer', 'newsletter']
]);

// Send transactional email
apiRequest('POST', '/emails/send', [
    'to' => 'recipient@example.com',
    'subject' => 'Welcome!',
    'html' => '<h1>Hello</h1><p>Welcome to our service.</p>'
]);`);

	let codeExamples = $derived<Record<string, { code: string; language: string }>>({
		curl: { code: curlCode, language: 'bash' },
		nodejs: { code: nodejsCode, language: 'javascript' },
		python: { code: pythonCode, language: 'python' },
		go: { code: goCode, language: 'go' },
		php: { code: phpCode, language: 'php' }
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

	function formatDate(dateStr: string | null | undefined): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		});
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
				<h2 class="text-lg font-medium text-text mb-1">API Keys</h2>
				<p class="text-sm text-text-muted">Manage API keys for REST API and SMTP authentication</p>
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
				<p class="mt-2">No API keys yet</p>
				<p class="text-sm">Create an API key to use the REST API or SMTP</p>
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
								<td class="py-3 text-text-muted">{formatDate(key.last_used)}</td>
								<td class="py-3 text-text-muted">{formatDate(key.created_at)}</td>
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
				<h2 class="text-lg font-medium text-text">REST API</h2>
				<p class="text-sm text-text-muted">Connect to Outlet programmatically via the REST API</p>
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
				<p class="text-sm text-text-muted mb-2">Include your API key in the Authorization header:</p>
				<pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">Authorization: Bearer YOUR_API_KEY</pre>
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

	<!-- SDKs -->
	<Card>
		<div class="flex items-center gap-3 mb-4">
			<Package class="h-5 w-5 text-primary" />
			<div>
				<h2 class="text-lg font-medium text-text">Official SDKs</h2>
				<p class="text-sm text-text-muted">Pre-built client libraries for popular languages</p>
			</div>
		</div>

		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
			<div class="p-4 bg-surface-secondary rounded-lg">
				<div class="flex items-center gap-2 mb-2">
					<span class="text-lg font-medium text-text">TypeScript</span>
				</div>
				<p class="text-sm text-text-muted mb-3">Full TypeScript support with types</p>
				<code class="block text-xs bg-surface-tertiary px-2 py-1 rounded font-mono text-text-muted">
					npm install @outlet/sdk
				</code>
			</div>

			<div class="p-4 bg-surface-secondary rounded-lg">
				<div class="flex items-center gap-2 mb-2">
					<span class="text-lg font-medium text-text">Python</span>
				</div>
				<p class="text-sm text-text-muted mb-3">Python 3.8+ support</p>
				<code class="block text-xs bg-surface-tertiary px-2 py-1 rounded font-mono text-text-muted">
					pip install outlet-sdk
				</code>
			</div>

			<div class="p-4 bg-surface-secondary rounded-lg">
				<div class="flex items-center gap-2 mb-2">
					<span class="text-lg font-medium text-text">Go</span>
				</div>
				<p class="text-sm text-text-muted mb-3">Go 1.20+ support</p>
				<code class="block text-xs bg-surface-tertiary px-2 py-1 rounded font-mono text-text-muted">
					go get github.com/outlet/sdk-go
				</code>
			</div>

			<div class="p-4 bg-surface-secondary rounded-lg">
				<div class="flex items-center gap-2 mb-2">
					<span class="text-lg font-medium text-text">PHP</span>
				</div>
				<p class="text-sm text-text-muted mb-3">PHP 8.0+ support</p>
				<code class="block text-xs bg-surface-tertiary px-2 py-1 rounded font-mono text-text-muted">
					composer require outlet/sdk
				</code>
			</div>
		</div>

		<div class="mt-6 pt-4 border-t border-border">
			<h3 class="font-medium text-text mb-3">SDK Quick Start (TypeScript)</h3>
			<CodeBlock
				code={`import { Outlet } from '@outlet/sdk';

const outlet = new Outlet('YOUR_API_KEY', '${apiBaseUrl.replace('/sdk/v1', '')}');

// Send a transactional email
const result = await outlet.emails.sendEmail({
  to: 'user@example.com',
  subject: 'Welcome!',
  html_body: '<h1>Welcome to our platform</h1>',
});

console.log('Message ID:', result.message_id);`}
				language="typescript"
			/>
		</div>
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
