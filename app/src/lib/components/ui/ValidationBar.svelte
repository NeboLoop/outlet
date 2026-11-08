<!--
  ValidationBar Component
  Stacked bar of validation status counts for a job.
  Segments are SVG rects sized with width attributes (no inline styles).
-->

<script lang="ts">
	import type { ValidationJobInfo } from '$lib/api';
	import { VALIDATION_STATUSES, percent } from '$lib/validation';

	interface Props {
		job: ValidationJobInfo;
		class?: string;
	}

	let { job, class: extraClass = '' }: Props = $props();

	const total = $derived(
		VALIDATION_STATUSES.reduce((sum, s) => sum + Number(job[s.countKey] || 0), 0)
	);

	const segments = $derived.by(() => {
		let x = 0;
		return VALIDATION_STATUSES.map((s) => {
			const count = Number(job[s.countKey] || 0);
			const width = percent(count, total);
			const seg = { ...s, count, x, width };
			x += width;
			return seg;
		}).filter((s) => s.count > 0);
	});
</script>

<svg class="vstatus-bar {extraClass}" role="img" aria-label="Validation status breakdown" preserveAspectRatio="none">
	{#each segments as seg (seg.key)}
		<rect x="{seg.x}%" y="0" width="{seg.width}%" height="100%" class={seg.barClass}>
			<title>{seg.label}: {seg.count.toLocaleString()} ({seg.width.toFixed(1)}%)</title>
		</rect>
	{/each}
</svg>
