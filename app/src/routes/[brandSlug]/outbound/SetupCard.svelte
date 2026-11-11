<script lang="ts">
	import type { OutboundSetupStep } from '$lib/api';
	import { Button } from '$lib/components/ui';
	import { ArrowRight } from '@lucide/svelte';
	import { SETUP_STEPS, nextSetupStep, setupTitle } from './shared';

	// The "Finish setup" card: the steps done and the one to do next. It
	// shows nothing once setup is done.
	interface Props {
		setup: OutboundSetupStep[];
		base: string; // the brand's Outbound path
		class?: string;
	}

	const { setup, base, class: className = '' }: Props = $props();

	let nextStep = $derived(nextSetupStep(setup));
	let doneCount = $derived(setup.filter((s) => s.done).length);
</script>

{#if nextStep}
	<div class="setup-card {className}">
		<div class="min-w-0">
			<p class="font-medium text-text">Finish setup</p>
			<p class="text-sm text-text-muted">
				{doneCount} of {SETUP_STEPS.length} steps done. Next: {setupTitle(nextStep.step)}{#if nextStep.needs}, which
					needs {nextStep.needs}{/if}.
			</p>
		</div>
		<Button type="primary" size="sm" href="{base}/get-started?step={nextStep.step}">
			Continue setup
			<ArrowRight class="ml-2 h-4 w-4" />
		</Button>
	</div>
{/if}
