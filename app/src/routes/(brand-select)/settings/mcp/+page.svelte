<script lang="ts">
	import { Card, Button, Input, Alert, Badge } from '$lib/components/ui';
	import { Copy, ChevronDown, ChevronRight, CheckCircle } from '@lucide/svelte';
	import { browser } from '$app/environment';

	// Dynamic endpoint based on install domain
	let installDomain = $state(browser ? window.location.origin : 'https://your-domain.com');
	let mcpEndpoint = $derived(`${installDomain}/mcp`);

	let error = $state('');
	let success = $state('');
	let copiedEndpoint = $state(false);

	// Setup guide state
	let activeTab = $state<'claude-desktop' | 'chatgpt' | 'claude-code' | 'cursor'>('claude-desktop');

	// MCP Tools grouped by category - uses unified resource/action pattern
	// All tools require brand(resource: brand, action: select) first to select a brand
	const mcpTools = {
		Brand: [
			{ name: 'brand(resource: brand, action: list)', desc: 'List brands you have access to' },
			{ name: 'brand(resource: brand, action: select)', desc: 'Select a brand to work with' },
			{ name: 'brand(resource: brand, action: get)', desc: 'Get current brand details' },
			{ name: 'brand(resource: brand, action: update)', desc: 'Update brand settings' },
			{ name: 'brand(resource: brand, action: create)', desc: 'Create a new brand' },
			{ name: 'brand(resource: brand, action: delete)', desc: 'Delete a brand' },
			{ name: 'brand(resource: brand, action: dashboard_stats)', desc: 'Get dashboard statistics' },
			{ name: 'brand(resource: domain, action: list)', desc: 'List domain identities' },
			{ name: 'brand(resource: domain, action: create)', desc: 'Add a domain for email sending' },
			{ name: 'brand(resource: domain, action: get)', desc: 'Get domain verification status' },
			{ name: 'brand(resource: domain, action: refresh)', desc: 'Refresh domain DNS verification' },
			{ name: 'brand(resource: domain, action: delete)', desc: 'Remove a domain identity' }
		],
		'Email Lists & Sequences': [
			{ name: 'email(resource: list, action: create)', desc: 'Create an email list' },
			{ name: 'email(resource: list, action: list)', desc: 'List all email lists' },
			{ name: 'email(resource: list, action: get)', desc: 'Get list details' },
			{ name: 'email(resource: list, action: stats)', desc: 'Get list statistics' },
			{ name: 'email(resource: list, action: subscribers)', desc: 'List subscribers in a list' },
			{ name: 'email(resource: list, action: subscribe)', desc: 'Subscribe contact to list' },
			{ name: 'email(resource: list, action: unsubscribe)', desc: 'Unsubscribe contact from list' },
			{ name: 'email(resource: sequence, action: create)', desc: 'Create an email sequence' },
			{ name: 'email(resource: sequence, action: list)', desc: 'List all sequences' },
			{ name: 'email(resource: sequence, action: get)', desc: 'Get sequence with templates' },
			{ name: 'email(resource: sequence, action: stats)', desc: 'Get sequence statistics' },
			{ name: 'email(resource: template, action: create)', desc: 'Add email to sequence' },
			{ name: 'email(resource: template, action: list)', desc: 'List sequence emails' },
			{ name: 'email(resource: template, action: update)', desc: 'Update sequence email' }
		],
		'Sequence Enrollments': [
			{ name: 'email(resource: enrollment, action: enroll)', desc: 'Enroll contact in sequence' },
			{ name: 'email(resource: enrollment, action: unenroll)', desc: 'Remove contact from sequence' },
			{ name: 'email(resource: enrollment, action: pause)', desc: 'Pause contact\'s sequence' },
			{ name: 'email(resource: enrollment, action: resume)', desc: 'Resume contact\'s sequence' },
			{ name: 'email(resource: enrollment, action: list)', desc: 'List contact enrollments' },
			{ name: 'email(resource: entry_rule, action: create)', desc: 'Create sequence entry rule' },
			{ name: 'email(resource: entry_rule, action: list)', desc: 'List sequence entry rules' },
			{ name: 'email(resource: queue, action: list)', desc: 'View pending emails in queue' },
			{ name: 'email(resource: queue, action: cancel)', desc: 'Cancel a queued email' }
		],
		Campaigns: [
			{ name: 'campaign(action: create)', desc: 'Create a new campaign' },
			{ name: 'campaign(action: list)', desc: 'List all campaigns' },
			{ name: 'campaign(action: get)', desc: 'Get campaign details' },
			{ name: 'campaign(action: update)', desc: 'Update campaign content' },
			{ name: 'campaign(action: delete)', desc: 'Delete a campaign' },
			{ name: 'campaign(action: schedule)', desc: 'Schedule campaign for future send' },
			{ name: 'campaign(action: send)', desc: 'Send campaign immediately' },
			{ name: 'campaign(action: stats)', desc: 'Get campaign statistics' }
		],
		Contacts: [
			{ name: 'contact(action: create)', desc: 'Create a new contact' },
			{ name: 'contact(action: list)', desc: 'List contacts with pagination' },
			{ name: 'contact(action: get)', desc: 'Get contact details' },
			{ name: 'contact(action: update)', desc: 'Update contact information' },
			{ name: 'contact(action: add_tags)', desc: 'Add tags to contact' },
			{ name: 'contact(action: remove_tags)', desc: 'Remove tags from contact' },
			{ name: 'contact(action: unsubscribe)', desc: 'Unsubscribe contact globally' },
			{ name: 'contact(action: block)', desc: 'Block a contact' },
			{ name: 'contact(action: unblock)', desc: 'Unblock a contact' },
			{ name: 'contact(action: activity)', desc: 'Get contact activity history' }
		],
		'Transactional Email': [
			{ name: 'transactional(action: create)', desc: 'Create transactional template' },
			{ name: 'transactional(action: list)', desc: 'List transactional templates' },
			{ name: 'transactional(action: get)', desc: 'Get template details' },
			{ name: 'transactional(action: update)', desc: 'Update template' },
			{ name: 'transactional(action: delete)', desc: 'Delete template' },
			{ name: 'transactional(action: stats)', desc: 'Get template statistics' }
		],
		'Email Designs': [
			{ name: 'design(action: create)', desc: 'Create email design template' },
			{ name: 'design(action: list)', desc: 'List all designs' },
			{ name: 'design(action: get)', desc: 'Get design details' },
			{ name: 'design(action: update)', desc: 'Update design' },
			{ name: 'design(action: delete)', desc: 'Delete design' }
		],
		Statistics: [
			{ name: 'stats(resource: overview, action: get)', desc: 'Get overall org statistics' },
			{ name: 'stats(resource: email, action: get)', desc: 'Get email performance stats' },
			{ name: 'stats(resource: contact, action: get)', desc: 'Get individual contact stats' }
		],
		Blocklist: [
			{ name: 'blocklist(resource: suppression, action: list)', desc: 'List suppressed emails' },
			{ name: 'blocklist(resource: suppression, action: add)', desc: 'Add email to suppression list' },
			{ name: 'blocklist(resource: suppression, action: delete)', desc: 'Remove from suppression' },
			{ name: 'blocklist(resource: suppression, action: bulk_add)', desc: 'Bulk add to suppression' },
			{ name: 'blocklist(resource: suppression, action: clear)', desc: 'Clear suppression list' },
			{ name: 'blocklist(resource: domain, action: list)', desc: 'List blocked domains' },
			{ name: 'blocklist(resource: domain, action: add)', desc: 'Block a domain' },
			{ name: 'blocklist(resource: domain, action: delete)', desc: 'Unblock domain' },
			{ name: 'blocklist(resource: domain, action: bulk_add)', desc: 'Bulk block domains' }
		],
		GDPR: [
			{ name: 'gdpr(action: lookup)', desc: 'Look up contact by email' },
			{ name: 'gdpr(action: export)', desc: 'Export all contact data' },
			{ name: 'gdpr(action: delete)', desc: 'Delete contact and all data' },
			{ name: 'gdpr(action: get_consent)', desc: 'Get contact consent status' },
			{ name: 'gdpr(action: update_consent)', desc: 'Update consent preferences' }
		],
		Webhooks: [
			{ name: 'webhook(action: create)', desc: 'Register a webhook endpoint' },
			{ name: 'webhook(action: list)', desc: 'List all webhooks' },
			{ name: 'webhook(action: get)', desc: 'Get webhook details' },
			{ name: 'webhook(action: update)', desc: 'Update webhook configuration' },
			{ name: 'webhook(action: delete)', desc: 'Delete webhook' },
			{ name: 'webhook(action: test)', desc: 'Send test webhook' },
			{ name: 'webhook(action: logs)', desc: 'View webhook delivery logs' }
		]
	};

	let expandedCategories = $state<Record<string, boolean>>({
		Brand: true,
		'Email Lists & Sequences': false,
		'Sequence Enrollments': false,
		Campaigns: false,
		Contacts: false,
		'Transactional Email': false,
		'Email Designs': false,
		Statistics: false,
		Blocklist: false,
		GDPR: false,
		Webhooks: false
	});

	function copyEndpoint() {
		navigator.clipboard.writeText(mcpEndpoint);
		copiedEndpoint = true;
		setTimeout(() => (copiedEndpoint = false), 2000);
	}

	function toggleCategory(category: string) {
		expandedCategories[category] = !expandedCategories[category];
	}

	// Calculate total tools
	let totalTools = $derived(Object.values(mcpTools).flat().length);
</script>

<svelte:head>
	<title>MCP Configuration - Outlet</title>
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

	<!-- MCP Endpoint -->
	<Card>
		<div class="flex items-start justify-between gap-4">
			<div class="flex-1">
				<h2 class="text-lg font-medium text-text mb-1">MCP Endpoint</h2>
				<p class="text-sm text-text-muted">
					Connect AI assistants to help you configure and manage your platform
				</p>
			</div>
			<Badge type="success">Active</Badge>
		</div>

		<div class="mt-4">
			<label class="form-label">Your MCP Server URL</label>
			<div class="flex gap-2">
				<Input type="text" value={mcpEndpoint} readonly class="font-mono text-sm" />
				<Button type="secondary" onclick={copyEndpoint}>
					{#if copiedEndpoint}
						<CheckCircle class="h-4 w-4 text-green-500" />
					{:else}
						<Copy class="h-4 w-4" />
					{/if}
				</Button>
			</div>
			<p class="mt-2 text-sm text-text-muted">
				MCP uses OAuth for authentication. When connecting, your AI assistant will open a browser
				window for you to log in.
			</p>
		</div>
	</Card>

	<!-- Setup Instructions -->
	<Card>
		<h2 class="text-lg font-medium text-text mb-1">Setup Instructions</h2>
		<p class="text-sm text-text-muted">Connect your favorite AI assistant to Outlet</p>

		<div class="mt-4 flex flex-wrap gap-2">
			<Button
				type={activeTab === 'claude-desktop' ? 'primary' : 'secondary'}
				size="sm"
				onclick={() => (activeTab = 'claude-desktop')}
			>
				Claude Desktop
			</Button>
			<Button
				type={activeTab === 'chatgpt' ? 'primary' : 'secondary'}
				size="sm"
				onclick={() => (activeTab = 'chatgpt')}
			>
				ChatGPT
			</Button>
			<Button
				type={activeTab === 'claude-code' ? 'primary' : 'secondary'}
				size="sm"
				onclick={() => (activeTab = 'claude-code')}
			>
				Claude Code
			</Button>
			<Button
				type={activeTab === 'cursor' ? 'primary' : 'secondary'}
				size="sm"
				onclick={() => (activeTab = 'cursor')}
			>
				Cursor / IDE
			</Button>
		</div>

		<div class="mt-4 rounded-lg bg-surface-secondary p-4">
			{#if activeTab === 'claude-desktop'}
				<div class="flex items-center gap-2 mb-3">
					<h4 class="font-medium text-text">Claude Desktop Setup</h4>
					<Badge type="success" size="sm">OAuth</Badge>
				</div>
				<p class="text-sm text-text-muted mb-3">
					Claude Desktop uses OAuth for secure authentication. Add the following to your <code
						class="bg-surface-tertiary px-1 rounded">claude_desktop_config.json</code
					>:
				</p>
				<pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">{`{
  "mcpServers": {
    "outlet": {
      "url": "${mcpEndpoint}"
    }
  }
}`}</pre>
				<p class="mt-3 text-sm text-text-muted">
					When you connect, Claude Desktop will open a browser window for you to log in and
					authorize access. OAuth discovery is handled automatically.
				</p>
			{:else if activeTab === 'chatgpt'}
				<div class="flex items-center gap-2 mb-3">
					<h4 class="font-medium text-text">ChatGPT Setup</h4>
					<Badge type="success" size="sm">OAuth</Badge>
				</div>
				<p class="text-sm text-text-muted mb-3">
					ChatGPT uses OAuth for secure authentication when connecting to MCP servers:
				</p>
				<ol class="text-sm text-text-muted space-y-2 list-decimal list-inside">
					<li>Go to <strong>Configure</strong> &rarr; <strong>Actions</strong></li>
					<li>Click <strong>Create new action</strong></li>
					<li>
						Set the server URL to: <code class="bg-surface-tertiary px-1 rounded"
							>{mcpEndpoint}</code
						>
					</li>
				</ol>
				<p class="mt-3 text-sm text-text-muted">
					ChatGPT will prompt you to authorize access when first connecting. OAuth is discovered automatically from the server.
				</p>
			{:else if activeTab === 'claude-code'}
				<div class="flex items-center gap-2 mb-3">
					<h4 class="font-medium text-text">Claude Code Setup</h4>
					<Badge type="success" size="sm">OAuth</Badge>
				</div>
				<p class="text-sm text-text-muted mb-3">
					Add the following to your <code class="bg-surface-tertiary px-1 rounded">~/.claude/settings.json</code>:
				</p>
				<pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">{`{
  "mcpServers": {
    "outlet": {
      "url": "${mcpEndpoint}"
    }
  }
}`}</pre>
				<p class="mt-3 text-sm text-text-muted">
					Claude Code will open a browser window for you to log in and authorize access.
				</p>
			{:else if activeTab === 'cursor'}
				<div class="flex items-center gap-2 mb-3">
					<h4 class="font-medium text-text">Cursor / Windsurf / VS Code Setup</h4>
					<Badge type="success" size="sm">OAuth</Badge>
				</div>
				<p class="text-sm text-text-muted mb-3">
					Add to your IDE's MCP configuration:
				</p>
				<pre class="bg-surface-tertiary p-3 rounded text-sm overflow-x-auto font-mono">{`{
  "mcp.servers": [
    {
      "name": "Outlet",
      "url": "${mcpEndpoint}"
    }
  ]
}`}</pre>
				<p class="mt-3 text-sm text-text-muted">
					Your IDE will open a browser window for you to log in and authorize access. Configuration file location varies by IDE.
				</p>
			{/if}
		</div>
	</Card>

	<!-- Available Tools -->
	<Card>
		<h2 class="text-lg font-medium text-text mb-1">Available Tools ({totalTools} tools)</h2>
		<p class="text-sm text-text-muted">
			MCP tools that AI assistants can use to help you manage your platform
		</p>

		<div class="mt-4 space-y-2">
			{#each Object.entries(mcpTools) as [category, tools]}
				<div class="border border-border rounded-lg">
					<button
						type="button"
						class="w-full flex items-center justify-between p-3 hover:bg-surface-secondary transition-colors"
						onclick={() => toggleCategory(category)}
					>
						<span class="font-medium text-text">{category}</span>
						<div class="flex items-center gap-2">
							<Badge type="neutral">{tools.length}</Badge>
							{#if expandedCategories[category]}
								<ChevronDown class="h-4 w-4 text-text-muted" />
							{:else}
								<ChevronRight class="h-4 w-4 text-text-muted" />
							{/if}
						</div>
					</button>
					{#if expandedCategories[category]}
						<div class="border-t border-border p-3 space-y-2">
							{#each tools as tool}
								<div class="flex items-start gap-3 text-sm">
									<code
										class="bg-surface-tertiary px-1.5 py-0.5 rounded text-xs font-mono text-primary"
										>{tool.name}</code
									>
									<span class="text-text-muted">{tool.desc}</span>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			{/each}
		</div>
	</Card>
</div>
