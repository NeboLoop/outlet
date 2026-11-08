<!--
  ValidationBadge Component
  Consistent badge for email validation statuses (ZeroBounce v2):
  valid, invalid, catch-all, unknown, spamtrap, abuse, do_not_mail
-->

<script lang="ts">
	import Badge from './Badge.svelte';
	import Tooltip from './Tooltip.svelte';
	import { validationStatusMeta, subStatusLabel } from '$lib/validation';

	interface Props {
		status: string;
		subStatus?: string;
		/** Show the sub-status as a tooltip on hover */
		showSubStatus?: boolean;
		size?: 'xs' | 'sm' | 'md' | 'lg';
	}

	let { status, subStatus, showSubStatus = true, size = 'sm' }: Props = $props();

	const meta = $derived(validationStatusMeta(status));
	const subLabel = $derived(subStatusLabel(subStatus));
</script>

{#if showSubStatus && subLabel}
	<Tooltip content={subLabel}>
		<Badge variant={meta.variant} outline={meta.outline} {size}>{meta.label}</Badge>
	</Tooltip>
{:else}
	<Badge variant={meta.variant} outline={meta.outline} {size}>{meta.label}</Badge>
{/if}
