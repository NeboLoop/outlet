<script lang="ts">
	import { page } from '$app/stores';
	import { getListEmbedCode, type EmbedCodeResponse } from '$lib/api';
	import {
		Button,
		Card,
		LoadingSpinner,
		Tabs
	} from '$lib/components/ui';
	import {
		Copy,
		Check,
		ExternalLink
	} from '@lucide/svelte';
	import { getListContext } from '../listContext';

	const ctx = getListContext();
	let listId = $derived($page.params.id!);

	// State
	let embedCode = $state<EmbedCodeResponse | null>(null);
	let embedLoading = $state(true);
	let urlCopied = $state(false);
	let htmlCopied = $state(false);
	let apiCodeTab = $state('curl');

	$effect(() => {
		loadEmbedCode();
	});

	async function loadEmbedCode() {
		embedLoading = true;
		try {
			embedCode = await getListEmbedCode({}, listId);
		} catch (err) {
			console.error('Failed to load embed code:', err);
		} finally {
			embedLoading = false;
		}
	}

	function copyUrl() {
		if (!embedCode) return;
		const url = `${embedCode.base_url}/s/${embedCode.public_id}`;
		navigator.clipboard.writeText(url);
		urlCopied = true;
		setTimeout(() => { urlCopied = false; }, 2000);
	}

	function copyHtml() {
		if (!embedCode) return;
		navigator.clipboard.writeText(embedCode.html);
		htmlCopied = true;
		setTimeout(() => { htmlCopied = false; }, 2000);
	}
</script>

{#if embedLoading}
	<div class="flex justify-center py-12">
		<LoadingSpinner />
	</div>
{:else if embedCode}
	<div class="space-y-6">
		<!-- Hosted Form URL -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-2">Hosted Subscribe Form</h2>
			<p class="text-sm text-text-muted mb-4">
				Use this URL to link to a ready-to-use subscription page for this list.
			</p>
			<div class="flex items-center gap-2 flex-wrap sm:flex-nowrap">
				<div class="flex-1 min-w-0 bg-bg-secondary px-3 py-2 rounded-md overflow-hidden">
					<code class="text-sm font-mono text-text truncate block">
						{embedCode.base_url}/s/{embedCode.public_id}
					</code>
				</div>
				<Button type="secondary" onclick={copyUrl} class="flex-shrink-0">
					{#if urlCopied}
						<Check class="mr-2 h-4 w-4 text-green-500" />
						Copied
					{:else}
						<Copy class="mr-2 h-4 w-4" />
						Copy URL
					{/if}
				</Button>
				<a
					href="{embedCode.base_url}/s/{embedCode.public_id}"
					target="_blank"
					rel="noopener noreferrer"
					class="btn btn-secondary flex-shrink-0"
				>
					<ExternalLink class="mr-2 h-4 w-4" />
					Preview
				</a>
			</div>
		</Card>

		<!-- Embeddable HTML -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-2">Embeddable HTML Form</h2>
			<p class="text-sm text-text-muted mb-4">
				Copy this HTML code to embed a subscribe form on your website.
			</p>
			<div class="relative">
				<pre class="bg-bg-secondary p-4 rounded-md overflow-x-auto text-sm font-mono text-text whitespace-pre-wrap">{embedCode.html}</pre>
				<div class="absolute top-2 right-2">
					<Button type="secondary" size="sm" onclick={copyHtml}>
						{#if htmlCopied}
							<Check class="mr-1 h-3 w-3 text-green-500" />
							Copied
						{:else}
							<Copy class="mr-1 h-3 w-3" />
							Copy
						{/if}
					</Button>
				</div>
			</div>
		</Card>

		<!-- API Subscription -->
		<Card>
			<h2 class="text-lg font-medium text-text mb-2">API Subscription</h2>
			<p class="text-sm text-text-muted mb-4">
				To subscribe users programmatically, use the Outlet SDK or API. Replace <code class="text-xs bg-bg-secondary px-1 py-0.5 rounded">your-api-key</code> with your organization's API key.
			</p>

			<Tabs
				tabs={[
					{ id: 'curl', label: 'cURL' },
					{ id: 'typescript', label: 'TypeScript' },
					{ id: 'go', label: 'Go' },
					{ id: 'python', label: 'Python' },
					{ id: 'php', label: 'PHP' }
				]}
				bind:activeTab={apiCodeTab}
				variant="pills"
			/>

			<div class="mt-4">
				{#if apiCodeTab === 'curl'}
					<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full">
						<pre class="text-sm font-mono text-text whitespace-pre-wrap break-all">{`curl -X POST ${embedCode?.base_url || 'https://your-outlet-instance.com'}/api/sdk/v1/lists/${ctx.list?.slug || 'your-list-slug'}/subscribe \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer your-api-key" \\
  -d '{
    "email": "user@example.com",
    "name": "John Doe"
  }'`}</pre>
					</div>
				{:else if apiCodeTab === 'typescript'}
					<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full">
						<pre class="text-sm font-mono text-text whitespace-pre-wrap break-all">{`import { Outlet } from '@outlet/sdk';

const client = new Outlet(
  'your-api-key',
  '${embedCode?.base_url || 'https://your-outlet-instance.com'}'
);

await client.lists.subscribeToList('${ctx.list?.slug || 'your-list-slug'}', {
  email: 'user@example.com',
  name: 'John Doe'
});`}</pre>
					</div>
					<p class="text-xs text-text-muted mt-2">
						Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">npm install @outlet/sdk</code>
					</p>
				{:else if apiCodeTab === 'go'}
					<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full">
						<pre class="text-sm font-mono text-text whitespace-pre-wrap break-all">{`package main

import (
    "context"
    outlet "github.com/localrivet/outlet/sdk/go"
)

func main() {
    client := outlet.NewClient(
        "your-api-key",
        "${embedCode?.base_url || 'https://your-outlet-instance.com'}",
    )

    _, err := client.Lists.SubscribeToList(
        context.Background(),
        "${ctx.list?.slug || 'your-list-slug'}",
        &outlet.SubscribeRequest{
            Email: "user@example.com",
            Name:  "John Doe",
        },
    )
}`}</pre>
					</div>
					<p class="text-xs text-text-muted mt-2">
						Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">go get github.com/localrivet/outlet/sdk/go</code>
					</p>
				{:else if apiCodeTab === 'python'}
					<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full">
						<pre class="text-sm font-mono text-text whitespace-pre-wrap break-all">{`from outlet_sdk import Outlet, SubscribeRequest

client = Outlet(
    api_key="your-api-key",
    base_url="${embedCode?.base_url || 'https://your-outlet-instance.com'}"
)

client.lists.subscribe_to_list(
    "${ctx.list?.slug || 'your-list-slug'}",
    SubscribeRequest(
        email="user@example.com",
        name="John Doe"
    )
)`}</pre>
					</div>
					<p class="text-xs text-text-muted mt-2">
						Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">pip install outlet-sdk</code>
					</p>
				{:else if apiCodeTab === 'php'}
					<div class="bg-bg-secondary p-4 rounded-md overflow-x-auto max-w-full">
						<pre class="text-sm font-mono text-text whitespace-pre-wrap break-all">{`<?php

use Outlet\\SDK\\Client;
use Outlet\\SDK\\Types\\SubscribeRequest;

$client = new Client(
    'your-api-key',
    '${embedCode?.base_url || 'https://your-outlet-instance.com'}'
);

$client->lists->subscribeToList(
    '${ctx.list?.slug || 'your-list-slug'}',
    new SubscribeRequest(
        email: 'user@example.com',
        name: 'John Doe'
    )
);`}</pre>
					</div>
					<p class="text-xs text-text-muted mt-2">
						Install: <code class="bg-bg-secondary px-1 py-0.5 rounded">composer require outlet/sdk</code>
					</p>
				{/if}
			</div>
		</Card>
	</div>
{/if}
