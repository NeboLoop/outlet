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

<header class="auth-header">
	<h1 class="auth-title">Email Verification</h1>
</header>

<Card class="auth-card">
	{#if verifying}
		<div class="auth-status">
			<div class="auth-status-icon">
				<Spinner size={32} class="text-primary" />
			</div>
			<div>
				<h2 class="auth-status-title">Verifying your email...</h2>
				<p class="auth-status-text">
					Please wait while we verify your email address.
				</p>
			</div>
		</div>
	{:else if verified}
		<div class="auth-status">
			<div class="auth-status-icon bg-success/10">
				<CheckCircle class="h-6 w-6 text-success" />
			</div>
			<div>
				<h2 class="auth-status-title">Email verified successfully</h2>
				<p class="auth-status-text">
					Your email address has been verified. You can now access all features of your Outlet account.
				</p>
			</div>
			<div class="pt-2">
				<Button
					href="/auth/login"
					type="primary"
					size="lg"
					class="w-full justify-center"
				>
					Continue to login
				</Button>
			</div>
		</div>
	{:else if error}
		<div class="auth-status">
			<div class="auth-status-icon bg-error/10">
				<XCircle class="h-6 w-6 text-error" />
			</div>
			<div>
				<h2 class="auth-status-title">Verification failed</h2>
				<p class="auth-status-text">
					{error}
				</p>
			</div>
			<div class="pt-2">
				<a href="/auth/login" class="auth-link">Return to login</a>
			</div>
		</div>
	{/if}
</Card>
