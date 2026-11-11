<script lang="ts">
	import type { Snippet } from 'svelte';
	import { getOutboundProfile, updateOutboundProfile, type OutboundProfile } from '$lib/api';
	import { Button, Alert, LoadingSpinner, Input, Textarea } from '$lib/components/ui';
	import { errorMessage } from './shared';

	// The brand profile: what the business sells, to whom, its proof and how
	// its emails sound. Outreach is written from it. assist, when given, sits
	// under the website field; onsaved hears each save.

	let {
		profile = $bindable({}),
		assist,
		onsaved
	}: { profile?: OutboundProfile; assist?: Snippet; onsaved?: (p: OutboundProfile) => void } = $props();

	let loading = $state(true);
	let saving = $state(false);
	let error = $state('');
	let notice = $state('');

	$effect(() => {
		load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			profile = await getOutboundProfile();
		} catch (err) {
			error = errorMessage(err, 'Could not load the profile');
		} finally {
			loading = false;
		}
	}

	async function save() {
		saving = true;
		error = '';
		notice = '';
		try {
			profile = await updateOutboundProfile({
				website: profile.website?.trim(),
				sells: profile.sells?.trim(),
				audience: profile.audience?.trim(),
				proof: profile.proof?.trim(),
				voice: profile.voice?.trim()
			});
			notice = 'Profile saved.';
			onsaved?.(profile);
		} catch (err) {
			error = errorMessage(err, 'Could not save the profile');
		} finally {
			saving = false;
		}
	}
</script>

{#if error}
	<Alert type="error" title="Error" class="mb-4">
		<p>{error}</p>
	</Alert>
{/if}
{#if notice}
	<Alert type="success" dismissible onclose={() => (notice = '')} class="mb-4">
		<p>{notice}</p>
	</Alert>
{/if}

{#if loading}
	<div class="flex justify-center py-6">
		<LoadingSpinner />
	</div>
{:else}
	<div class="space-y-4">
		<div>
			<label for="tp-website" class="form-label">Website</label>
			<Input id="tp-website" bind:value={profile.website} placeholder="https://example.com" />
		</div>
		{@render assist?.()}
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
			<div>
				<label for="tp-sells" class="form-label">What it sells</label>
				<Textarea id="tp-sells" bind:value={profile.sells} rows={3} />
			</div>
			<div>
				<label for="tp-audience" class="form-label">Who buys it</label>
				<Textarea id="tp-audience" bind:value={profile.audience} rows={3} />
			</div>
			<div>
				<label for="tp-proof" class="form-label">Proof</label>
				<Textarea id="tp-proof" bind:value={profile.proof} rows={3} placeholder="Customers, results, numbers" />
			</div>
			<div>
				<label for="tp-voice" class="form-label">How its emails sound</label>
				<Textarea id="tp-voice" bind:value={profile.voice} rows={3} placeholder="Plain, friendly, no hype" />
			</div>
		</div>
		<div class="flex justify-end">
			<Button type="primary" onclick={save} disabled={saving}>
				{saving ? 'Saving...' : 'Save profile'}
			</Button>
		</div>
	</div>
{/if}
