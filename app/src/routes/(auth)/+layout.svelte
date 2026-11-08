<script lang="ts">
	import { BrandMark } from '$lib/components/ui';

	let { children } = $props();
</script>

<div class="auth-shell">
	<main class="auth-container">
		<BrandMark />
		{@render children()}
	</main>
</div>
