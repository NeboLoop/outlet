<!--
  The product mark on the sign-in, account and setup screens
-->

<script lang="ts">
	import { Send } from '@lucide/svelte';

	let { class: extraClass = '' }: { class?: string } = $props();
</script>

<div class="brand-mark {extraClass}">
	<span class="brand-mark-icon" aria-hidden="true">
		<Send class="h-[18px] w-[18px]" />
	</span>
	<span class="brand-mark-name">Outlet</span>
</div>
