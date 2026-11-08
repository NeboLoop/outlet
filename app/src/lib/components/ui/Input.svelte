<!--
  Reusable Input Component
-->

<script lang="ts">
	import { Eye, EyeOff } from '@lucide/svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';

	let {
		type = 'text',
		size = 'md',
		placeholder = '',
		value = $bindable(),
		disabled = false,
		readonly = false,
		required = false,
		id,
		label,
		min,
		max,
		step,
		autocomplete,
		reveal = false,
		class: extraClass = '',
		oninput,
		onblur,
		onfocus,
		onkeydown
	}: {
		type?: string;
		size?: 'xs' | 'sm' | 'md' | 'lg';
		placeholder?: string;
		value?: string | number;
		disabled?: boolean;
		readonly?: boolean;
		required?: boolean;
		id?: string;
		label?: string;
		min?: string | number;
		max?: string | number;
		step?: string | number;
		autocomplete?: HTMLInputAttributes['autocomplete'];
		// A password field gets a button that shows and hides what was typed
		reveal?: boolean;
		class?: string;
		oninput?: (e: Event) => void;
		onblur?: (e: Event) => void;
		onfocus?: (e: Event) => void;
		onkeydown?: (e: KeyboardEvent) => void;
	} = $props();

	// Ensure value is never undefined
	let safeValue = $derived(value ?? '');

	const sizeClasses: Record<string, string> = {
		xs: 'input-xs',
		sm: 'input-sm',
		md: 'input-md',
		lg: 'input-lg'
	};

	const className = $derived(`input input-bordered w-full ${sizeClasses[size]} ${extraClass}`.trim());

	let shown = $state(false);
	const revealable = $derived(reveal && type === 'password');

	function handleInput(e: Event) {
		const inputEl = e.currentTarget as HTMLInputElement;
		if (type === 'number') {
			value = inputEl.valueAsNumber;
		} else {
			value = inputEl.value;
		}
		oninput?.(e);
	}
</script>

{#if label}
	<label class="label" for={id}>
		<span class="label-text">{label}</span>
	</label>
{/if}
{#snippet field(inputClass: string | undefined, inputType: string)}
	<input class={inputClass} type={inputType} {placeholder} value={safeValue} oninput={handleInput} onblur={onblur} onfocus={onfocus} {onkeydown} {disabled} {readonly} {required} {id} {min} {max} {step} {autocomplete} />
{/snippet}

{#if revealable}
	<div class="{className} password-field">
		{@render field(undefined, shown ? 'text' : 'password')}
		<button
			type="button"
			class="password-toggle"
			aria-label={shown ? 'Hide password' : 'Show password'}
			aria-controls={id}
			aria-pressed={shown}
			{disabled}
			onclick={() => (shown = !shown)}
		>
			{#if shown}
				<EyeOff class="h-4 w-4" />
			{:else}
				<Eye class="h-4 w-4" />
			{/if}
		</button>
	</div>
{:else}
	{@render field(className, type)}
{/if}
