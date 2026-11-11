<script lang="ts">
	import { Button } from '$lib/components/ui';
	import { Sparkles, Copy, Check, RefreshCw } from '@lucide/svelte';

	// Hands a step to an AI connected to Outlet over MCP. Outlet doesn't
	// browse or write itself: the AI does it through Outlet's tools, and
	// onrefresh reads back what it saved.

	let {
		prompt,
		disabled = false,
		hint = '',
		onrefresh
	}: { prompt: string; disabled?: boolean; hint?: string; onrefresh: () => void } = $props();

	let copied = $state(false);

	async function copy() {
		await navigator.clipboard.writeText(prompt);
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}
</script>

<div class="setup-assist">
	<div>
		<p class="flex items-center gap-2 text-sm font-medium text-text">
			<Sparkles class="h-4 w-4 shrink-0 text-primary" aria-hidden="true" />
			Let your AI do it
		</p>
		<div class="min-w-0 mt-1">
			<p class="form-hint">
				Ask an AI connected to Outlet (Claude, ChatGPT, Cursor or any MCP client) to do this step. It saves through
				Outlet's tools, and what it saved shows here. Not connected yet?
				<a href="/settings/mcp" class="link link-primary">Connect an AI</a>.
			</p>
			<p class="setup-assist-prompt">{prompt}</p>
			{#if hint}
				<p class="form-hint mt-2">{hint}</p>
			{/if}
			<div class="flex flex-wrap gap-2 mt-3">
				<Button type="secondary" size="sm" onclick={copy} {disabled}>
					{#if copied}
						<Check class="mr-2 h-4 w-4" />
						Copied
					{:else}
						<Copy class="mr-2 h-4 w-4" />
						Copy what to ask
					{/if}
				</Button>
				<Button type="ghost" size="sm" onclick={onrefresh}>
					<RefreshCw class="mr-2 h-4 w-4" />
					Show what it saved
				</Button>
			</div>
		</div>
	</div>
</div>
