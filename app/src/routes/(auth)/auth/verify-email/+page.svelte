<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { verifyEmail } from '$lib/api';
	import { Card, Button, Spinner } from '$lib/components/ui';
	import { CheckCircle, XCircle } from '@lucide/svelte';

	let verifying = $state(true);
	let verified = $state(false);
	let error = $state('');

	onMount(async () => {
		const token = $page.url.searchParams.get('token');

		if (!token) {
			error = 'Invalid verification link';
			verifying = false;
			return;
		}

		try {
			await verifyEmail({ token });
			verified = true;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Verification failed.';
		} finally {
			verifying = false;
		}
	});
</script>

<svelte:head>
	<title>Verify Email - Outlet</title>
</svelte:head>

<div class="space-y-8">
	<div class="text-center">
		<h1 class="text-4xl font-bold text-text">Outlet</h1>
		<h2 class="mt-6 text-2xl font-semibold text-text">Email Verification</h2>
	</div>

	<Card>
		{#if verifying}
			<div class="text-center space-y-4 py-4">
				<div class="mx-auto flex items-center justify-center h-12 w-12">
					<Spinner size={32} class="text-primary" />
				</div>
				<div>
					<h3 class="text-lg font-medium text-text">Verifying your email...</h3>
					<p class="mt-2 text-sm text-text-muted">
						Please wait while we verify your email address.
					</p>
				</div>
			</div>
		{:else if verified}
			<div class="text-center space-y-4 py-4">
				<div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-success/10">
					<CheckCircle class="h-6 w-6 text-success" />
				</div>
				<div>
					<h3 class="text-lg font-medium text-text">Email verified successfully</h3>
					<p class="mt-2 text-sm text-text-muted">
						Your email address has been verified. You can now access all features of your Outlet account.
					</p>
				</div>
				<div class="pt-4">
					<Button
						href="/auth/login"
						type="primary"
						size="lg"
						class="justify-center"
					>
						Continue to login
					</Button>
				</div>
			</div>
		{:else if error}
			<div class="text-center space-y-4 py-4">
				<div class="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-error/10">
					<XCircle class="h-6 w-6 text-error" />
				</div>
				<div>
					<h3 class="text-lg font-medium text-text">Verification failed</h3>
					<p class="mt-2 text-sm text-text-muted">
						{error}
					</p>
				</div>
				<div class="pt-4">
					<a href="/auth/login" class="block text-sm font-medium text-primary hover:underline">
						Return to login
					</a>
				</div>
			</div>
		{/if}
	</Card>
</div>
