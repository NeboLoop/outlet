<script lang="ts">
	import { page } from '$app/stores';
	import type { OutboundCampaignResults } from '$lib/api';
	import { Badge } from '$lib/components/ui';
	import { BOUNCE_LIMIT, percent } from './shared';

	let { campaigns }: { campaigns: OutboundCampaignResults[] } = $props();

	let base = $derived(`/${$page.params.brandSlug}/outbound`);
	let anyOpens = $derived(campaigns.some((c) => c.track_opens));
</script>

<div class="data-table overflow-x-auto">
	<table class="w-full text-sm">
		<thead>
			<tr>
				<th class="text-left">Campaign</th>
				<th class="text-right">Reached</th>
				<th class="text-right">Replied</th>
				<th class="text-right">Opened</th>
				<th class="text-right">Clicked</th>
				<th class="text-right">Bounced</th>
				<th class="text-right">Spam blocks</th>
				<th class="text-right">Opt-outs</th>
				<th class="text-right">Drafts waiting</th>
			</tr>
		</thead>
		<tbody>
			{#each campaigns as c (c.id)}
				<tr>
					<td>
						<a href="{base}/campaigns/{c.id}" class="block group">
							<span class="font-medium text-text group-hover:text-primary transition-colors">{c.name}</span>
							<span class="block mt-1">
								<Badge variant={c.status === 'active' ? 'success' : 'warning'} size="sm">{c.status}</Badge>
							</span>
						</a>
					</td>
					<td class="text-right">{c.reached}</td>
					<td class="text-right">
						{c.replied}
						<span class="block text-xs text-text-muted">{percent(c.reply_rate)}</span>
					</td>
					<td class="text-right">
						{#if c.track_opens}
							{c.opened}
							<span class="block text-xs text-text-muted">{percent(c.open_rate)}</span>
						{:else}
							<span class="text-text-muted">Off</span>
						{/if}
					</td>
					<td class="text-right">
						{#if c.track_clicks}
							{c.clicked}
							<span class="block text-xs text-text-muted">{percent(c.click_rate)}</span>
						{:else}
							<span class="text-text-muted">Off</span>
						{/if}
					</td>
					<td class="text-right" class:text-error={c.bounce_rate > BOUNCE_LIMIT}>
						{c.bounced}
						<span class="block text-xs" class:text-text-muted={c.bounce_rate <= BOUNCE_LIMIT}>{percent(c.bounce_rate)}</span>
					</td>
					<td class="text-right" class:text-error={c.spam_blocks > 0}>{c.spam_blocks}</td>
					<td class="text-right">{c.opted_out}</td>
					<td class="text-right">{c.drafts_waiting}</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
{#if anyOpens}
	<p class="form-hint">
		Apple Mail opens every email automatically, so open rates run high. Reply rate is the measure that counts.
	</p>
{/if}
